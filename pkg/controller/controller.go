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
	"context"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/configuration"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/controller/anomalystore"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/controller/consumer"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/handler"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/model"
	"github.com/SENERGY-Platform/converter/lib/converter"
	devicerepo "github.com/SENERGY-Platform/device-repository/lib/client"
	"github.com/SENERGY-Platform/device-selection/pkg/client"
	deviceselectionmodel "github.com/SENERGY-Platform/device-selection/pkg/model"
	marshallerconfig "github.com/SENERGY-Platform/marshaller/lib/config"
	marshaller "github.com/SENERGY-Platform/marshaller/lib/marshaller/v2"
	"github.com/SENERGY-Platform/models/go/models"
	"github.com/SENERGY-Platform/service-commons/pkg/cache/invalidator"
	"github.com/SENERGY-Platform/service-commons/pkg/kafka"
	"github.com/SENERGY-Platform/service-commons/pkg/signal"
	"github.com/valkey-io/valkey-go"
	"log"
	"slices"
	"strings"
	"sync"
	"time"
)

type Controller struct {
	config           configuration.Config
	mux              sync.RWMutex
	handler          []HandlerInfo
	selectionClient  client.Client
	deviceRepoClient devicerepo.Interface
	anomalyStore     *anomalystore.Mongo
	valKeyClient     valkey.Client
	marshaller       *marshaller.Marshaller
	debounce         *Debounce
	consumer         *consumer.ManagedKafkaConsumer
}

func StartController(ctx context.Context, wg *sync.WaitGroup, config configuration.Config, register *handler.Register) (controller *Controller, err error) {
	selectionClient := client.NewClient(config.DeviceSelectionUrl)
	repoClient := devicerepo.NewClient(config.DeviceRepositoryUrl, nil)

	valkeyClient, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{config.ValKeyUrl}})

	conv, err := converter.New()
	if err != nil {
		return controller, err
	}

	s := &signal.Broker{}
	conceptrepo, err := NewConceptRepo(ctx, wg, config, s)
	if err != nil {
		return controller, err
	}

	m := marshaller.New(marshallerconfig.Config{}, conv, conceptrepo)

	anomalyStore, err := anomalystore.New(config)
	if err != nil {
		return controller, err
	}

	controller = &Controller{
		config:           config,
		mux:              sync.RWMutex{},
		handler:          nil,
		selectionClient:  selectionClient,
		deviceRepoClient: repoClient,
		valKeyClient:     valkeyClient,
		anomalyStore:     anomalyStore,
		marshaller:       m,
		debounce:         &Debounce{Duration: 2 * time.Second}, //to prevent to many reloads if a series of changes happens
		consumer: consumer.NewManagedKafkaConsumer(config, func(topic string, err error) {
			log.Fatalf("FATAL: error while consuming topic %s: %v\n", topic, err)
		}),
	}

	controller.consumer.SetOutputCallback(controller.HandleConsumerMessage)

	serviceIDs, err := controller.LoadRegister(register)
	if err != nil {
		return controller, err
	}

	err = controller.updateConsumer(serviceIDs)
	if err != nil {
		return controller, err
	}

	err = invalidator.StartCacheInvalidatorAll(ctx, kafka.Config{
		KafkaUrl:    config.KafkaUrl,
		StartOffset: kafka.LastOffset,
		Wg:          wg,
	}, config.CacheInvalidationKafkaTopics, s)
	if err != nil {
		return nil, err
	}

	s.Sub("reload", signal.Known.CacheInvalidationAll, func(value string, wg *sync.WaitGroup) {
		controller.debounce.Do(func() {
			serviceIDs, err = controller.LoadRegister(register)
			if err != nil {
				log.Fatalln("ERROR: unable to refresh register", err)
				return
			}
			err = controller.updateConsumer(serviceIDs)
			if err != nil {
				log.Fatalln("FATAL: unable to update kafka consumer", err)
				return
			}
		})
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		s.Unsub("reload")
		controller.consumer.Stop()
		controller.anomalyStore.Disconnect()
	}()

	return controller, nil
}

func (this *Controller) Send(msg model.EventMessageWithTimestamp) (err error) {
	this.mux.RLock()
	defer this.mux.RUnlock()
	for _, entry := range this.handler {
		err = entry.Send(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

// InternalAdminToken is expired and invalid. but because this service does not validate the received tokens,
// it may be used by trusted internal services which are within the same network (kubernetes cluster).
// requests with this token may not be routed over an ingres with token validation
const InternalAdminToken = devicerepo.InternalAdminToken

func (this *Controller) LoadRegister(register *handler.Register) (serviceIds []string, err error) {
	this.mux.Lock()
	defer this.mux.Unlock()
	if register == nil {
		register = handler.Registry
	}
	this.handler = []HandlerInfo{}

	protocols := map[string]models.Protocol{}
	protocolList, err, _ := this.deviceRepoClient.ListProtocols(InternalAdminToken, 9999, 0, "name.asc")
	if err != nil {
		return nil, err
	}
	for _, protocol := range protocolList {
		protocols[protocol.Id] = protocol
	}

	for _, h := range register.List() {
		selectables, _, err := this.selectionClient.GetSelectables(InternalAdminToken, []models.DeviceGroupFilterCriteria{
			{
				Interaction: models.EVENT,
				FunctionId:  h.Function,
				AspectId:    h.Aspect,
			},
		}, &client.GetSelectablesOptions{
			IncludeGroups:               false,
			IncludeImports:              false,
			IncludeDevices:              true,
			IncludeIdModified:           false,
			WithLocalDeviceIds:          nil,
			FilterByDeviceAttributeKeys: []string{this.config.AnomalyDetectorAttribute},
		})
		if err != nil {
			return nil, err
		}
		match := []deviceselectionmodel.Selectable{}
		for _, selectable := range selectables {
			if selectable.Device != nil && this.hasAnomalyDetectorAttribute(selectable.Device) {
				for _, s := range selectable.Services {
					if !slices.Contains(serviceIds, s.Id) {
						serviceIds = append(serviceIds, s.Id)
					}
				}
				match = append(match, selectable)
			}
		}
		entry, err := this.createRouterEntry(h, match, protocols)
		if err != nil {
			return nil, err
		}
		this.handler = append(this.handler, entry)
	}
	return serviceIds, nil
}

func (this *Controller) createRouterEntry(h handler.Entry, match []deviceselectionmodel.Selectable, protocols map[string]models.Protocol) (HandlerInfo, error) {
	aspectNode, err, _ := this.deviceRepoClient.GetAspectNode(h.Aspect)
	if err != nil {
		return HandlerInfo{}, err
	}
	return HandlerInfo{
		config:           this.config,
		handler:          h,
		match:            match,
		protocols:        protocols,
		marshaller:       this.marshaller,
		aspectNode:       aspectNode,
		valKeyClient:     this.valKeyClient,
		deviceRepoClient: this.deviceRepoClient,
		anomalyStore:     this.anomalyStore,
	}, nil
}

func (this *Controller) hasAnomalyDetectorAttribute(device *deviceselectionmodel.PermSearchDevice) bool {
	if device == nil {
		return false
	}
	for _, attr := range device.Attributes {
		if attr.Key == this.config.AnomalyDetectorAttribute && strings.ToLower(strings.TrimSpace(attr.Value)) == "true" {
			return true
		}
	}
	return false
}
