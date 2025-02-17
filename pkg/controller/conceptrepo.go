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
	"errors"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/configuration"
	"github.com/SENERGY-Platform/device-repository/lib/client"
	"github.com/SENERGY-Platform/marshaller/lib/marshaller/model"
	marshaller "github.com/SENERGY-Platform/marshaller/lib/marshaller/v2"
	"github.com/SENERGY-Platform/models/go/models"
	"github.com/SENERGY-Platform/service-commons/pkg/cache"
	"github.com/SENERGY-Platform/service-commons/pkg/signal"
	"log"
	"sync"
	"time"
)

type CachedConceptRepository struct {
	config configuration.Config
	cache  *cache.Cache
	client client.Interface
	exp    time.Duration
}

func NewConceptRepo(ctx context.Context, wg *sync.WaitGroup, config configuration.Config, s *signal.Broker) (*CachedConceptRepository, error) {
	c, err := cache.New(cache.Config{
		CacheInvalidationSignalBroker: s,
		CacheInvalidationSignalHooks: map[cache.Signal]cache.ToKey{
			signal.Known.CacheInvalidationAll: nil,
		},
	})
	if err != nil {
		return nil, err
	}
	exp, err := time.ParseDuration(config.CacheDuration)
	if err != nil {
		return nil, err
	}
	return &CachedConceptRepository{
		config: config,
		cache:  c,
		client: client.NewClient(config.DeviceRepositoryUrl, nil),
		exp:    exp,
	}, nil
}

func (this *CachedConceptRepository) GetConcept(id string) (concept model.Concept, err error) {
	return cache.Use(this.cache, "concept:"+id, func() (model.Concept, error) {
		result, err, _ := this.client.GetConceptWithoutCharacteristics(id)
		return result, err
	}, cache.NoValidation, this.exp)
}

func (this *CachedConceptRepository) GetConceptIdOfFunction(id string) string {
	cid, err := cache.Use(this.cache, "concept-of-function:"+id, func() (string, error) {
		f, err, _ := this.client.GetFunction(id)
		if err != nil {
			return "", err
		}
		return f.ConceptId, err
	}, cache.NoValidation, this.exp)
	if err != nil {
		log.Println("ERROR: unable to get concept of function: " + err.Error())
		return ""
	}
	return cid
}

func (this *CachedConceptRepository) GetCharacteristic(id marshaller.CharacteristicId) (model.Characteristic, error) {
	infos, err := this.GetCharacteristicInfos()
	if err != nil {
		return model.Characteristic{}, err
	}
	result, ok := infos.ById[id]
	if !ok {
		return model.Characteristic{}, errors.New("characteristic not found")
	}
	return result, nil
}

type ConceptInfos struct {
	ConceptIdsByCharacteristicId map[string][]string
}

func (this *CachedConceptRepository) GetConceptsInfos() (infos ConceptInfos, err error) {
	return cache.Use(this.cache, "concept-infos", func() (ConceptInfos, error) {
		concepts, _, err, _ := this.client.ListConcepts(client.ConceptListOptions{Limit: 99999})
		if err != nil {
			return ConceptInfos{}, err
		}
		infos = ConceptInfos{
			ConceptIdsByCharacteristicId: map[string][]string{},
		}
		for _, concept := range concepts {
			for _, characteristic := range concept.CharacteristicIds {
				infos.ConceptIdsByCharacteristicId[characteristic] = append(infos.ConceptIdsByCharacteristicId[characteristic], concept.Id)
			}
		}
		return infos, nil
	}, cache.NoValidation, this.exp)
}

type CharacteristicInfos struct {
	ById       map[string]model.Characteristic //has also child characteristics
	IdToRootId map[string]string
}

func (this *CachedConceptRepository) GetCharacteristicInfos() (infos CharacteristicInfos, err error) {
	return cache.Use(this.cache, "characteristic-infos", func() (CharacteristicInfos, error) {
		characteristics, _, err, _ := this.client.ListCharacteristics(client.CharacteristicListOptions{Limit: 99999})
		if err != nil {
			return CharacteristicInfos{}, err
		}
		infos = CharacteristicInfos{
			ById:       map[string]model.Characteristic{},
			IdToRootId: map[string]string{},
		}
		for _, c := range characteristics {
			infos.ById[c.Id] = c
			for _, child := range listCharacteristicDescendents(c.SubCharacteristics) {
				infos.ById[child.Id] = child
				infos.IdToRootId[child.Id] = c.Id
			}
		}
		return infos, nil
	}, cache.NoValidation, this.exp)
}

func listCharacteristicDescendents(characteristics []models.Characteristic) (result []models.Characteristic) {
	for _, c := range characteristics {
		result = append(result, c)
		result = append(result, listCharacteristicDescendents(c.SubCharacteristics)...)
	}
	return result
}

/*
func (this *CachedConceptRepository) GetAspectNode(id string) (result model.AspectNode, err error) {
	return cache.Use(this.cache, "aspect-node:"+id, func() (model.AspectNode, error) {
		result, err, _ := this.client.GetAspectNode(id)
		return result, err
	}, cache.NoValidation, this.exp)
}

func (this *CachedConceptRepository) GetConceptsOfCharacteristic(characteristicId string) (conceptIds []string, err error) {
	infos, err := this.GetConceptsInfos()
	if err != nil {
		return []string{}, err
	}
	result, ok := infos.ConceptIdsByCharacteristicId[characteristicId]
	if !ok {
		return []string{}, errors.New("characteristic not found")
	}
	return result, nil
}


func (this *CachedConceptRepository) GetRootCharacteristics(ids []marshaller.CharacteristicId) (result []marshaller.CharacteristicId) {
	infos, err := this.GetCharacteristicInfos()
	if err != nil {
		log.Println("ERROR: unable to get characteristics in GetRootCharacteristics()", err)
		return []marshaller.CharacteristicId{}
	}
	for _, id := range ids {
		rootId, ok := infos.IdToRootId[id]
		if ok && rootId != "" {
			result = append(result)
		}
	}
	return result
}

func (this *CachedConceptRepository) GetCharacteristicsOfFunction(functionId string) (characteristicIds []string, err error) {
	return cache.Use(this.cache, "function-characteristics:"+functionId, func() ([]string, error) {
		f, err, _ := this.client.GetFunction(functionId)
		if err != nil {
			return nil, err
		}
		c, err, _ := this.client.GetConceptWithoutCharacteristics(f.ConceptId)
		if err != nil {
			return nil, err
		}
		return c.CharacteristicIds, err
	}, cache.NoValidation, this.exp)
}

*/
