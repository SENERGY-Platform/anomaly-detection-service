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
	"context"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/configuration"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"log"
	"reflect"
	"runtime/debug"
	"slices"
	"strings"
	"time"
)

type Mongo struct {
	config configuration.Config
	client *mongo.Client
}

var CreateCollections = []func(db *Mongo) error{}

func getTimeoutContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	return ctx
}

func New(conf configuration.Config) (*Mongo, error) {
	c, err := mongo.Connect(getTimeoutContext(), options.Client().ApplyURI(conf.MongoUrl), options.Client().SetReadConcern(readconcern.Majority()))
	if err != nil {
		return nil, err
	}
	db := &Mongo{config: conf, client: c}
	for _, creators := range CreateCollections {
		err = creators(db)
		if err != nil {
			c.Disconnect(context.Background())
			return nil, err
		}
	}
	return db, nil
}

func (this *Mongo) Disconnect() {
	timeout, _ := context.WithTimeout(context.Background(), 10*time.Second)
	log.Println(this.client.Disconnect(timeout))
}

func getBsonFieldObject[T any]() T {
	v := new(T)
	err := fillObjectWithItsBsonFieldNames(v, nil, nil)
	if err != nil {
		panic(err)
	}
	return *v
}

func fillObjectWithItsBsonFieldNames(ptr interface{}, prefix []string, done []string) error {
	ptrval := reflect.ValueOf(ptr)
	objval := reflect.Indirect(ptrval)
	objecttype := objval.Type()
	objTypeStr := objecttype.Name()
	if slices.Contains(done, objTypeStr) {
		return nil
	}
	done = append(done, objTypeStr)
	for i := 0; i < objecttype.NumField(); i++ {
		field := objecttype.Field(i)
		if field.Type.Kind() == reflect.String {
			tags, err := bsoncodec.DefaultStructTagParser.ParseStructTags(field)
			if err != nil {
				return err
			}
			objval.Field(i).SetString(strings.Join(append(prefix, tags.Name), "."))
		}
		if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.String {
			tags, err := bsoncodec.DefaultStructTagParser.ParseStructTags(field)
			if err != nil {
				return err
			}
			objval.Field(i).Set(reflect.ValueOf([]string{strings.Join(append(prefix, tags.Name), ".")}))
		}
		if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Struct {
			tags, err := bsoncodec.DefaultStructTagParser.ParseStructTags(field)
			if err != nil {
				return err
			}
			element := reflect.New(objval.Field(i).Type().Elem())
			if tags.Inline {
				err = fillObjectWithItsBsonFieldNames(element.Interface(), prefix, done)
			} else {
				err = fillObjectWithItsBsonFieldNames(element.Interface(), append(prefix, tags.Name), done)
			}
			if err != nil {
				return err
			}
			list := reflect.New(objval.Field(i).Type())
			list = reflect.Append(list.Elem(), element.Elem())
			objval.Field(i).Set(list)
		}
		if field.Type.Kind() == reflect.Struct {
			tags, err := bsoncodec.DefaultStructTagParser.ParseStructTags(field)
			if err != nil {
				return err
			}
			if tags.Inline {
				err = fillObjectWithItsBsonFieldNames(objval.Field(i).Addr().Interface(), prefix, done)
			} else {
				err = fillObjectWithItsBsonFieldNames(objval.Field(i).Addr().Interface(), append(prefix, tags.Name), done)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (this *Mongo) ensureIndex(collection *mongo.Collection, indexname string, indexKey string, asc bool, unique bool) error {
	var direction int32 = -1
	if asc {
		direction = 1
	}
	_, err := collection.Indexes().CreateOne(getTimeoutContext(), mongo.IndexModel{
		Keys:    bson.D{{indexKey, direction}},
		Options: options.Index().SetName(indexname).SetUnique(unique),
	})
	if err != nil {
		debug.PrintStack()
	}
	return err
}

func (this *Mongo) ensureCompoundIndex(collection *mongo.Collection, indexname string, asc bool, unique bool, indexKeys ...string) error {
	var direction int32 = -1
	if asc {
		direction = 1
	}
	keys := []bson.E{}
	for _, key := range indexKeys {
		keys = append(keys, bson.E{Key: key, Value: direction})
	}
	_, err := collection.Indexes().CreateOne(getTimeoutContext(), mongo.IndexModel{
		Keys:    bson.D(keys),
		Options: options.Index().SetName(indexname).SetUnique(unique),
	})
	return err
}
