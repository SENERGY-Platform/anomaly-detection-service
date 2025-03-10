//go:build exclude

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
	Registry.Register("example", "function-id", "aspect-id", "characteristic-id", 0, ExampleHandler{})
}

type ExampleHandler struct{}

func (this ExampleHandler) Handle(values []interface{}) (anomaly bool, description string, err error) {
	castValues, err := CastList[float64](values)
	if err != nil {
		return false, "", err
	}
	log.Println("example", castValues)
	return false, "", nil
}
