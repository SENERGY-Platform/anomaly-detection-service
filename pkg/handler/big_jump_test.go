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
	"testing"
)

func TestBigJumpHandler_Handle(t *testing.T) {
	type args struct {
		values []interface{}
	}
	tests := []struct {
		name            string
		args            args
		wantAnomaly     bool
		wantDescription string
		wantErr         bool
	}{
		{
			name: "close_big_jump",
			args: args{
				values: []interface{}{1.0, 3.6},
			},
			wantAnomaly:     true,
			wantDescription: "Meter reading had big jump.",
			wantErr:         false,
		},
		{
			name: "close_no_big_jump",
			args: args{
				values: []interface{}{1.0, 3.4},
			},
			wantAnomaly:     false,
			wantDescription: "",
			wantErr:         false,
		},
		{
			name: "big_jump",
			args: args{
				values: []interface{}{1.0, 5.0},
			},
			wantAnomaly:     true,
			wantDescription: "Meter reading had big jump.",
			wantErr:         false,
		},
	}
	store := &TestStore{}
	store.Set("handlerstore_big_jump_test-device_test-service_mean", 2.0)
	store.Set("handlerstore_big_jump_test-device_test-service_stddev", 0.1)
	store.Set("handlerstore_big_jump_test-device_test-service_num_datepoints", 4.0)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := BigJumpHandler{}
			gotAnomaly, gotDescription, err := this.Handle(Context{DeviceId: "test-device", ServiceId: "test-service", Store: store}, tt.args.values)
			if (err != nil) != tt.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotAnomaly != tt.wantAnomaly {
				t.Errorf("Handle() gotAnomaly = %v, want %v", gotAnomaly, tt.wantAnomaly)
			}
			if gotDescription != tt.wantDescription {
				t.Errorf("Handle() gotDescription = %v, want %v", gotDescription, tt.wantDescription)
			}
		})
	}
}
