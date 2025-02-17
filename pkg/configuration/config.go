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

package configuration

import (
	envldr "github.com/SENERGY-Platform/go-env-loader"
	"github.com/SENERGY-Platform/go-service-base/config-hdl"
	"reflect"
	"strings"
)

type Config struct {
	Debug                                bool     `json:"debug" env_var:"DEBUG"`
	KafkaUrl                             string   `json:"kafka_url" env_var:"KAFKA_URL"`
	KafkaConsumerGroup                   string   `json:"kafka_consumer_group" env_var:"KAFKA_CONSUMER_GROUP"`
	ValKeyUrl                            string   `json:"val_key_url" env_var:"VAL_KEY_URL"`
	DeviceRepositoryUrl                  string   `json:"device_repository_url" env_var:"DEVICE_REPOSITORY_URL"`
	DeviceSelectionUrl                   string   `json:"device_selection_url" env_var:"DEVICE_SELECTION_URL"`
	CacheInvalidationKafkaTopics         []string `json:"cache_invalidation_kafka_topics" env_var:"CACHE_INVALIDATION_KAFKA_TOPICS"`
	CacheDuration                        string   `json:"cache_duration" env_var:"CACHE_DURATION"`
	NotificationUrl                      string   `json:"notification_url" env_var:"NOTIFICATION_URL"`
	NotificationTopic                    string   `json:"notification_topic" env_var:"NOTIFICATION_TOPIC"`
	NotificationsIgnoreDuplicatesWithinS int64    `json:"notifications_ignore_duplicates_within_seconds" env_var:"NOTIFICATIONS_IGNORE_DUPLICATES_WITHIN_SECONDS"`
	MongoUrl                             string   `json:"mongo_url" env_var:"MONGO_URL"`
	MongoTable                           string   `json:"mongo_table" env_var:"MONGO_TABLE"`
	MongoAnomalyCollection               string   `json:"mongo_anomaly_collection" env_var:"MONGO_ANOMALY_COLLECTION"`
	AnomalyDetectorAttribute             string   `json:"anomaly_detector_attribute" env_var:"ANOMALY_DETECTOR_ATTRIBUTE"`
}

func Load(location string) (conf Config, err error) {
	err = config_hdl.Load(&conf, nil, typeParser, nil, location)
	return conf, err
}

var typeParser = map[reflect.Type]envldr.Parser{
	reflect.TypeOf([]string{}): listParser,
}

func listParser(_ reflect.Type, val string, _ []string, kwParams map[string]string) (interface{}, error) {
	sep, ok := kwParams["sep"]
	if !ok {
		sep = ","
	}
	return strings.Split(val, sep), nil
}
