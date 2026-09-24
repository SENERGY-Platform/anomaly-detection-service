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

package configuration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
)

var mongoEnvVars = []string{
	"MONGO_URL",
	"MONGO_USER",
	"MONGO_PASSWORD",
	"MONGO_AUTH_SOURCE",
	"MONGO_DATABASE",
	"MONGO_ANOMALY_COLLECTION",
}

// unsetMongoEnv removes the variables for the test; t.Setenv restores them afterwards.
func unsetMongoEnv(t *testing.T) {
	for _, key := range mongoEnvVars {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

func TestLoadMongoDefaults(t *testing.T) {
	unsetMongoEnv(t)
	conf, err := Load("../../config.json")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{
		"MongoUrl":               "mongodb://localhost:27017",
		"MongoUser":              "",
		"MongoPassword":          "",
		"MongoAuthSource":        "admin",
		"MongoDatabase":          "anomaly_detection",
		"MongoAnomalyCollection": "anomalies",
	}
	actual := map[string]string{
		"MongoUrl":               conf.MongoUrl,
		"MongoUser":              conf.MongoUser,
		"MongoPassword":          string(conf.MongoPassword),
		"MongoAuthSource":        conf.MongoAuthSource,
		"MongoDatabase":          conf.MongoDatabase,
		"MongoAnomalyCollection": conf.MongoAnomalyCollection,
	}
	for key, e := range expected {
		if actual[key] != e {
			t.Errorf("%v: expected %q, got %q", key, e, actual[key])
		}
	}
}

func TestLoadMongoFromEnv(t *testing.T) {
	unsetMongoEnv(t)
	t.Setenv("MONGO_URL", "mongodb://mongo-0.mongo:27017,mongo-1.mongo:27017/?replicaSet=rs0")
	t.Setenv("MONGO_USER", "anomaly-detection")
	t.Setenv("MONGO_PASSWORD", "p@ss:w/rd")
	t.Setenv("MONGO_AUTH_SOURCE", "users")
	t.Setenv("MONGO_DATABASE", "anomaly_detection_test")
	t.Setenv("MONGO_ANOMALY_COLLECTION", "anomalies_test")
	conf, err := Load("../../config.json")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{
		"MongoUrl":               "mongodb://mongo-0.mongo:27017,mongo-1.mongo:27017/?replicaSet=rs0",
		"MongoUser":              "anomaly-detection",
		"MongoPassword":          "p@ss:w/rd",
		"MongoAuthSource":        "users",
		"MongoDatabase":          "anomaly_detection_test",
		"MongoAnomalyCollection": "anomalies_test",
	}
	actual := map[string]string{
		"MongoUrl":               conf.MongoUrl,
		"MongoUser":              conf.MongoUser,
		"MongoPassword":          string(conf.MongoPassword),
		"MongoAuthSource":        conf.MongoAuthSource,
		"MongoDatabase":          conf.MongoDatabase,
		"MongoAnomalyCollection": conf.MongoAnomalyCollection,
	}
	for key, e := range expected {
		if actual[key] != e {
			t.Errorf("%v: expected %q, got %q", key, e, actual[key])
		}
	}
}

func TestMongoPasswordNotPrinted(t *testing.T) {
	const password = "must-not-appear-in-output"
	unsetMongoEnv(t)
	t.Setenv("MONGO_USER", "anomaly-detection")
	t.Setenv("MONGO_PASSWORD", password)
	conf, err := Load("../../config.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(conf.MongoPassword) != password {
		t.Fatalf("password not loaded, got %q", string(conf.MongoPassword))
	}

	jsonOut, err := json.Marshal(conf)
	if err != nil {
		t.Fatal(err)
	}
	slogJson := &bytes.Buffer{}
	slog.New(slog.NewJSONHandler(slogJson, nil)).Info("config", "config", conf)
	slogText := &bytes.Buffer{}
	slog.New(slog.NewTextHandler(slogText, nil)).Info("config", "config", conf)

	outputs := map[string]string{
		"fmt %v":    fmt.Sprintf("%v", conf),
		"fmt %+v":   fmt.Sprintf("%+v", conf),
		"json":      string(jsonOut),
		"slog json": slogJson.String(),
		"slog text": slogText.String(),
	}
	for name, out := range outputs {
		if !strings.Contains(out, "anomaly-detection") {
			t.Errorf("%v: expected output to contain the config, got %v", name, out)
		}
		if strings.Contains(out, password) {
			t.Errorf("%v: password leaked: %v", name, out)
		}
	}
}
