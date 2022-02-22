/*
 *  Copyright 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package pubsub

import (
	"log"
	"os"

	"github.com/caarlos0/env"
	"github.com/devtron-labs/ci-runner/util"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

const ImageScannerEndpoint string = "http://image-scanner-new-demo-devtroncd-service.devtroncd:80"

type PubSubClient struct {
	Logger     *zap.SugaredLogger
	JetStrCtxt nats.JetStreamContext
	Conn       *nats.Conn
}

type PubSubConfig struct {
	NatsServerHost       string `env:"NATS_SERVER_HOST" envDefault:"nats://localhost:4222"`
	ImageScannerEndpoint string `env:"IMAGE_SCANNER_ENDPOINT" envDefault:"http://image-scanner-new-demo-devtroncd-service.devtroncd:80"`
}

func NewPubSubClient() (*PubSubClient, error) {
	cfg := &PubSubConfig{}
	err := env.Parse(cfg)
	if err != nil {
		return &PubSubClient{}, err
	}
	nc, err := nats.Connect(cfg.NatsServerHost)
	if err != nil {
		log.Println(util.DEVTRON, "err", err)
		os.Exit(1)
	}

	//Create a jetstream context
	js, _ := nc.JetStream()
	natsClient := &PubSubClient{
		Conn:       nc,
		JetStrCtxt: js,
	}
	return natsClient, nil
}
