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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	devicerepo "github.com/SENERGY-Platform/device-repository/lib/client"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

func (this *HandlerInfo) reactToAnomaly(handlerName string, deviceId string, serviceId string, desc string, timestamp int64) (err error) {
	err = errors.Join(err, this.notify(handlerName, deviceId, serviceId, desc))
	err = errors.Join(err, this.storeAnomalyState(handlerName, deviceId, serviceId, desc, timestamp))
	return err
}

type Notification struct {
	UserId  string `json:"userId" bson:"userId"`
	Title   string `json:"title" bson:"title"`
	Message string `json:"message" bson:"message"`
	Topic   string `json:"topic" bson:"topic"`
}

func (this *HandlerInfo) notify(handlerName string, deviceId string, serviceId string, desc string) error {
	device, err, _ := this.deviceRepoClient.ReadExtendedDevice(deviceId, InternalAdminToken, devicerepo.READ, false)
	if err != nil {
		return fmt.Errorf("unable to get device id=%#v err=%w", deviceId, err)
	}
	msg := Notification{
		UserId:  device.OwnerId,
		Title:   "Anomaly Detected",
		Message: fmt.Sprintf("%v anomaly detected for device %v (%v) in service %v\ndesc: %v\n", handlerName, device.DisplayName, device.Id, serviceId, desc),
		Topic:   this.config.NotificationTopic,
	}
	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(msg)
	if err != nil {
		return err
	}
	endpoint := this.config.NotificationUrl + "/notifications?ignore_duplicates_within_seconds=" + strconv.FormatInt(this.config.NotificationsIgnoreDuplicatesWithinS, 10)
	if this.config.Debug {
		log.Printf("DEBUG: send notification to %v with %v\n", msg.UserId, endpoint)
	}
	req, err := http.NewRequest("POST", endpoint, b)
	if err != nil {
		return err
	}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		respMsg, _ := io.ReadAll(resp.Body)
		log.Println("ERROR: unexpected response status from notifier", resp.StatusCode, string(respMsg))
		return errors.New("unexpected response status from notifier " + resp.Status)
	}
	return nil
}

func (this *HandlerInfo) storeAnomalyState(handlerName string, deviceId string, serviceId string, desc string, timestamp int64) error {
	return this.anomalyStore.StoreAnomaly(handlerName, deviceId, serviceId, desc, timestamp)
}
