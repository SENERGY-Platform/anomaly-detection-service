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

type Handler interface {
	// Handle
	//		receives the last n values (oldest first, newest last)
	//		returns
	//			err only for internal errors
	//			anomaly = true if one was found
	//			description with a human-readable description of the anomaly, if one was found
	Handle(context Context, values []interface{}) (anomaly bool, description string, err error)
}

type Context struct {
	DeviceId  string
	ServiceId string
	Store     Store
}

type Store interface {
	Get(key string, value interface{}) error
	Set(key string, value interface{}) error
}
