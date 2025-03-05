//"go:build exclude"

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

// this file/example is ignored by the compiler because of the "//go:build exclude" at the top

func init() {
	//the example handler will be ignored because bufferSize is 0
	Registry.Register("big_jump_anom", "urn:infai:ses:measuring-function:57dfd369-92db-462c-aca4-a767b52c972e", "urn:infai:ses:aspect:412a48ad-3a80-46f7-8b99-408c4b9c3528", "urn:infai:ses:characteristic:3febed55-ba9b-43dc-8709-9c73bae3716e", 3, BigJumpHandler{})
}

type BigJumpHandler struct{}

func (this BigJumpHandler) Handle(values []interface{}) (anomaly bool, description string, err error) {
	castValues, err := CastList[float64](values)
	if err != nil {
		return false, "", err
	}
	log.Println("Values:", castValues)

	const valueEqualityThreshold = 1e-1
	var diffStartingTwo = castValues[1] - castValues[0]
	var diffEndingTwo = castValues[2] - castValues[1]

	if (diffStartingTwo > valueEqualityThreshold) && (diffEndingTwo > 100*diffStartingTwo) {
		return true, "Meter reading had big jump.", nil
	}
	return false, "", nil
}
