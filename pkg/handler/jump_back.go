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
	Registry.Register("jump_back_anom", "urn:infai:ses:measuring-function:57dfd369-92db-462c-aca4-a767b52c972e", "urn:infai:ses:aspect:fdc999eb-d366-44e8-9d24-bfd48d5fece1", "urn:infai:ses:characteristic:3febed55-ba9b-43dc-8709-9c73bae3716e", 2, JumpBackHandler{})
	/* Get Electricity Consumption, Electricity-->Total Subaspect, kWh */
	Registry.Register("jump_back_anom", "urn:infai:ses:measuring-function:57dfd369-92db-462c-aca4-a767b52c972e", "urn:infai:ses:aspect:fdc999eb-d366-44e8-9d24-bfd48d5fece1", "urn:infai:ses:characteristic:d4ac88cf-f10b-45d5-a3a9-e42b4b2a55ca", 2, JumpBackHandler{})
	/* Get Electricity Consumption, Electricity-->Total Subaspect, Wh */
	Registry.Register("jump_back_anom", "urn:infai:ses:measuring-function:57dfd369-92db-462c-aca4-a767b52c972e", "urn:infai:ses:aspect:fdc999eb-d366-44e8-9d24-bfd48d5fece1", "urn:infai:ses:characteristic:00413fba-f7e9-447c-8476-1d236db9ec53", 2, JumpBackHandler{})
	/* Get Electricity Consumption, Electricity-->Total Subaspect, Wmin */

	Registry.Register("jump_back_anom", "urn:infai:ses:measuring-function:cfa56e75-8e8f-4f0d-a3fa-ed2758422b2a", "urn:infai:ses:aspect:b8b3b549-3b01-4604-a727-20aa528c21c9", "urn:infai:ses:characteristic:aeb260f8-5fe5-4989-9e66-3c0a4ff273c4", 2, JumpBackHandler{})
	/* Get Volume, Water, Liter */
	Registry.Register("jump_back_anom", "urn:infai:ses:measuring-function:cfa56e75-8e8f-4f0d-a3fa-ed2758422b2a", "urn:infai:ses:aspect:b8b3b549-3b01-4604-a727-20aa528c21c9", "urn:infai:ses:characteristic:fbfea6c7-3392-4ec2-9ec5-0d0ef6362b9b", 2, JumpBackHandler{})
	/* Get Volume, Water, Cubic Meter*/

	Registry.Register("jump_back_anom", "urn:infai:ses:measuring-function:4daa591f-ad97-4e57-8014-aa3f5e552c3b", "urn:infai:ses:aspect:7ea324c1-48e4-419a-a499-325d79dac09f", "urn:infai:ses:characteristic:aeb260f8-5fe5-4989-9e66-3c0a4ff273c4", 2, JumpBackHandler{})
	/* Get Gas Consumption, Gas, Liter*/
	Registry.Register("jump_back_anom", "urn:infai:ses:measuring-function:4daa591f-ad97-4e57-8014-aa3f5e552c3b", "urn:infai:ses:aspect:7ea324c1-48e4-419a-a499-325d79dac09f", "urn:infai:ses:characteristic:fbfea6c7-3392-4ec2-9ec5-0d0ef6362b9b", 2, JumpBackHandler{})
	/* Get Gas Consumption, Gas, Cubic Meter*/
}

type JumpBackHandler struct{}

func (this JumpBackHandler) Handle(values []interface{}) (anomaly bool, description string, err error) {
	castValues, err := CastList[float64](values)
	if err != nil {
		return false, "", err
	}
	log.Println("Values:", castValues)
	if castValues[1] < castValues[0] {
		return true, "Meter reading jumped back.", nil
	}
	return false, "", nil
}
