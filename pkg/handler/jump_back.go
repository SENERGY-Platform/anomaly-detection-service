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

import "log"

func init() {
	/* Get Electricity Consumption, Electricity-->Total Subaspect, kWh */
	Registry.Register("jump_back_anom_electricity_consumption_total_kwh", "urn:infai:ses:measuring-function:57dfd369-92db-462c-aca4-a767b52c972e", "urn:infai:ses:aspect:fdc999eb-d366-44e8-9d24-bfd48d5fece1", "urn:infai:ses:characteristic:3febed55-ba9b-43dc-8709-9c73bae3716e", 2, JumpBackHandler{})

	/* Get Volume, Water, Liter */
	Registry.Register("jump_back_anom_volume_water_liter", "urn:infai:ses:measuring-function:cfa56e75-8e8f-4f0d-a3fa-ed2758422b2a", "urn:infai:ses:aspect:b8b3b549-3b01-4604-a727-20aa528c21c9", "urn:infai:ses:characteristic:aeb260f8-5fe5-4989-9e66-3c0a4ff273c4", 2, JumpBackHandler{})

	/* Get Gas Consumption, Gas, Liter*/
	Registry.Register("jump_back_anom_consumption_gas_liter", "urn:infai:ses:measuring-function:4daa591f-ad97-4e57-8014-aa3f5e552c3b", "urn:infai:ses:aspect:7ea324c1-48e4-419a-a499-325d79dac09f", "urn:infai:ses:characteristic:aeb260f8-5fe5-4989-9e66-3c0a4ff273c4", 2, JumpBackHandler{})
}

type JumpBackHandler struct{}

func (this JumpBackHandler) Handle(context Context, values []interface{}) (anomaly bool, description string, err error) {
	castValues, err := CastList[float64](values)
	if err != nil {
		return false, "", err
	}
	log.Println("Values:", castValues)
	if castValues[1] < castValues[0] {
		log.Println("Meter reading jumped back.")
		return true, "Meter reading jumped back.", nil
	}
	return false, "", nil
}
