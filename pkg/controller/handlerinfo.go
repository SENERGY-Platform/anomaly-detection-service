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
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/configuration"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/controller/anomalystore"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/handler"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/model"
	"github.com/SENERGY-Platform/device-repository/lib/client"
	deviceselectionmodel "github.com/SENERGY-Platform/device-selection/pkg/model"
	marshaller "github.com/SENERGY-Platform/marshaller/lib/marshaller/v2"
	"github.com/SENERGY-Platform/models/go/models"
	"github.com/valkey-io/valkey-go"
)

type HandlerInfo struct {
	config           configuration.Config
	handler          handler.Entry
	match            []deviceselectionmodel.Selectable
	protocols        map[string]models.Protocol
	aspectNode       models.AspectNode
	marshaller       *marshaller.Marshaller
	valKeyClient     valkey.Client
	deviceRepoClient client.Interface
	anomalyStore     *anomalystore.Mongo
}

func (this *HandlerInfo) Send(msg model.EventMessageWithTimestamp) error {
	for _, e := range this.match {
		if e.Device != nil && e.Device.Id == msg.DeviceId {
			for _, s := range e.Services {
				if s.Id == msg.ServiceId {
					return this.do(msg.DeviceId, s, msg.Value, msg.Timestamp)
				}
			}
			return nil
		}
	}
	return nil
}

func (this *HandlerInfo) do(deviceId string, service models.Service, rawValue map[string]interface{}, timestamp int64) error {
	marshalledValue, err := this.marshal(rawValue, service)
	if err != nil {
		return errors.Join(fmt.Errorf("unable to marshal"), err, model.ErrWillBeIgnored)
	}
	list, err := this.storeAndListValues(this.handler.Name, deviceId, service.Id, this.handler.BufferSize, marshalledValue)
	if err != nil {
		return fmt.Errorf("unable to storeAndListValues: %w", err)
	}
	if len(list) < this.handler.BufferSize {
		return nil
	}
	anomaly, desc, err := this.callHandler(list)
	if err != nil {
		return errors.Join(fmt.Errorf("unable to handle"), err, model.ErrWillBeIgnored)
	}
	if anomaly {
		err = this.reactToAnomaly(this.handler.Name, deviceId, service.Id, desc, timestamp)
		if err != nil {
			return errors.Join(fmt.Errorf("unable to react to anomaly"), err, model.ErrWillBeIgnored)
		}
	}
	return nil
}

func (this *HandlerInfo) callHandler(values []interface{}) (anomaly bool, description string, err error) {
	defer func() {
		if r := recover(); r != nil {
			anomaly = false
			description = ""
			err = errors.New("panic:" + fmt.Sprint(r))
		}
	}()
	anomaly, description, err = this.handler.Handle(values)
	return anomaly, description, err
}
