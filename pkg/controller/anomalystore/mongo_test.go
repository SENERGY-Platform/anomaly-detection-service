/*
 * Copyright 2026 InfAI (CC SES)
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
	"reflect"
	"strings"
	"testing"

	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/configuration"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
)

const replicaSetUrl = "mongodb://mongo-0.mongo:27017,mongo-1.mongo:27017/?replicaSet=rs0"

func checkUriAndReadConcern(t *testing.T, opts *options.ClientOptions, url string) {
	t.Helper()
	if err := opts.Validate(); err != nil {
		t.Fatal(err)
	}
	if opts.GetURI() != url {
		t.Errorf("expected uri %q, got %q", url, opts.GetURI())
	}
	if opts.ReadConcern == nil || opts.ReadConcern.Level != readconcern.Majority().Level {
		t.Errorf("expected read concern majority, got %#v", opts.ReadConcern)
	}
}

func TestClientOptionsWithUser(t *testing.T) {
	conf := configuration.Config{
		MongoUrl:        replicaSetUrl,
		MongoUser:       "anomaly-detection",
		MongoPassword:   "p@ss:w/rd",
		MongoAuthSource: "admin",
	}
	opts := clientOptions(conf)
	checkUriAndReadConcern(t, opts, replicaSetUrl)
	if opts.Auth == nil {
		t.Fatal("expected auth to be set")
	}
	expected := options.Credential{Username: "anomaly-detection", Password: "p@ss:w/rd", AuthSource: "admin"}
	if !reflect.DeepEqual(*opts.Auth, expected) {
		t.Errorf("unexpected credential: user=%q source=%q mechanism=%q passwordMatches=%v", opts.Auth.Username, opts.Auth.AuthSource, opts.Auth.AuthMechanism, opts.Auth.Password == expected.Password)
	}
}

func TestClientOptionsWithoutUser(t *testing.T) {
	conf := configuration.Config{
		MongoUrl:        replicaSetUrl,
		MongoAuthSource: "admin",
	}
	opts := clientOptions(conf)
	checkUriAndReadConcern(t, opts, replicaSetUrl)
	if opts.Auth != nil {
		t.Errorf("expected no auth, got user=%q source=%q", opts.Auth.Username, opts.Auth.AuthSource)
	}
}

func TestClientOptionsPasswordWithoutUser(t *testing.T) {
	conf := configuration.Config{
		MongoUrl:        replicaSetUrl,
		MongoPassword:   "orphaned",
		MongoAuthSource: "admin",
	}
	opts := clientOptions(conf)
	checkUriAndReadConcern(t, opts, replicaSetUrl)
	if opts.Auth != nil {
		t.Errorf("expected no auth without user, got user=%q source=%q", opts.Auth.Username, opts.Auth.AuthSource)
	}
}

func TestClientOptionsUserReplacesUriCredentials(t *testing.T) {
	url := "mongodb://uri-user:uri-pw@localhost:27017/?authSource=uri-db&authMechanism=SCRAM-SHA-1"
	conf := configuration.Config{
		MongoUrl:        url,
		MongoUser:       "anomaly-detection",
		MongoPassword:   "config-pw",
		MongoAuthSource: "admin",
	}
	opts := clientOptions(conf)
	checkUriAndReadConcern(t, opts, url)
	expected := options.Credential{Username: "anomaly-detection", Password: "config-pw", AuthSource: "admin"}
	if opts.Auth == nil {
		t.Fatal("expected auth to be set")
	}
	if !reflect.DeepEqual(*opts.Auth, expected) {
		t.Errorf("expected config credentials to replace uri credentials, got user=%q source=%q passwordMatches=%v", opts.Auth.Username, opts.Auth.AuthSource, opts.Auth.Password == expected.Password)
	}
}

func TestAnomalyCollectionUsesConfiguredDatabase(t *testing.T) {
	conf := configuration.Config{
		MongoUrl:               "mongodb://localhost:27017",
		MongoDatabase:          "database_under_test",
		MongoAnomalyCollection: "collection_under_test",
	}
	// Connect does not contact the server, so this needs no running MongoDB.
	c, err := mongo.Connect(context.Background(), clientOptions(conf))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Disconnect(context.Background())
	collection := (&Mongo{config: conf, client: c}).anomalyCollection()
	if collection.Database().Name() != "database_under_test" {
		t.Errorf("expected database %q, got %q", "database_under_test", collection.Database().Name())
	}
	if collection.Name() != "collection_under_test" {
		t.Errorf("expected collection %q, got %q", "collection_under_test", collection.Name())
	}
}

func TestValidateConfig(t *testing.T) {
	valid := configuration.Config{
		MongoUrl:        replicaSetUrl,
		MongoUser:       "anomaly-detection",
		MongoPassword:   "pw",
		MongoAuthSource: "admin",
		MongoDatabase:   "anomaly_detection",
	}
	cases := map[string]struct {
		change  func(conf *configuration.Config)
		wantErr bool
	}{
		"valid with user":          {change: func(conf *configuration.Config) {}, wantErr: false},
		"valid without user":       {change: func(conf *configuration.Config) { conf.MongoUser = ""; conf.MongoPassword = "" }, wantErr: false},
		"empty database":           {change: func(conf *configuration.Config) { conf.MongoDatabase = "" }, wantErr: true},
		"blank database":           {change: func(conf *configuration.Config) { conf.MongoDatabase = " " }, wantErr: true},
		"user with empty password": {change: func(conf *configuration.Config) { conf.MongoPassword = "" }, wantErr: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			conf := valid
			tc.change(&conf)
			err := validateConfig(conf)
			if tc.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewRejectsInvalidConfig(t *testing.T) {
	url := "mongodb://127.0.0.1:1/?serverSelectionTimeoutMS=300&connectTimeoutMS=300"
	cases := map[string]struct {
		conf    configuration.Config
		wantErr string
	}{
		"empty database":           {conf: configuration.Config{MongoUrl: url, MongoDatabase: ""}, wantErr: "MONGO_DATABASE"},
		"user with empty password": {conf: configuration.Config{MongoUrl: url, MongoDatabase: "anomaly_detection", MongoUser: "anomaly-detection"}, wantErr: "MONGO_PASSWORD"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			db, err := New(tc.conf)
			if err == nil {
				db.Disconnect()
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error naming %v, got: %v", tc.wantErr, err)
			}
		})
	}
}

func TestNewFailsWhenServerUnreachable(t *testing.T) {
	// Port 1 refuses connections; the short selection timeout keeps the test fast without a server.
	conf := configuration.Config{
		MongoUrl:      "mongodb://127.0.0.1:1/?serverSelectionTimeoutMS=300&connectTimeoutMS=300",
		MongoDatabase: "anomaly_detection",
	}
	db, err := New(conf)
	if err == nil {
		db.Disconnect()
		t.Fatal("expected error for unreachable server")
	}
	if !strings.Contains(err.Error(), "mongo startup check failed") {
		t.Errorf("unexpected error: %v", err)
	}
}
