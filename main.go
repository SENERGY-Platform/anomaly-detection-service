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

package main

import (
	"context"
	"flag"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg"
	"github.com/SENERGY-Platform/anomaly-detection-service/pkg/configuration"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	configLocation := flag.String("config", "config.json", "configuration file")
	flag.Parse()

	conf, err := configuration.Load(*configLocation)
	if err != nil {
		log.Fatal("ERROR: unable to load config", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	err = pkg.Start(ctx, wg, conf)
	if err != nil {
		log.Fatal(err)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	sig := <-shutdown
	log.Println("received shutdown signal", sig)
	cancel()
	wg.Wait() //wait for clean disconnects
}
