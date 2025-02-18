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

package consumer

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/configuration"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/model"
	"log"
	"reflect"
	"sort"
	"sync"
)

type ManagedKafkaConsumer struct {
	cancel  context.CancelFunc
	wg      *sync.WaitGroup
	config  configuration.Config
	output  func(msg model.ConsumerMessage) error
	mux     sync.Mutex
	onError func(topic string, err error)
	stopped bool
	topics  []string
}

func NewManagedKafkaConsumer(config configuration.Config, onError func(topic string, err error)) *ManagedKafkaConsumer {
	return &ManagedKafkaConsumer{
		config:  config,
		onError: onError,
	}
}

func (this *ManagedKafkaConsumer) Stop() {
	this.mux.Lock()
	defer this.mux.Unlock()
	this.stop()
	this.stopped = true
	return
}

func (this *ManagedKafkaConsumer) stop() {
	log.Println("stop consumer")
	if this.cancel != nil {
		this.cancel()
		this.cancel = nil
	}
	if this.wg != nil {
		this.wg.Wait()
		this.wg = nil
	}
	return
}

func (this *ManagedKafkaConsumer) SetOutputCallback(callback func(msg model.ConsumerMessage) error) {
	this.output = callback
}

func (this *ManagedKafkaConsumer) UpdateTopics(topics []string) (err error) {
	this.mux.Lock()
	defer this.mux.Unlock()
	sort.Strings(this.topics)
	sort.Strings(topics)
	if !this.stopped && reflect.DeepEqual(this.topics, topics) {
		log.Println("no topic changes -> continue with current consumer")
		return nil
	}
	defer func() {
		if err == nil {
			this.topics = topics
		} else {
			this.topics = nil
		}
	}()
	if len(topics) <= 20 {
		log.Println("update consumer topics: ", topics)
	} else {
		log.Println("update consumer topics: ", len(topics))
	}
	this.stop()
	var ctx context.Context
	ctx, this.cancel = context.WithCancel(context.Background())
	this.wg = &sync.WaitGroup{}
	return StartKafkaLastOffsetConsumerGroup(ctx, this.wg, this.config.KafkaUrl, this.config.KafkaConsumerGroup, topics, func(msg model.ConsumerMessage) error {
		if !this.stopped && this.output != nil {
			err := this.output(msg)
			if errors.Is(err, model.ErrWillBeIgnored) {
				log.Println("WARNING: kafka listener has thrown an error but will not be retried", err)
				return nil
			}
			return err
		}
		return nil
	}, this.onError)
}
