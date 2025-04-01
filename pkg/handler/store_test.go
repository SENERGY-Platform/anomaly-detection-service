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
	"errors"
	"reflect"
	"sync"
	"testing"
)

type TestStore struct {
	values map[string]interface{}
	mux    sync.Mutex
}

func (this *TestStore) Set(key string, value interface{}) (err error) {
	this.mux.Lock()
	defer this.mux.Unlock()
	if this.values == nil {
		this.values = make(map[string]interface{})
	}
	this.values[key] = value
	return nil
}

func (this *TestStore) Get(key string, value interface{}) (err error) {
	this.mux.Lock()
	defer this.mux.Unlock()
	if this.values == nil {
		this.values = make(map[string]interface{})
	}
	val, ok := this.values[key]
	if !ok {
		return errors.New("key not found")
	}
	reflect.Indirect(reflect.ValueOf(value)).Set(reflect.ValueOf(val))
	return nil
}

func TestTestStore(t *testing.T) {
	store := &TestStore{}
	err := store.Set("test", 1.0)
	if err != nil {
		t.Error(err)
		return
	}
	var result float64
	err = store.Get("test", &result)
	if err != nil {
		t.Error(err)
		return
	}
	if result != 1.0 {
		t.Errorf("result should be 1.0, got %v", result)
	}
}
