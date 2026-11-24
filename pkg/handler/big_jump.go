/*
 * Copyright 2025 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package handler

import (
	"log/slog"
	"math"
	"slices"
	"time"
)

func init() {
	/* Get Energy, Total, kWh */
	Registry.Register("big_jump_anom_electricity_consumption_total_kwh", "urn:infai:ses:measuring-function:474471c3-e401-4b54-8963-53739a3b1de4", "urn:infai:ses:aspect:fdc999eb-d366-44e8-9d24-bfd48d5fece1", "urn:infai:ses:characteristic:3febed55-ba9b-43dc-8709-9c73bae3716e", 2, BigJumpHandler{})

	/* Get Volume, Water, Liter */
	Registry.Register("big_jump_anom_volume_water_liter", "urn:infai:ses:measuring-function:f5344662-63d2-40b7-9b9e-0b28a2d34252", "urn:infai:ses:aspect:b8b3b549-3b01-4604-a727-20aa528c21c9", "urn:infai:ses:characteristic:aeb260f8-5fe5-4989-9e66-3c0a4ff273c4", 2, BigJumpHandler{})

	/* Get Volume, Gas, Liter */
	Registry.Register("big_jump_anom_consumption_gas_liter", "urn:infai:ses:measuring-function:f5344662-63d2-40b7-9b9e-0b28a2d34252", "urn:infai:ses:aspect:7ea324c1-48e4-419a-a499-325d79dac09f", "urn:infai:ses:characteristic:aeb260f8-5fe5-4989-9e66-3c0a4ff273c4", 2, BigJumpHandler{})
}

// BigJumpHandler reports a meter reading that rose by more than the meter can plausibly count
// in the time since the previous reading. docs/big-jump-and-jump-back.md derives the test and
// lists the alternatives that were rejected. A reading is reported if
//
//	increment > powerLimit * elapsed + bigJumpAllowanceSteps * resolution
//
// resolution is the smallest increase seen so far, the step the meter rounds to; the allowance
// covers the rounding of both readings. powerLimit is bigJumpPowerFactor times the
// bigJumpQuantile quantile of the mean power ("power" stands for counting speed, of any unit)
// of the windows closed within the last bigJumpHorizon.
// A window starts at a reading and closes with the first reading at least bigJumpWindowSteps
// steps or bigJumpMaxWindow later, so rounding changes its mean power by about
// 1/bigJumpWindowSteps at most. Every window is learned, jumps included: the quantile keeps a
// rare jump from raising the limit, while a lasting change of the load is learned.
// A falling reading (a reset, or readings processed out of order) is neither tested nor learned
// and starts a new window.
//
// Times are those at which the platform received the readings. When readings arrive bundled,
// elapsed is shorter than the time the meter counted, which this test cannot see.
type BigJumpHandler struct{}

const (
	bigJumpWindowSteps    = 10
	bigJumpMaxWindow      = 30 * time.Minute
	bigJumpHorizon        = 7 * 24 * time.Hour
	bigJumpQuantile       = 0.99
	bigJumpPowerFactor    = 1.5
	bigJumpAllowanceSteps = 2
	// bigJumpMinWindows is the learning phase: readings are tested once this many windows
	// within the horizon are known
	bigJumpMinWindows = 100
	// bigJumpMaxWindows bounds the stored windows of a meter whose windows close every few seconds
	bigJumpMaxWindows = 5000
	// bigJumpTolerance is the relative difference below which two readings count as equal,
	// so float noise neither counts as an increase nor shrinks the resolution
	bigJumpTolerance = 1e-9
)

// bigJumpState is read and written for every reading. Times are unix seconds.
type bigJumpState struct {
	PrevTime    float64 `json:"prev_time"`
	PrevValue   float64 `json:"prev_value"`
	WindowTime  float64 `json:"window_time"`
	WindowValue float64 `json:"window_value"`
	Resolution  float64 `json:"resolution"`  // 0 until the first increase
	PowerLimit  float64 `json:"power_limit"` // per second, as of the last closed window
	Windows     int     `json:"windows"`     // windows within the horizon, as of the last closed window
}

