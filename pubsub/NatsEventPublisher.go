/*
 * Copyright (c) 2024. Devtron Inc.
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
 */

package pubsub

import (
	"github.com/devtron-labs/ci-runner/util"
	pubsub1 "github.com/devtron-labs/common-lib/pubsub-lib"
	"github.com/devtron-labs/common-lib/utils"
	"log"
)

type PubSubConfig struct {
	ImageScannerEndpoint string `env:"IMAGE_SCANNER_ENDPOINT" envDefault:"http://image-scanner-new-demo-devtroncd-service.devtroncd:80"`
}

func PublishEventsOnNats(jsonBody []byte, topic string) error {
	logger, err := utils.NewSugardLogger()
	if err != nil || logger == nil {
		log.Print(util.DEVTRON, "err", err)
		return err
	}
	client, err := pubsub1.NewPubSubClientServiceImpl(logger)
	if client == nil {
		log.Print(util.DEVTRON, "err", err)
		return err
	}
	err = client.Publish(topic, string(jsonBody))
	if err != nil {
		log.Print(util.DEVTRON, "error in publishing event to pubsub client", "topic", topic, "body", string(jsonBody))
	} else {
		log.Print(util.DEVTRON, "ci complete event notification done", "eventBody", string(jsonBody))
	}

	log.Print(util.DEVTRON, " housekeeping done. exiting now")
	return nil

}
