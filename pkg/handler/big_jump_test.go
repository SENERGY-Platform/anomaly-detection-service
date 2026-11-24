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
	"encoding/json"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// jsonStore keeps values as JSON, like the valkey store of the service, so the tests also cover
// that the handler state survives the round trip.
type jsonStore struct {
	mux    sync.Mutex
	values map[string][]byte
}

func (this *jsonStore) Set(key string, value interface{}) error {
	buff, err := json.Marshal(value)
	if err != nil {
		return err
	}
	this.mux.Lock()
	defer this.mux.Unlock()
	if this.values == nil {
		this.values = map[string][]byte{}
	}
	this.values[key] = buff
	return nil
}

func (this *jsonStore) Get(key string, value interface{}) (exists bool, err error) {
	this.mux.Lock()
	defer this.mux.Unlock()
	buff, ok := this.values[key]
	if !ok {
		return false, nil
	}
	return true, json.Unmarshal(buff, value)
}

type reading struct {
	at    time.Time
	value float64
}

var testStart = time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)

// quietShare is the share of reports that come only after Tasmota's TelePeriod of 300 s.
// A Tasmota plug that raised false alarms mostly reported every 20 to 60 s. With 1 %, the
// non-zero increments of a steady 40 W plug have a mean of 1.04 Wh and a standard deviation of
// 0.31 Wh, close to the 1.12 Wh and 0.37 Wh measured on that plug.
const quietShare = 0.01

// meterReadings simulates a meter like that plug, starting after from.
// Every one to four hours the load changes to one of powers (in watts). The counter is reported
// in kWh rounded down to 0.001, mostly every 20 to 60 s and otherwise after 300 s.
func meterReadings(seed int64, from reading, duration time.Duration, powers ...float64) (result []reading) {
	rnd := rand.New(rand.NewSource(seed))
	energy := from.value
	at := from.at
	end := from.at.Add(duration)
	segmentEnd := at
	power := 0.0
	for at.Before(end) {
		if !at.Before(segmentEnd) {
			power = powers[rnd.Intn(len(powers))]
			segmentEnd = at.Add(time.Hour + time.Duration(rnd.Int63n(int64(3*time.Hour))))
		}
		gap := 300 * time.Second
		if rnd.Float64() >= quietShare {
			gap = 20*time.Second + time.Duration(rnd.Int63n(int64(40*time.Second)))
		}
		at = at.Add(gap)
		energy += power / 1000 * gap.Hours()
		result = append(result, reading{at: at, value: math.Floor(energy*1000) / 1000})
	}
	return result
}

// after returns the reading that follows r after d with the counter increased by increment.
func after(r reading, d time.Duration, increment float64) reading {
	return reading{at: r.at.Add(d), value: math.Round((r.value+increment)*1000) / 1000}
}

// feed hands the readings to the handler one by one, as the controller does with a buffer of
// two values, and returns the readings reported as a big jump.
func feed(t *testing.T, store Store, previous reading, readings ...reading) (alarms []reading) {
	t.Helper()
	handler := BigJumpHandler{}
	for _, current := range readings {
		context := Context{DeviceId: "test-device", ServiceId: "test-service", Store: store, Time: current.at}
		anomaly, _, err := handler.Handle(context, []interface{}{previous.value, current.value})
		if err != nil {
			t.Fatal(err)
		}
		if anomaly {
			alarms = append(alarms, current)
		}
		previous = current
	}
	return alarms
}

// learnedPlug returns a store that has learned two days of a simulated plug with the given loads
// (in watts), which is well past the learning phase, and the last reading.
func learnedPlug(t *testing.T, powers ...float64) (*jsonStore, reading) {
	t.Helper()
	store := &jsonStore{}
	start := reading{at: testStart, value: 33.0}
	readings := meterReadings(1, start, 48*time.Hour, powers...)
	if alarms := feed(t, store, start, readings...); len(alarms) > 0 {
		t.Fatalf("learning the plug raised %v alarms, the first at %v", len(alarms), alarms[0])
	}
	return store, readings[len(readings)-1]
}

func TestBigJumpHandler_PlugWithIrregularReportsRaisesNoAlarmForAWeek(t *testing.T) {
	tests := []struct {
		name   string
		powers []float64
	}{
		{name: "steady 40 W", powers: []float64{40}},
		{name: "steady 95 W", powers: []float64{95}},
		{name: "changing between 40 and 95 W", powers: []float64{40, 40, 95}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start := reading{at: testStart, value: 33.0}
			readings := meterReadings(2, start, 7*24*time.Hour, test.powers...)
			if alarms := feed(t, &jsonStore{}, start, readings...); len(alarms) > 0 {
				t.Errorf("%v false alarms, the first at %v", len(alarms), alarms[0])
			}
		})
	}
}

