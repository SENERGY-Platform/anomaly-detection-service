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

package anomalystore

import (
	"go.mongodb.org/mongo-driver/mongo"
)

type Anomaly struct {
	Handler       string `json:"handler" bson:"handler"`
	Device        string `json:"device" bson:"device"`
	Service       string `json:"service" bson:"service"`
	Description   string `json:"description" bson:"description"`
	UnixTimestamp int64  `json:"unix_timestamp" bson:"unix_timestamp"`
}

var AnomalyBson = getBsonFieldObject[Anomaly]()

func (this *Mongo) anomalyCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoAnomalyCollection)
}

func (this *Mongo) StoreAnomaly(handlerName string, deviceId string, serviceId string, desc string, timestamp int64) error {
	_, err := this.anomalyCollection().InsertOne(getTimeoutContext(), Anomaly{
		Handler:       handlerName,
		Device:        deviceId,
		Service:       serviceId,
		Description:   desc,
		UnixTimestamp: timestamp,
	})
	if err != nil {
		return err
	}
	return nil
}