// bigJumpWindow is stored for every closed window
type bigJumpWindow struct {
	End   float64 `json:"end"`   // unix seconds
	Power float64 `json:"power"` // mean increase per second
}

func (this BigJumpHandler) Handle(context Context, values []interface{}) (anomaly bool, description string, err error) {
	castValues, err := CastList[float64](values)
	if err != nil {
		return false, "", err
	}
	slog.Debug("BigJumpHandler.Handle", "castValues", castValues)
	// the previous value is taken from the state, where it is stored together with its time
	value := castValues[len(castValues)-1]
	now := float64(context.Time.UnixMilli()) / 1000

	stateKey := context.PrepareKey("big_jump", "state")
	state := bigJumpState{}
	exists, err := context.Store.Get(stateKey, &state)
	if err != nil {
		return false, "", err
	}
	if !exists {
		state = bigJumpState{PrevTime: now, PrevValue: value, WindowTime: now, WindowValue: value}
		return false, "", context.Store.Set(stateKey, state)
	}

	tolerance := bigJumpTolerance * math.Max(1, math.Abs(value))
	increment := value - state.PrevValue
	if math.Abs(increment) <= tolerance {
		increment = 0
	}
	// readings processed out of order can go back in time
	elapsed := math.Max(now-state.PrevTime, 0)
	if increment > 0 && (state.Resolution == 0 || increment < state.Resolution) {
		state.Resolution = increment
	}

	limit := 0.0
	if state.Windows >= bigJumpMinWindows {
		limit = state.PowerLimit*elapsed + bigJumpAllowanceSteps*state.Resolution
		anomaly = increment > limit+tolerance
	}
	usedPowerLimit, usedWindows := state.PowerLimit, state.Windows

	switch {
	case increment < 0:
		state.WindowTime, state.WindowValue = now, value
	case windowCloses(state, now, value, tolerance):
		state.PowerLimit, state.Windows, err = learnWindow(context, now, (value-state.WindowValue)/(now-state.WindowTime))
		if err != nil {
			return false, "", err
		}
		state.WindowTime, state.WindowValue = now, value
	}
	state.PrevTime, state.PrevValue = now, value
	err = context.Store.Set(stateKey, state)
	if err != nil {
		return false, "", err
	}

	if anomaly {
		slog.Info("Meter reading had big jump.", "castValues", castValues, "increment", increment, "elapsedSeconds", elapsed, "limit", limit, "powerLimitPerHour", usedPowerLimit*3600, "resolution", state.Resolution, "windows", usedWindows)
		return true, "Meter reading had big jump.", nil
	}
	return false, "", nil
}

func windowCloses(state bigJumpState, now float64, value float64, tolerance float64) bool {
	duration := now - state.WindowTime
	if duration <= 0 {
		return false
	}
	enoughSteps := state.Resolution > 0 && value-state.WindowValue >= bigJumpWindowSteps*state.Resolution-tolerance
	return enoughSteps || duration >= bigJumpMaxWindow.Seconds()
}

// learnWindow stores the mean power of a closed window, forgets the windows outside the horizon,
// and returns the power limit and the number of windows it rests on
func learnWindow(context Context, end float64, power float64) (powerLimit float64, windows int, err error) {
	key := context.PrepareKey("big_jump", "windows")
	known := []bigJumpWindow{}
	_, err = context.Store.Get(key, &known)
	if err != nil {
		return 0, 0, err
	}
	known = append(known, bigJumpWindow{End: end, Power: power})
	horizonStart := end - bigJumpHorizon.Seconds()
	known = slices.DeleteFunc(known, func(window bigJumpWindow) bool {
		return window.End < horizonStart
	})
	if len(known) > bigJumpMaxWindows {
		known = known[len(known)-bigJumpMaxWindows:]
	}
	err = context.Store.Set(key, known)
	if err != nil {
		return 0, 0, err
	}
	powers := make([]float64, len(known))
	for i, window := range known {
		powers[i] = window.Power
	}
	slices.Sort(powers)
	quantile := powers[min(len(powers)-1, int(bigJumpQuantile*float64(len(powers))))]
	return bigJumpPowerFactor * quantile, len(known), nil
}
