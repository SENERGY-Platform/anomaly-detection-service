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

package docker

import (
	"context"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/configuration"
	"github.com/segmentio/kafka-go"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
)

func Start(ctx context.Context, wg *sync.WaitGroup, confIn configuration.Config, notifierF func(http.ResponseWriter, *http.Request)) (confOut configuration.Config, err error) {
	confOut = confIn

	_, mongoIp, err := MongoDB(ctx, wg)
	if err != nil {
		return confOut, err
	}
	confOut.MongoUrl = "mongodb://" + mongoIp + ":27017"

	_, zkIp, err := Zookeeper(ctx, wg)
	if err != nil {
		return confOut, err
	}
	zookeeperUrl := zkIp + ":2181"

	confOut.KafkaUrl, err = Kafka(ctx, wg, zookeeperUrl)
	if err != nil {
		return confOut, err
	}

	_, permV2Ip, err := PermissionsV2(ctx, wg, confOut.MongoUrl, confOut.KafkaUrl)
	if err != nil {
		return confOut, err
	}
	permissionsV2Url := "http://" + permV2Ip + ":8080"

	_, deviceRepoIp, err := DeviceRepo(ctx, wg, confOut.KafkaUrl, confOut.MongoUrl, permissionsV2Url)
	if err != nil {
		return confOut, err
	}
	confOut.DeviceRepositoryUrl = "http://" + deviceRepoIp + ":8080"

	_, deviceSelectionIp, err := DeviceSelection(ctx, wg, confOut.KafkaUrl, confOut.DeviceRepositoryUrl)
	if err != nil {
		return confOut, err
	}
	confOut.DeviceSelectionUrl = "http://" + deviceSelectionIp + ":8080"

	_, valKeyIp, err := ValKey(ctx, wg)
	if err != nil {
		return confOut, err
	}
	confOut.ValKeyUrl = valKeyIp + ":6379"

	if notifierF != nil {
		notifierServer := httptest.NewServer(http.HandlerFunc(notifierF))
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ctx.Done()
			notifierServer.Close()
		}()
		confOut.NotificationUrl = notifierServer.URL
	}

	return confOut, nil
}

func InitTopic(bootstrapUrl string, topics ...string) (err error) {
	conn, err := kafka.Dial("tcp", bootstrapUrl)
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}
	var controllerConn *kafka.Conn
	controllerConn, err = kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{}

	for _, topic := range topics {
		topicConfigs = append(topicConfigs, kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
			ConfigEntries: []kafka.ConfigEntry{
				{
					ConfigName:  "retention.ms",
					ConfigValue: "2592000000", // 30d
				},
			},
		})
	}

	return controllerConn.CreateTopics(topicConfigs...)
}
