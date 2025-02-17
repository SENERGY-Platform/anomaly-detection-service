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

package tests

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/configuration"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/controller"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/controller/anomalystore"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/handler"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/model"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/tests/docker"
	"github.com/SENERGY-Platform/device-repository/lib/client"
	devicemodel "github.com/SENERGY-Platform/device-repository/lib/model"
	"github.com/SENERGY-Platform/models/go/models"
	permissions "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	model2 "github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"io"
	"log"
	"net/http"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestIntegration(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initialConfig, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	notifications := map[string][]string{}
	notificationsMux := sync.Mutex{}

	config, err := docker.Start(ctx, wg, initialConfig, func(writer http.ResponseWriter, request *http.Request) {
		notificationsMux.Lock()
		defer notificationsMux.Unlock()
		temp, _ := io.ReadAll(request.Body)
		path := request.URL.Path + "?" + request.URL.Query().Encode()
		notifications[path] = append(notifications[path], string(temp))
		log.Printf("notifications: %v %v", path, string(temp))
	})
	if err != nil {
		t.Error(err)
		return
	}

	testHandler := &TestHandler{
		F: func(values []interface{}) (anomaly bool, description string, err error) {
			convertedValues, err := handler.CastList[float64](values)
			if err != nil {
				return false, "", err
			}
			if slices.Contains(convertedValues, 100) {
				return true, "contains 100", nil
			}
			return false, "", nil
		},
	}
	ignoredTestHandler := &TestHandler{
		F: func(values []interface{}) (anomaly bool, description string, err error) {
			return false, "", nil
		},
	}

	protocolId := models.URN_PREFIX + "protocol:1"
	functionId := models.URN_PREFIX + "measuring-function:f1"
	functionIgnoredId := models.URN_PREFIX + "measuring-function:f2"
	aspectId := models.URN_PREFIX + "aspect:a1"
	subAspectId := models.URN_PREFIX + "aspect:a1s1"
	characteristicId1 := models.URN_PREFIX + "characteristic:c1"
	characteristicId2 := models.URN_PREFIX + "characteristic:c2"
	conceptId1 := models.URN_PREFIX + "concept:c1"
	serviceId := models.URN_PREFIX + "service:s1"
	deviceId := models.URN_PREFIX + "device:d1"
	ignoredDeviceId := models.URN_PREFIX + "device:d2"
	deviceTypeId := models.URN_PREFIX + "device-type:dt1"
	protocolSegmentId := models.URN_PREFIX + "protocol-segment:ps1"

	err = docker.InitTopic(config.KafkaUrl, model.ServiceIdToTopic(serviceId))
	if err != nil {
		t.Error(err)
		return
	}

	err, _ = client.NewClient(config.DeviceRepositoryUrl, nil).Import(
		client.InternalAdminToken,
		devicemodel.ImportExport{
			Protocols: []models.Protocol{
				{
					Id:      protocolId,
					Name:    "protocol1",
					Handler: "foo",
					ProtocolSegments: []models.ProtocolSegment{
						{
							Id:   protocolSegmentId,
							Name: "pl",
						},
					},
				},
			},
			Functions: []models.Function{
				{
					Id:          functionId,
					Name:        "f1",
					DisplayName: "f1",
					ConceptId:   conceptId1,
					RdfType:     models.SES_ONTOLOGY_MEASURING_FUNCTION,
				},
				{
					Id:          functionIgnoredId,
					Name:        "f2",
					DisplayName: "f2",
					ConceptId:   conceptId1,
					RdfType:     models.SES_ONTOLOGY_MEASURING_FUNCTION,
				},
			},
			Aspects: []models.Aspect{
				{
					Id:   aspectId,
					Name: "a1",
					SubAspects: []models.Aspect{
						{
							Id:   subAspectId,
							Name: "a1s1",
						},
					},
				},
			},
			Concepts: []models.Concept{
				{
					Id:                   conceptId1,
					Name:                 "concept1",
					CharacteristicIds:    []string{characteristicId1, characteristicId2},
					BaseCharacteristicId: characteristicId1,
					Conversions: []models.ConverterExtension{
						{
							From:            characteristicId1,
							To:              characteristicId2,
							Distance:        0,
							Formula:         "x/10",
							PlaceholderName: "x",
						},
						{
							From:            characteristicId2,
							To:              characteristicId1,
							Distance:        0,
							Formula:         "x*10",
							PlaceholderName: "x",
						},
					},
				},
			},
			Characteristics: []models.Characteristic{
				{
					Id:   characteristicId1,
					Name: "c1",
					Type: models.Float,
				},
				{
					Id:   characteristicId2,
					Name: "c2",
					Type: models.Float,
				},
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
												AspectId:         subAspectId,
												CharacteristicId: characteristicId2,
											},
											{
												Id:   models.URN_PREFIX + "content-variable:cvs3",
												Name: "foo",
												Type: models.String,
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
					Attributes:   []models.Attribute{},
					DeviceTypeId: deviceTypeId,
					OwnerId:      "owner",
				},
				{
					Id:           ignoredDeviceId,
					LocalId:      "device2",
					Name:         "device2",
					Attributes:   []models.Attribute{},
					DeviceTypeId: deviceTypeId,
					OwnerId:      "owner",
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
					Id:      ignoredDeviceId,
					TopicId: "devices",
					ResourcePermissions: permissions.ResourcePermissions{
						UserPermissions: map[string]model2.PermissionsMap{
							"owner": {Read: true, Write: true, Execute: true, Administrate: true},
						},
					},
				},
			},
		},
		devicemodel.ImportExportOptions{
			IncludeOwnedInformation: true,
		},
	)
	if err != nil {
		t.Error(err)
		return
	}

	reg := handler.NewRegister()
	reg.Register("test", functionId, aspectId, characteristicId1, 5, testHandler)
	reg.Register("ignoredTestHandler", functionIgnoredId, aspectId, characteristicId1, 5, ignoredTestHandler)

	_, err = controller.StartController(ctx, wg, config, reg)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(5 * time.Second)

	//update device
	_, err, _ = client.NewClient(config.DeviceRepositoryUrl, nil).SetDevice(client.InternalAdminToken, models.Device{
		Id:      deviceId,
		LocalId: "device1",
		Name:    "device1",
		Attributes: []models.Attribute{{
			Key:    config.AnomalyDetectorAttribute,
			Value:  "true",
			Origin: "test",
		}},
		DeviceTypeId: deviceTypeId,
		OwnerId:      "owner",
	}, client.DeviceUpdateOptions{})
	if err != nil {
		t.Error(err)
		return
	}

	producer := kafka.Writer{
		Addr:                   kafka.TCP(config.KafkaUrl),
		Topic:                  model.ServiceIdToTopic(serviceId),
		MaxAttempts:            10,
		BatchSize:              1,
		AllowAutoTopicCreation: true,
	}
	defer producer.Close()

	now := time.Unix(0, 0)
	send := func(value map[string]interface{}, duration time.Duration) error {
		pl1, err := json.Marshal(model.EventMessage{
			DeviceId:  deviceId,
			ServiceId: serviceId,
			Value:     value,
		})
		if err != nil {
			return err
		}
		pl2, err := json.Marshal(model.EventMessage{
			DeviceId:  ignoredDeviceId,
			ServiceId: serviceId,
			Value:     value,
		})
		if err != nil {
			return err
		}
		return producer.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte("key1"),
				Value: pl1,
				Time:  now.Add(duration),
			}, kafka.Message{
				Key:   []byte("key2"),
				Value: pl2,
				Time:  now.Add(duration),
			},
		)
	}

	time.Sleep(10 * time.Second)

	t.Run("send events", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			err = send(map[string]interface{}{
				"root": map[string]interface{}{
					"time":    now.Add(time.Duration(i) * time.Minute).Unix(),
					"value":   i,
					"ignored": "foo",
				},
			}, time.Duration(i)*time.Minute)
			if err != nil {
				t.Error(err)
				return
			}
		}
	})

	time.Sleep(5 * time.Second)

	t.Run("check test handler inputs", func(t *testing.T) {
		if len(testHandler.Inputs) != 16 { //20 events, first 4 are invalid
			t.Errorf("test handler should have 15 inputs %v %#v", len(testHandler.Inputs), testHandler.Inputs)
		}
		expected := [][]interface{}{
			{0.0, 10.0, 20.0, 30.0, 40.0},
			{10.0, 20.0, 30.0, 40.0, 50.0},
			{20.0, 30.0, 40.0, 50.0, 60.0},
			{30.0, 40.0, 50.0, 60.0, 70.0},
			{40.0, 50.0, 60.0, 70.0, 80.0},
			{50.0, 60.0, 70.0, 80.0, 90.0},
			{60.0, 70.0, 80.0, 90.0, 100.0},
			{70.0, 80.0, 90.0, 100.0, 110.0},
			{80.0, 90.0, 100.0, 110.0, 120.0},
			{90.0, 100.0, 110.0, 120.0, 130.0},
			{100.0, 110.0, 120.0, 130.0, 140.0},
			{110.0, 120.0, 130.0, 140.0, 150.0},
			{120.0, 130.0, 140.0, 150.0, 160.0},
			{130.0, 140.0, 150.0, 160.0, 170.0},
			{140.0, 150.0, 160.0, 170.0, 180.0},
			{150.0, 160.0, 170.0, 180.0, 190.0},
		}
		if !reflect.DeepEqual(testHandler.Inputs, expected) {
			t.Errorf("unexpected inputs \ne=%#v\na=%#v\n", expected, testHandler.Inputs)
		}
	})
	t.Run("check test handler outputs", func(t *testing.T) {
		if len(testHandler.Outputs) != 16 { //20 events, first 4 are invalid
			t.Errorf("test handler should have 15 inputs %v %#v", len(testHandler.Inputs), testHandler.Inputs)
		}
		expected := []struct {
			Anomaly     bool
			Description string
			Err         error
		}{
			{Anomaly: false, Description: "", Err: error(nil)},
			{Anomaly: false, Description: "", Err: error(nil)},
			{Anomaly: false, Description: "", Err: error(nil)},
			{Anomaly: false, Description: "", Err: error(nil)},
			{Anomaly: false, Description: "", Err: error(nil)},
			{Anomaly: false, Description: "", Err: error(nil)},
			{Anomaly: true, Description: "contains 100", Err: error(nil)},
			{Anomaly: true, Description: "contains 100", Err: error(nil)},
			{Anomaly: true, Description: "contains 100", Err: error(nil)},
			{Anomaly: true, Description: "contains 100", Err: error(nil)},
			{Anomaly: true, Description: "contains 100", Err: error(nil)},
			{Anomaly: false, Description: "", Err: error(nil)},
			{Anomaly: false, Description: "", Err: error(nil)},
			{Anomaly: false, Description: "", Err: error(nil)},
			{Anomaly: false, Description: "", Err: error(nil)},
			{Anomaly: false, Description: "", Err: error(nil)},
		}
		if !reflect.DeepEqual(testHandler.Outputs, expected) {
			t.Errorf("unexpected outputs \ne=%#v\na=%#v\n", expected, testHandler.Outputs)
		}
	})

	t.Run("check ignored test handler", func(t *testing.T) {
		if len(ignoredTestHandler.Inputs) != 0 {
			t.Errorf("ignored test handler should have 0 inputs %v", ignoredTestHandler.Inputs)
		}
		if len(ignoredTestHandler.Outputs) != 0 {
			t.Errorf("ignored test handler should have 0 outputs %v", ignoredTestHandler.Inputs)
		}
	})
	t.Run("check notifications", func(t *testing.T) {
		notificationsMux.Lock()
		defer notificationsMux.Unlock()
		expected := map[string][]string{
			"/notifications?ignore_duplicates_within_seconds=86400": {
				`{"userId":"owner","title":"Anomaly Detected","message":"test anomaly detected for device device1 (urn:infai:ses:device:d1) in service urn:infai:ses:service:s1\ndesc: contains 100\n","topic":"analytics"}` + "\n",
				`{"userId":"owner","title":"Anomaly Detected","message":"test anomaly detected for device device1 (urn:infai:ses:device:d1) in service urn:infai:ses:service:s1\ndesc: contains 100\n","topic":"analytics"}` + "\n",
				`{"userId":"owner","title":"Anomaly Detected","message":"test anomaly detected for device device1 (urn:infai:ses:device:d1) in service urn:infai:ses:service:s1\ndesc: contains 100\n","topic":"analytics"}` + "\n",
				`{"userId":"owner","title":"Anomaly Detected","message":"test anomaly detected for device device1 (urn:infai:ses:device:d1) in service urn:infai:ses:service:s1\ndesc: contains 100\n","topic":"analytics"}` + "\n",
				`{"userId":"owner","title":"Anomaly Detected","message":"test anomaly detected for device device1 (urn:infai:ses:device:d1) in service urn:infai:ses:service:s1\ndesc: contains 100\n","topic":"analytics"}` + "\n",
			},
		}
		if !reflect.DeepEqual(notifications, expected) {
			t.Errorf("unexpected notification calls \ne=%#v\na=%#v\n", expected, notifications)
		}
	})
	t.Run("check db", func(t *testing.T) {
		c, err := mongo.Connect(ctx, options.Client().ApplyURI(config.MongoUrl), options.Client().SetReadConcern(readconcern.Majority()))
		if err != nil {
			t.Error(err)
		}
		cursor, err := c.Database(config.MongoTable).Collection(config.MongoAnomalyCollection).Find(ctx, bson.D{})
		if err != nil {
			t.Error(err)
			return
		}
		var list []anomalystore.Anomaly
		err = cursor.All(ctx, &list)
		if err != nil {
			t.Error(err)
			return
		}
		expected := []anomalystore.Anomaly{
			{Handler: "test", Device: "urn:infai:ses:device:d1", Service: "urn:infai:ses:service:s1", Description: "contains 100", UnixTimestamp: now.Add(time.Duration(10) * time.Minute).Unix()},
			{Handler: "test", Device: "urn:infai:ses:device:d1", Service: "urn:infai:ses:service:s1", Description: "contains 100", UnixTimestamp: now.Add(time.Duration(11) * time.Minute).Unix()},
			{Handler: "test", Device: "urn:infai:ses:device:d1", Service: "urn:infai:ses:service:s1", Description: "contains 100", UnixTimestamp: now.Add(time.Duration(12) * time.Minute).Unix()},
			{Handler: "test", Device: "urn:infai:ses:device:d1", Service: "urn:infai:ses:service:s1", Description: "contains 100", UnixTimestamp: now.Add(time.Duration(13) * time.Minute).Unix()},
			{Handler: "test", Device: "urn:infai:ses:device:d1", Service: "urn:infai:ses:service:s1", Description: "contains 100", UnixTimestamp: now.Add(time.Duration(14) * time.Minute).Unix()},
		}
		if !reflect.DeepEqual(list, expected) {
			t.Errorf("unexpected anomaly\ne=%#v\na=%#v\n", expected, list)
		}
	})

}

type TestHandler struct {
	mux     sync.Mutex
	F       func(values []interface{}) (anomaly bool, description string, err error)
	Inputs  [][]interface{}
	Outputs []struct {
		Anomaly     bool
		Description string
		Err         error
	}
}

func (this *TestHandler) Handle(values []interface{}) (anomaly bool, description string, err error) {
	this.mux.Lock()
	defer this.mux.Unlock()
	f := this.F
	if f == nil {
		f = func(values []interface{}) (anomaly bool, description string, err error) {
			return false, "", nil
		}
	}
	this.Inputs = append(this.Inputs, values)
	anomaly, description, err = f(values)
	this.Outputs = append(this.Outputs, struct {
		Anomaly     bool
		Description string
		Err         error
	}{
		Anomaly:     anomaly,
		Description: description,
		Err:         err,
	})
	return anomaly, description, err
}
