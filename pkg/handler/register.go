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
	"github.com/SENERGY-Platform/marshaller/lib/marshaller/model"
)

var Registry = NewRegister()

type Entry struct {
	Name     string
	Function string
	// Aspect is deprecated: use Aspects. It is kept as an alias for an Aspects list with
	// one element, so a registration that names a single aspect keeps its old meaning.
	Aspect         string
	Aspects        []string
	Characteristic string
	BufferSize     int
	Handler        Handler
}

// GetAspects returns the aspects of the entry, with the deprecated single Aspect folded in.
// Every reader of an Entry has to use it rather than the fields: Register stores what it was
// given, so an entry from the deprecated Register carries the single field only.
func (this Entry) GetAspects() []string {
	return model.AspectIdsAlias(this.Aspect, this.Aspects)
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
// Deprecated: use RegisterWithAspects. A single aspect is an alias for an aspect list with
// one element and keeps behaving exactly as it did.
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

// RegisterWithAspects stores the Handler for use by the anomaly-detection-service
//
//	if bufferSize is 0, the handler will not be stored.
//	the bufferSize determines how many values ar send to the handler.
//	the handler will not be called, if the bufferSize is larger than the count of available values
//	the handler will only be called for devices/services with matching functions and aspects (aspect-hierarchy is observed)
//	if no device/service with matching function and aspect is found, the handler will never be called
//	the characteristic determines to what characteristic the incoming values are converted
//
// Several aspects are an AND on one content variable, not a choice between them: the handler
// is called for a value that carries all of them, each covering its own aspect subtree. Two
// aspects of the same hierarchy therefore match nothing, because a content variable carries
// at most one aspect per aspect class.
func (this *Register) RegisterWithAspects(name string, function string, aspects []string, characteristic string, bufferSize int, handler Handler) {
	if bufferSize == 0 {
		return
	}
	this.entries[name] = Entry{
		Name:           name,
		Function:       function,
		Aspects:        aspects,
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
