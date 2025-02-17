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
)

// CastList
// preferred casts are to string, float64, bool
// others are slower because they need to json.Marshal
func CastList[T any](values []interface{}) (result []T, err error) {
	for _, value := range values {
		e, err := Cast[T](value)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

// Cast
// preferred casts are to string, float64, bool
// others are slower because they need to json.Marshal
func Cast[T any](value interface{}) (result T, err error) {
	result, ok := value.(T)
	if ok {
		return result, nil
	}
	temp, err := json.Marshal(value)
	if err != nil {
		return result, nil
	}
	err = json.Unmarshal(temp, &result)
	if err != nil {
		return result, nil
	}
	return result, nil
}
