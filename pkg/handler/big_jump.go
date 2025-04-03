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
	"log"
	"math"
)

func init() {
	/* Get Electricity Consumption, Electricity-->Total Subaspect, kWh */
	Registry.Register("big_jump_anom_electricity_consumption_total_kwh", "urn:infai:ses:measuring-function:57dfd369-92db-462c-aca4-a767b52c972e", "urn:infai:ses:aspect:fdc999eb-d366-44e8-9d24-bfd48d5fece1", "urn:infai:ses:characteristic:3febed55-ba9b-43dc-8709-9c73bae3716e", 2, BigJumpHandler{})

	/* Get Volume, Water, Liter */
	Registry.Register("big_jump_anom_volume_water_liter", "urn:infai:ses:measuring-function:cfa56e75-8e8f-4f0d-a3fa-ed2758422b2a", "urn:infai:ses:aspect:b8b3b549-3b01-4604-a727-20aa528c21c9", "urn:infai:ses:characteristic:aeb260f8-5fe5-4989-9e66-3c0a4ff273c4", 2, BigJumpHandler{})

	/* Get Gas Consumption, Gas, Liter*/
	Registry.Register("big_jump_anom_consumption_gas_liter", "urn:infai:ses:measuring-function:4daa591f-ad97-4e57-8014-aa3f5e552c3b", "urn:infai:ses:aspect:7ea324c1-48e4-419a-a499-325d79dac09f", "urn:infai:ses:characteristic:aeb260f8-5fe5-4989-9e66-3c0a4ff273c4", 2, BigJumpHandler{})
}

type BigJumpHandler struct{}

func (this BigJumpHandler) Handle(context Context, values []interface{}) (anomaly bool, description string, err error) {
	castValues, err := CastList[float64](values)
	if err != nil {
		return false, "", err
	}
	log.Println("Values:", castValues)

	/* Std deviation of differences between consecutive meter values*/
	var CurrentStddev float64
	err = context.Store.Get(context.PrepareKey("big_jump", "stddev"), &CurrentStddev)
	if err != nil {
		CurrentStddev = 0
	}

	/* Mean of differences between consecutive meter values*/
	var CurrentMean float64
	err = context.Store.Get(context.PrepareKey("big_jump", "mean"), &CurrentMean)
	if err != nil {
		CurrentMean = 0
	}

	/* Number of tracked differences up to now*/
	var NumDatepoints float64
	err = context.Store.Get(context.PrepareKey("big_jump", "num_datepoints"), &NumDatepoints)
	if err != nil {
		NumDatepoints = 0.0
	}

	latestDifference := castValues[1] - castValues[0]

	var bigJump bool = latestDifference > CurrentMean+5*CurrentStddev

	CurrentStddev = UpdateStddev(latestDifference, CurrentStddev, CurrentMean, NumDatepoints)
	context.Store.Set(context.PrepareKey("big_jump", "stddev"), CurrentStddev)

	CurrentMean = UpdateMean(latestDifference, CurrentMean, NumDatepoints)
	context.Store.Set(context.PrepareKey("big_jump", "mean"), CurrentMean)

	NumDatepoints = NumDatepoints + 1
	context.Store.Set(context.PrepareKey("big_jump", "num_datepoints"), NumDatepoints)

	if bigJump {
		log.Println("Meter reading had big jump.")
		return true, "Meter reading had big jump.", nil
	}
	return false, "", nil
}

/*Sample Update of standard deviation*/
func UpdateStddev(latestValue float64, CurrentStddev float64, CurrentMean float64, NumDatepoints float64) float64 {
	return math.Sqrt(NumDatepoints/(NumDatepoints+1)*math.Pow(CurrentStddev, 2) + NumDatepoints/math.Pow(NumDatepoints+1, 2)*math.Pow(latestValue-CurrentMean, 2))
}

/*Sample Update of mean*/
func UpdateMean(latestValue float64, CurrentMean float64, NumDatepoints float64) float64 {
	return (NumDatepoints*CurrentMean + latestValue) / (NumDatepoints + 1)
}
