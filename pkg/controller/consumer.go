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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/model"
	"log"
	"sort"
	"time"
)

func (this *Controller) updateConsumer(serviceIds []string) error {
	topics := []string{}
	for _, serviceId := range serviceIds {
		topics = append(topics, model.ServiceIdToTopic(serviceId))
	}
	sort.Strings(topics)
	return this.consumer.UpdateTopics(topics)
}

func (this *Controller) HandleConsumerMessage(msg model.ConsumerMessage) (err error) {
	event := model.EventMessage{}
	err = json.Unmarshal(msg.Message, &event)
	if err != nil {
		err = fmt.Errorf("unable to unmarshal event: err=%#v msg=%#v", err.Error(), string(msg.Message))
		log.Println("ERROR:", err)
		err = errors.Join(err, model.ErrWillBeIgnored)
		return err
	}
	eventWithTimestamp := model.EventMessageWithTimestamp{
		EventMessage: event,
		Timestamp:    msg.Timestamp,
	}
	if eventWithTimestamp.Timestamp == 0 {
		eventWithTimestamp.Timestamp = time.Now().Unix()
	}
	return this.Send(eventWithTimestamp)
}
