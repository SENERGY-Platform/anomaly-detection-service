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
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/model"
	"github.com/SENERGY-Platform/device-selection/pkg/model/devicemodel"
	"log"
)

func (this *HandlerInfo) marshal(rawValue map[string]interface{}, service devicemodel.Service) (value interface{}, err error) {
	protocol, ok := this.protocols[service.ProtocolId]
	if !ok {
		return value, fmt.Errorf("unknown service protocol: %s -> %w", service.ProtocolId, model.ErrWillBeIgnored)
	}
	paths := this.marshaller.GetOutputPaths(service, this.handler.Function, &this.aspectNode)
	if len(paths) > 1 {
		log.Println("WARNING: only first path found by FunctionId and AspectNode is used for Unmarshal:", service.Id, paths)
	}
	if len(paths) == 0 {
		return value, errors.New("no output path found for criteria")
	}
	return this.marshaller.Unmarshal(protocol, service, this.handler.Characteristic, paths[0], nil, rawValue)
}
