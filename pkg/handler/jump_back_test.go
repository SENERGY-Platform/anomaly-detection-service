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
				values: []interface{}{1253.8, 1253.8, 1253.8},
			},
			wantAnomaly:     false,
			wantDescription: "",
			wantErr:         false,
		},
		{
			name: "increasing_meter_reading",
			args: args{
				values: []interface{}{1253.8, 1253.9, 1254.0},
			},
			wantAnomaly:     false,
			wantDescription: "",
			wantErr:         false,
		},
		{
			name: "drop_confirmed_by_next_value_is_reported",
			args: args{
				values: []interface{}{1253.8, 1253.7, 1253.75},
			},
			wantAnomaly:     true,
			wantDescription: "Meter reading jumped back.",
			wantErr:         false,
		},
		{
			name: "reset_to_zero_is_reported",
			args: args{
				values: []interface{}{33.498, 0.0, 0.001},
			},
			wantAnomaly:     true,
			wantDescription: "Meter reading jumped back.",
			wantErr:         false,
		},
		{
			// observed on a plug: 7.445 and 7.446 were processed in swapped order
			name: "value_delivered_out_of_order_is_not_reported",
			args: args{
				values: []interface{}{7.446, 7.445, 7.447},
			},
			wantAnomaly:     false,
			wantDescription: "",
			wantErr:         false,
		},
		{
			name: "drop_followed_by_previous_level_is_not_reported",
			args: args{
				values: []interface{}{7.446, 7.445, 7.446},
			},
			wantAnomaly:     false,
			wantDescription: "",
			wantErr:         false,
		},
		{
			name: "drop_in_newest_value_waits_for_the_next_value",
			args: args{
				values: []interface{}{7.444, 7.446, 7.445},
			},
			wantAnomaly:     false,
			wantDescription: "",
			wantErr:         false,
		},
	}
	store := &TestStore{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			this := JumpBackHandler{}
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
