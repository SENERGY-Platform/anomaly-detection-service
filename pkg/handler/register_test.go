/*
 * Copyright 2026 InfAI (CC SES)
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
	"reflect"
	"testing"
)

type noopHandler struct{}

func (this noopHandler) Handle(_ Context, _ []interface{}) (anomaly bool, description string, err error) {
	return false, "", nil
}

// TestEntryGetAspects covers the alias: whatever a registration named, the aspects a reader
// of the entry gets are the same list.
func TestEntryGetAspects(t *testing.T) {
	tests := []struct {
		name     string
		entry    Entry
		expected []string
	}{
		{
			name:     "deprecated single aspect",
			entry:    Entry{Aspect: "urn:infai:ses:aspect:a"},
			expected: []string{"urn:infai:ses:aspect:a"},
		},
		{
			name:     "list with one element",
			entry:    Entry{Aspects: []string{"urn:infai:ses:aspect:a"}},
			expected: []string{"urn:infai:ses:aspect:a"},
		},
		{
			name:     "list with two elements",
			entry:    Entry{Aspects: []string{"urn:infai:ses:aspect:a", "urn:infai:ses:aspect:b"}},
			expected: []string{"urn:infai:ses:aspect:a", "urn:infai:ses:aspect:b"},
		},
		{
			name:     "no aspect at all is no filter",
			entry:    Entry{},
			expected: nil,
		},
		{
			name: "deprecated field is appended to the list",
			entry: Entry{
				Aspect:  "urn:infai:ses:aspect:b",
				Aspects: []string{"urn:infai:ses:aspect:a"},
			},
			expected: []string{"urn:infai:ses:aspect:a", "urn:infai:ses:aspect:b"},
		},
		{
			name: "deprecated field already in the list is not duplicated",
			entry: Entry{
				Aspect:  "urn:infai:ses:aspect:a",
				Aspects: []string{"urn:infai:ses:aspect:a", "urn:infai:ses:aspect:b"},
			},
			expected: []string{"urn:infai:ses:aspect:a", "urn:infai:ses:aspect:b"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := test.entry.GetAspects()
			if !reflect.DeepEqual(actual, test.expected) {
				t.Errorf("unexpected aspects\ne=%#v\na=%#v\n", test.expected, actual)
			}
		})
	}
}

// TestRegisterIsAliasForSingleElementList is the compatibility guarantee: a handler
// registered the old way and one registered with a one element list are indistinguishable
// to everything that reads the register.
func TestRegisterIsAliasForSingleElementList(t *testing.T) {
	aspectId := "urn:infai:ses:aspect:a"
	functionId := "urn:infai:ses:measuring-function:f"
	characteristicId := "urn:infai:ses:characteristic:c"

	reg := NewRegister()
	reg.Register("deprecated", functionId, aspectId, characteristicId, 5, noopHandler{})
	reg.RegisterWithAspects("list", functionId, []string{aspectId}, characteristicId, 5, noopHandler{})

	entries := map[string]Entry{}
	for _, entry := range reg.List() {
		entries[entry.Name] = entry
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %v", len(entries))
		return
	}

	expected := []string{aspectId}
	for name, entry := range entries {
		actual := entry.GetAspects()
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("unexpected aspects for %v\ne=%#v\na=%#v\n", name, expected, actual)
		}
		if entry.Function != functionId {
			t.Errorf("unexpected function for %v: %v", name, entry.Function)
		}
		if entry.Characteristic != characteristicId {
			t.Errorf("unexpected characteristic for %v: %v", name, entry.Characteristic)
		}
		if entry.BufferSize != 5 {
			t.Errorf("unexpected buffer size for %v: %v", name, entry.BufferSize)
		}
	}
}

func TestRegisterWithAspectsStoresTheList(t *testing.T) {
	aspectIds := []string{"urn:infai:ses:aspect:a", "urn:infai:ses:aspect:b"}

	reg := NewRegister()
	reg.RegisterWithAspects("test", "urn:infai:ses:measuring-function:f", aspectIds, "urn:infai:ses:characteristic:c", 2, noopHandler{})

	list := reg.List()
	if len(list) != 1 {
		t.Errorf("expected 1 entry, got %v", len(list))
		return
	}
	if !reflect.DeepEqual(list[0].GetAspects(), aspectIds) {
		t.Errorf("unexpected aspects\ne=%#v\na=%#v\n", aspectIds, list[0].GetAspects())
	}
}

// TestRegisterWithAspectsIgnoresEmptyBuffer keeps the skip rule of the deprecated Register on
// the new one: a handler that never gets values is not stored.
func TestRegisterWithAspectsIgnoresEmptyBuffer(t *testing.T) {
	reg := NewRegister()
	reg.RegisterWithAspects("test", "urn:infai:ses:measuring-function:f", []string{"urn:infai:ses:aspect:a"}, "urn:infai:ses:characteristic:c", 0, noopHandler{})
	if len(reg.List()) != 0 {
		t.Errorf("expected no entry, got %#v", reg.List())
	}
}
