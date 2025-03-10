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

import "testing"

func TestJumpBackHandler_Handle(t *testing.T) {
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
			name: "const_meter_reading",
			args: args{
				values: []interface{}{1253.8, 1253.8},
			},
			wantAnomaly:     false,
			wantDescription: "",
			wantErr:         false,
		},
		{
			name: "increasing_meter_reading",
			args: args{
				values: []interface{}{1253.8, 1253.9},
			},
			wantAnomaly:     false,
			wantDescription: "",
			wantErr:         false,
		},
		{
			name: "decreasing_meter_reading",
			args: args{
				values: []interface{}{1253.8, 1253.7},
			},
			wantAnomaly:     true,
			wantDescription: "Meter reading jumped back.",
			wantErr:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := JumpBackHandler{}
			gotAnomaly, gotDescription, err := this.Handle(tt.args.values)
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
