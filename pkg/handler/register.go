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

var Registry = NewRegister()

type Entry struct {
	Name           string
	Function       string
	Aspect         string
	Characteristic string
	BufferSize     int
	Handler        Handler
}

func (this *Entry) Handle(context Context, values []interface{}) (anomaly bool, description string, err error) {
	return this.Handler.Handle(context, values)
}

type Register struct {
	entries map[string]Entry
}

func NewRegister() *Register {
	return &Register{
		entries: map[string]Entry{},
	}
}

// Register stores the Handler for use by the anomaly-detection-service
//
//	if bufferSize is 0, the handler will not be stored.
//	the bufferSize determines how many values ar send to the handler.
//	the handler will not be called, if the bufferSize is larger than the count of available values
//	the handler will only be called for devices/services with matching functions and aspects (aspect-hierarchy is observed)
//	if no device/service with matching function and aspect is found, the handler will never be called
//	the characteristic determines to what characteristic the incoming values are converted
func (this *Register) Register(name string, function string, aspect string, characteristic string, bufferSize int, handler Handler) {
	if bufferSize == 0 {
		return
	}
	this.entries[name] = Entry{
		Name:           name,
		Function:       function,
		Aspect:         aspect,
		Characteristic: characteristic,
		BufferSize:     bufferSize,
		Handler:        handler,
	}
}

func (this *Register) List() (result []Entry) {
	for _, entry := range this.entries {
		result = append(result, entry)
	}
	return result
}