func TestBigJumpHandler_PauseAtSteadyLoadIsNoBigJump(t *testing.T) {
	tests := []struct {
		name      string
		power     float64
		pause     time.Duration
		increment float64
	}{
		// the first two were reported as big jumps on plugs at steady load
		{name: "3 Wh in 276 s at 40 W", power: 40, pause: 276 * time.Second, increment: 0.003},
		{name: "4 Wh in 153 s at 95 W", power: 95, pause: 153 * time.Second, increment: 0.004},
		{name: "8 Wh in 300 s at 95 W", power: 95, pause: 300 * time.Second, increment: 0.008},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, last := learnedPlug(t, test.power)
			if alarms := feed(t, store, last, after(last, test.pause, test.increment)); len(alarms) > 0 {
				t.Errorf("reported as big jump: %v", alarms[0])
			}
		})
	}
}

func TestBigJumpHandler_ReportsBigJump(t *testing.T) {
	tests := []struct {
		name      string
		pause     time.Duration
		increment float64
	}{
		{name: "500 Wh in 30 s", pause: 30 * time.Second, increment: 0.5},
		{name: "50 Wh in 60 s", pause: 60 * time.Second, increment: 0.05},
		{name: "50 Wh in 300 s", pause: 300 * time.Second, increment: 0.05},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, last := learnedPlug(t, 40, 40, 95)
			if alarms := feed(t, store, last, after(last, test.pause, test.increment)); len(alarms) != 1 {
				t.Errorf("not reported as big jump")
			}
		})
	}
}

func TestBigJumpHandler_NoAlarmDuringLearningPhase(t *testing.T) {
	store := &jsonStore{}
	start := reading{at: testStart, value: 33.0}
	readings := meterReadings(3, start, 5*time.Hour, 40)
	readings = append(readings, after(readings[len(readings)-1], 30*time.Second, 0.5))
	if alarms := feed(t, store, start, readings...); len(alarms) > 0 {
		t.Errorf("reported %v during the learning phase", alarms[0])
	}
}

func TestBigJumpHandler_ResetRaisesNoAlarmAndDetectionContinues(t *testing.T) {
	store, last := learnedPlug(t, 40, 40, 95)
	reset := reading{at: last.at.Add(40 * time.Second), value: 0}
	readings := append([]reading{reset}, meterReadings(4, reset, 24*time.Hour, 40, 40, 95)...)
	if alarms := feed(t, store, last, readings...); len(alarms) > 0 {
		t.Fatalf("reported %v after a reset", alarms[0])
	}
	last = readings[len(readings)-1]
	if alarms := feed(t, store, last, after(last, 60*time.Second, 0.05)); len(alarms) != 1 {
		t.Errorf("50 Wh in 60 s after a reset not reported as big jump")
	}
}

func TestBigJumpHandler_ValuesProcessedOutOfOrderRaiseNoAlarm(t *testing.T) {
	// observed on a plug: 7.445 arrived at .444 and 7.446 at .488, but were processed swapped
	store, last := learnedPlug(t, 40, 40, 95)
	secondArrived := after(last, 40*time.Second, 0.002)
	firstArrived := reading{at: secondArrived.at.Add(-44 * time.Millisecond), value: secondArrived.value - 0.001}
	next := after(secondArrived, 40*time.Second, 0.001)
	if alarms := feed(t, store, last, secondArrived, firstArrived, next); len(alarms) > 0 {
		t.Errorf("reported %v", alarms[0])
	}
}

func TestBigJumpHandler_FloatNoiseDoesNotShrinkTheRoundingAllowance(t *testing.T) {
	store, last := learnedPlug(t, 40, 40, 95)
	noisy := reading{at: last.at.Add(time.Second), value: last.value + 1e-12}
	// two resolution steps arriving at once are covered by the rounding allowance alone
	bundled := reading{at: noisy.at, value: last.value + 0.002}
	if alarms := feed(t, store, last, noisy, bundled); len(alarms) > 0 {
		t.Errorf("reported %v", alarms[0])
	}
}

func TestBigJumpHandler_ForgetsWindowsOlderThanSevenDays(t *testing.T) {
	store := &jsonStore{}
	start := reading{at: testStart, value: 33.0}
	readings := meterReadings(5, start, 24*time.Hour, 2000)
	readings = append(readings, meterReadings(6, readings[len(readings)-1], 8*24*time.Hour, 40, 40, 95)...)
	feed(t, store, start, readings...)
	// plausible at 2000 W, but not at 40 or 95 W
	last := readings[len(readings)-1]
	if alarms := feed(t, store, last, after(last, 60*time.Second, 0.02)); len(alarms) != 1 {
		t.Errorf("20 Wh in 60 s not reported as big jump, the 2000 W day is still learned")
	}
}
