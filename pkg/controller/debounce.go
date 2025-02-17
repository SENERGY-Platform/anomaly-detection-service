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

package controller

import (
	"sync"
	"time"
)

type Debounce struct {
	Duration time.Duration
	mux      sync.Mutex
	after    *time.Timer
}

func (this *Debounce) Do(f func()) {
	this.mux.Lock()
	defer this.mux.Unlock()
	if this.after == nil || !this.after.Reset(this.Duration) {
		this.after = time.AfterFunc(this.Duration, f)
	}
}
