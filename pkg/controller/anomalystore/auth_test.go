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
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/configuration"
	"github.com/SENERGY-Platform/go-service-base/config-hdl/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func randomHex(t *testing.T) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}

// TestNewAuthenticatedStartupCheck needs root credentials of a throwaway server with access control,
// given by MONGO_AUTH_TEST_URL, _USER and _PASSWORD; it creates and drops its own users and databases.
func TestNewAuthenticatedStartupCheck(t *testing.T) {
	url, rootUser, rootPassword := os.Getenv("MONGO_AUTH_TEST_URL"), os.Getenv("MONGO_AUTH_TEST_USER"), os.Getenv("MONGO_AUTH_TEST_PASSWORD")
	if testing.Short() || url == "" || rootUser == "" || rootPassword == "" {
		t.Skip("needs MONGO_AUTH_TEST_URL, MONGO_AUTH_TEST_USER and MONGO_AUTH_TEST_PASSWORD, not in -short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	root, err := mongo.Connect(ctx, options.Client().ApplyURI(url).SetAuth(options.Credential{Username: rootUser, Password: rootPassword, AuthSource: "admin"}))
	if err != nil {
		t.Fatal(err)
	}
	// Registered first so it runs after the user cleanups, which need the connection.
	t.Cleanup(func() { root.Disconnect(context.Background()) })

	suffix := randomHex(t)
	database, otherDatabase := "anomaly_detection_authtest_"+suffix, "other_authtest_"+suffix
	user, otherUser := "anomaly-detection-"+suffix, "other-"+suffix
	password, otherPassword := randomHex(t), randomHex(t)
	createUser := func(name string, pw string, db string) {
		err := root.Database("admin").RunCommand(ctx, bson.D{
			{Key: "createUser", Value: name},
			{Key: "pwd", Value: pw},
			{Key: "roles", Value: bson.A{bson.D{{Key: "role", Value: "readWrite"}, {Key: "db", Value: db}}}},
		}).Err()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := root.Database("admin").RunCommand(context.Background(), bson.D{{Key: "dropUser", Value: name}}).Err(); err != nil {
				t.Errorf("unable to drop user %v: %v", name, err)
			}
			if err := root.Database(db).Drop(context.Background()); err != nil {
				t.Errorf("unable to drop database %v: %v", db, err)
			}
		})
	}
	createUser(user, password, database)
	createUser(otherUser, otherPassword, otherDatabase)

	cases := []struct {
		name     string
		user     string
		password string
		wantErr  bool
	}{
		{"correct user", user, password, false},
		{"no credentials", "", "", true},
		{"user of another database", otherUser, otherPassword, true},
		{"wrong password", user, password + "-wrong", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, err := New(configuration.Config{
				MongoUrl:               url,
				MongoUser:              c.user,
				MongoPassword:          types.Secret(c.password),
				MongoAuthSource:        "admin",
				MongoDatabase:          database,
				MongoAnomalyCollection: "anomalies",
			})
			if err == nil {
				defer db.client.Disconnect(context.Background())
			}
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if err != nil {
				for _, pw := range []string{password, otherPassword, rootPassword} {
					if strings.Contains(err.Error(), pw) {
						t.Fatal("error contains a password")
					}
				}
				if !strings.Contains(err.Error(), "mongo startup check failed") {
					t.Errorf("unexpected error: %v", err)
				}
				t.Log(err)
				return
			}
			if err = db.StoreAnomaly("test", "device", "service", "desc", 1); err != nil {
				t.Errorf("write after successful startup failed: %v", err)
			}
		})
	}
}
