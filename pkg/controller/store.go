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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/model"
	"github.com/valkey-io/valkey-go"
	"math/rand"
	"slices"
	"time"
)

type Store struct {
	ValKeyClient valkey.Client
}

func (this *Store) Get(key string, value interface{}) (err error) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	resp := this.ValKeyClient.Do(ctx, this.ValKeyClient.B().Get().Key(key).Build())
	err = resp.Error()
	if err != nil {
		return fmt.Errorf("unable to get value %v from value: %w", key, err)
	}
	err = resp.DecodeJSON(value)
	if err != nil {
		return fmt.Errorf("unable to unmarshal value %v: %w", key, err)
	}
	return nil
}

func (this *Store) Set(key string, value interface{}) (err error) {
	valueBuff, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("unable to marshal value %v: %w", key, err)
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	err = this.ValKeyClient.Do(ctx, this.ValKeyClient.B().Set().Key(key).Value(string(valueBuff)).Build()).Error()
	if err != nil {
		return fmt.Errorf("unable to store value %v: %w", key, err)
	}
	return nil
}

func (this *HandlerInfo) storeAndListValues(handlerName string, deviceId string, serviceId string, size int, value interface{}) (result []interface{}, err error) {
	key := fmt.Sprintf("%s_%s_%s", handlerName, deviceId, serviceId)

	valueBuff, err := json.Marshal(value)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("unable to marshal value: %w", err), model.ErrWillBeIgnored)
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)

	//add value to list
	err = this.valKeyClient.Do(ctx, this.valKeyClient.B().Lpush().Key(key).Element(string(valueBuff)).Build()).Error()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("unable to store value: %w", err), model.ErrWithRetry)
	}
	//trim the list
	//on average on every 5th call
	if rand.Int()%5 == 0 {
		err = this.valKeyClient.Do(ctx, this.valKeyClient.B().Ltrim().Key(key).Start(0).Stop(int64(size-1)).Build()).Error()
		if err != nil {
			return nil, errors.Join(fmt.Errorf("unable to trim store: %w", err), model.ErrWithRetry)
		}
	}

	//get list
	resp := this.valKeyClient.Do(ctx, this.valKeyClient.B().Lrange().Key(key).Start(0).Stop(int64(size-1)).Build())
	err = resp.Error()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("unable to get value list from store: %w", err), model.ErrWithRetry)
	}
	err = valkey.DecodeSliceOfJSON(resp, &result)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("unable to unmarshal list from store: %w", err), model.ErrWillBeIgnored)
	}
	slices.Reverse(result)
	return result, nil
}
