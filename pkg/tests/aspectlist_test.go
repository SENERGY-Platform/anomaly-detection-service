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

package tests

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/configuration"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/controller"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/handler"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/model"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/tests/docker"
	"github.com/SENERGY-Platform/device-repository/v2/lib/client"
	devicemodel "github.com/SENERGY-Platform/device-repository/v2/lib/model"
	"github.com/SENERGY-Platform/models/go/models"
	permissions "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	model2 "github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/segmentio/kafka-go"
)

// TestIntegrationAspectList drives the aspect list through the whole chain: the criteria that
// goes to device-selection, the aspect nodes that come back from the device-repository, and
// the path the marshaller then picks out of the message.
//
// The device type has one output variable carrying two aspects, one from each of two
// hierarchies. That shape is what makes the AND observable: a handler naming both aspects has
// to find it, and a handler naming one of them plus an aspect the variable does not carry has
// to find nothing. Two aspects of the same hierarchy would be unanswerable by construction,
// because a content variable carries at most one aspect per aspect class.
func TestIntegrationAspectList(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initialConfig, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	config, err := docker.Start(ctx, wg, initialConfig, nil)
	if err != nil {
		t.Error(err)
		return
	}

	protocolId := models.URN_PREFIX + "protocol:1"
	protocolSegmentId := models.URN_PREFIX + "protocol-segment:ps1"
	functionId := models.URN_PREFIX + "measuring-function:f1"
	conceptId := models.URN_PREFIX + "concept:c1"
	characteristicId1 := models.URN_PREFIX + "characteristic:c1"
	characteristicId2 := models.URN_PREFIX + "characteristic:c2"
	deviceTypeId := models.URN_PREFIX + "device-type:dt1"
	serviceId := models.URN_PREFIX + "service:s1"
	deviceId := models.URN_PREFIX + "device:d1"

	//a second device whose variable carries aspect a but not aspect b. Nothing is ever sent to
	//it; it exists so that a criteria naming both aspects has something it has to exclude.
	deviceTypeId2 := models.URN_PREFIX + "device-type:dt2"
	serviceId2 := models.URN_PREFIX + "service:s2"
	deviceId2 := models.URN_PREFIX + "device:d2"

	// three hierarchies; the leaves are what a content variable may carry, the roots are what
	// the handlers ask for, so the subtree coverage of an aspect is exercised as well
	aspectAId := models.URN_PREFIX + "aspect:a"
	aspectALeafId := models.URN_PREFIX + "aspect:a-leaf"
	aspectBId := models.URN_PREFIX + "aspect:b"
	aspectBLeafId := models.URN_PREFIX + "aspect:b-leaf"
	aspectCId := models.URN_PREFIX + "aspect:c"
	aspectCLeafId := models.URN_PREFIX + "aspect:c-leaf"

	err = docker.InitTopic(config.KafkaUrl, model.ServiceIdToTopic(serviceId), model.ServiceIdToTopic(serviceId2))
	if err != nil {
		t.Error(err)
		return
	}

	err, _ = client.NewClient(config.DeviceRepositoryUrl, nil).Import(
		client.InternalAdminToken,
		devicemodel.ImportExport{
			Protocols: []models.Protocol{
				{
					Id:               protocolId,
					Name:             "protocol1",
					Handler:          "foo",
					ProtocolSegments: []models.ProtocolSegment{{Id: protocolSegmentId, Name: "pl"}},
				},
			},
			Functions: []models.Function{
				{
					Id:          functionId,
					Name:        "f1",
					DisplayName: "f1",
					ConceptId:   conceptId,
					RdfType:     models.SES_ONTOLOGY_MEASURING_FUNCTION,
				},
			},
			Aspects: []models.Aspect{
				{Id: aspectAId, Name: "a", SubAspects: []models.Aspect{{Id: aspectALeafId, Name: "a-leaf"}}},
				{Id: aspectBId, Name: "b", SubAspects: []models.Aspect{{Id: aspectBLeafId, Name: "b-leaf"}}},
				{Id: aspectCId, Name: "c", SubAspects: []models.Aspect{{Id: aspectCLeafId, Name: "c-leaf"}}},
			},
			Concepts: []models.Concept{
				{
					Id:                   conceptId,
					Name:                 "concept1",
					CharacteristicIds:    []string{characteristicId1, characteristicId2},
					BaseCharacteristicId: characteristicId1,
					Conversions: []models.ConverterExtension{
						{From: characteristicId1, To: characteristicId2, Formula: "x/10", PlaceholderName: "x"},
						{From: characteristicId2, To: characteristicId1, Formula: "x*10", PlaceholderName: "x"},
					},
				},
			},
			Characteristics: []models.Characteristic{
				{Id: characteristicId1, Name: "c1", Type: models.Float},
				{Id: characteristicId2, Name: "c2", Type: models.Float},
			},
			DeviceTypes: []models.DeviceType{
				{
					Id:   deviceTypeId,
					Name: "dt1",
					Services: []models.Service{
						{
							Id:          serviceId,
							LocalId:     "service1",
							Name:        "service1",
							Interaction: models.EVENT,
							ProtocolId:  protocolId,
							Outputs: []models.Content{
								{
									Id: models.URN_PREFIX + "content:content1",
									ContentVariable: models.ContentVariable{
										Id:   models.URN_PREFIX + "content-variable:cv1",
										Name: "root",
										Type: models.Structure,
										SubContentVariables: []models.ContentVariable{
											{
												Id:   models.URN_PREFIX + "content-variable:cvs1",
												Name: "time",
												Type: models.Integer,
											},
											{
												Id:               models.URN_PREFIX + "content-variable:cvs2",
												Name:             "value",
												Type:             models.Float,
												FunctionId:       functionId,
												AspectIds:        []string{aspectALeafId, aspectBLeafId},
												CharacteristicId: characteristicId2,
											},
										},
									},
									Serialization:     models.JSON,
									ProtocolSegmentId: protocolSegmentId,
								},
							},
						},
					},
				},
				{
					Id:   deviceTypeId2,
					Name: "dt2",
					Services: []models.Service{
						{
							Id:          serviceId2,
							LocalId:     "service2",
							Name:        "service2",
							Interaction: models.EVENT,
							ProtocolId:  protocolId,
							Outputs: []models.Content{
								{
									Id: models.URN_PREFIX + "content:content2",
									ContentVariable: models.ContentVariable{
										Id:   models.URN_PREFIX + "content-variable:cv2",
										Name: "root",
										Type: models.Structure,
										SubContentVariables: []models.ContentVariable{
											{
												Id:               models.URN_PREFIX + "content-variable:cvs3",
												Name:             "value",
												Type:             models.Float,
												FunctionId:       functionId,
												AspectIds:        []string{aspectALeafId},
												CharacteristicId: characteristicId2,
											},
										},
									},
									Serialization:     models.JSON,
									ProtocolSegmentId: protocolSegmentId,
								},
							},
						},
					},
				},
			},
			Devices: []models.Device{
				{
					Id:           deviceId,
					LocalId:      "device1",
					Name:         "device1",
					DeviceTypeId: deviceTypeId,
					OwnerId:      "owner",
					Attributes: []models.Attribute{{
						Key:    config.AnomalyDetectorAttribute,
						Value:  "true",
						Origin: "test",
					}},
				},
				{
					Id:           deviceId2,
					LocalId:      "device2",
					Name:         "device2",
					DeviceTypeId: deviceTypeId2,
					OwnerId:      "owner",
					Attributes: []models.Attribute{{
						Key:    config.AnomalyDetectorAttribute,
						Value:  "true",
						Origin: "test",
					}},
				},
			},
			Permissions: []permissions.Resource{
				{
					Id:      deviceId,
					TopicId: "devices",
					ResourcePermissions: permissions.ResourcePermissions{
						UserPermissions: map[string]model2.PermissionsMap{
							"owner": {Read: true, Write: true, Execute: true, Administrate: true},
						},
					},
				},
				{
					Id:      deviceId2,
					TopicId: "devices",
					ResourcePermissions: permissions.ResourcePermissions{
						UserPermissions: map[string]model2.PermissionsMap{
							"owner": {Read: true, Write: true, Execute: true, Administrate: true},
						},
					},
				},
			},
		},
		devicemodel.ImportExportOptions{IncludeOwnedInformation: true},
	)
	if err != nil {
		t.Error(err)
		return
	}

	bothAspects := &TestHandler{}
	deprecatedAspect := &TestHandler{}
	singleElementList := &TestHandler{}
	unsatisfiable := &TestHandler{}

	reg := handler.NewRegister()
	reg.RegisterWithAspects("both", functionId, []string{aspectAId, aspectBId}, characteristicId1, 2, bothAspects)
	reg.Register("deprecated", functionId, aspectAId, characteristicId1, 2, deprecatedAspect)
	reg.RegisterWithAspects("single-element-list", functionId, []string{aspectAId}, characteristicId1, 2, singleElementList)
	reg.RegisterWithAspects("unsatisfiable", functionId, []string{aspectAId, aspectCId}, characteristicId1, 2, unsatisfiable)

	ctrl, err := controller.StartController(ctx, wg, config, reg)
	if err != nil {
		t.Error(err)
		return
	}

	//the consumer reads from the last offset, so it has to be subscribed before anything is sent
	time.Sleep(15 * time.Second)

	producer := kafka.Writer{
		Addr:                   kafka.TCP(config.KafkaUrl),
		Topic:                  model.ServiceIdToTopic(serviceId),
		MaxAttempts:            10,
		BatchSize:              1,
		AllowAutoTopicCreation: true,
	}
	defer producer.Close()

	now := time.Unix(0, 0)
	t.Run("send events", func(t *testing.T) {
		for i := 1; i <= 3; i++ {
			payload, err := json.Marshal(model.EventMessage{
				DeviceId:  deviceId,
				ServiceId: serviceId,
				Value: map[string]interface{}{
					"root": map[string]interface{}{
						"time":  now.Add(time.Duration(i) * time.Minute).Unix(),
						"value": i,
					},
				},
			})
			if err != nil {
				t.Error(err)
				return
			}
			err = producer.WriteMessages(context.Background(), kafka.Message{
				Key:   []byte("key1"),
				Value: payload,
				Time:  now.Add(time.Duration(i) * time.Minute),
			})
			if err != nil {
				t.Error(err)
				return
			}
		}
	})

	time.Sleep(10 * time.Second)

	//values arrive as characteristic c2 and are converted to the handler characteristic c1 with x*10
	expectedInputs := [][]interface{}{
		{10.0, 20.0},
		{20.0, 30.0},
	}

	t.Run("aspect list matches a variable carrying all of them", func(t *testing.T) {
		if !reflect.DeepEqual(bothAspects.Inputs, expectedInputs) {
			t.Errorf("unexpected inputs\ne=%#v\na=%#v\n", expectedInputs, bothAspects.Inputs)
		}
	})

	t.Run("deprecated single aspect still matches", func(t *testing.T) {
		if !reflect.DeepEqual(deprecatedAspect.Inputs, expectedInputs) {
			t.Errorf("unexpected inputs\ne=%#v\na=%#v\n", expectedInputs, deprecatedAspect.Inputs)
		}
	})

	t.Run("single element list is the alias of the deprecated aspect", func(t *testing.T) {
		if !reflect.DeepEqual(singleElementList.Inputs, deprecatedAspect.Inputs) {
			t.Errorf("single element list and deprecated aspect disagree\ndeprecated=%#v\nlist=%#v\n", deprecatedAspect.Inputs, singleElementList.Inputs)
		}
	})

	// The AND is what this asserts. Were the aspects read as a choice, aspectA alone would
	// match and this handler would have been called.
	t.Run("an aspect the variable does not carry matches nothing", func(t *testing.T) {
		if len(unsatisfiable.Inputs) != 0 {
			t.Errorf("expected no inputs, got %#v", unsatisfiable.Inputs)
		}
	})

	// The assertions above cannot tell a wrong criteria from a wrong path selection: the
	// marshaller ANDs the same aspects again when it looks for the path, so a handler whose
	// criteria selected too much still ends up never being called. These ask device-selection
	// directly, by reloading a register holding one handler at a time and reading which
	// services came back. service2 carries aspect a but not aspect b, so it is what a criteria
	// naming both has to leave out.
	t.Run("criteria", func(t *testing.T) {
		tests := []struct {
			name     string
			register func(reg *handler.Register)
			expected []string
		}{
			{
				name: "both aspects select only the service carrying both",
				register: func(reg *handler.Register) {
					reg.RegisterWithAspects("h", functionId, []string{aspectAId, aspectBId}, characteristicId1, 2, &TestHandler{})
				},
				expected: []string{serviceId},
			},
			{
				name: "one aspect selects every service carrying it",
				register: func(reg *handler.Register) {
					reg.RegisterWithAspects("h", functionId, []string{aspectAId}, characteristicId1, 2, &TestHandler{})
				},
				expected: []string{serviceId, serviceId2},
			},
			{
				name: "the deprecated single aspect selects the same",
				register: func(reg *handler.Register) {
					reg.Register("h", functionId, aspectAId, characteristicId1, 2, &TestHandler{})
				},
				expected: []string{serviceId, serviceId2},
			},
			{
				name: "an aspect no service carries together with the other selects nothing",
				register: func(reg *handler.Register) {
					reg.RegisterWithAspects("h", functionId, []string{aspectAId, aspectCId}, characteristicId1, 2, &TestHandler{})
				},
				expected: nil,
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				reg := handler.NewRegister()
				test.register(reg)
				serviceIds, err := ctrl.LoadRegister(reg)
				if err != nil {
					t.Error(err)
					return
				}
				slices.Sort(serviceIds)
				if !reflect.DeepEqual(serviceIds, test.expected) {
					t.Errorf("unexpected services\ne=%#v\na=%#v\n", test.expected, serviceIds)
				}
			})
		}
	})
}
