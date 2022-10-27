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
	"github.com/devtron-labs/ci-runner/util"
	pubsub1 "github.com/devtron-labs/common-lib/pubsub-lib"
	"github.com/devtron-labs/common-lib/utils"
	//"go.uber.org/zap"
	"log"
	"os"
)

// type NatsEventPublisher interface {
// 	PublishEventsOnNats(jsonBody []byte, topic string) error
// }

// func NewNatsEventPublisherImpl(logger *zap.SugaredLogger, pubSubClient *PubSubClient) *NatsEventPublisherImpl {
// 	return &NatsEventPublisherImpl{
// 		logger:       logger,
// 		pubSubClient: pubSubClient,
// 	}
// }

// type NatsEventPublisherImpl struct {
// 	logger       *zap.SugaredLogger
// 	pubSubClient *PubSubClient
// }

// func (impl *NatsEventPublisherImpl) PublishEventsOnNats(jsonBody []byte, topic string) error {

// 	err := AddStream(impl.pubSubClient.JetStrCtxt, CI_RUNNER_STREAM)

// 	if err != nil {
// 		impl.logger.Errorw("Error while adding stream", "error", err)
// 	}
// 	//Generate random string for passing as Header Id in message
// 	randString := "MsgHeaderId-" + util.Generate(10)
// 	_, err = impl.pubSubClient.JetStrCtxt.Publish(topic, jsonBody, nats.MsgId(randString))
// 	if err != nil {
// 		impl.logger.Errorw("Error while publishing Request", "topic", topic, "body", string(jsonBody), "err", err)
// 	}

// 	impl.logger.Info(util.DEVTRON, "ci complete event notification done")

// 	//Drain the connection
// 	err = impl.pubSubClient.Conn.Drain()

// 	if err != nil {
// 		impl.logger.Errorw("Error while draining the connection", "error", err)
// 	}

// 	impl.logger.Info(util.DEVTRON, " housekeeping done. exiting now")
// 	return nil
// }

func PublishEventsOnNats(jsonBody []byte, topic string) error {
	logger, err := utils.NewSugardLogger()
	if err != nil {
		log.Fatal(util.DEVTRON, "err", err)
		os.Exit(1)
	}
	client := pubsub1.NewPubSubClientServiceImpl(logger)
	if client == nil {
		log.Fatal(util.DEVTRON, "err", err)
		os.Exit(1)
	}
	err = client.Publish(topic, string(jsonBody))
	if err != nil {
		log.Print(util.DEVTRON, "error in publishing event to pubsub client", "topic", topic, "body", string(jsonBody))
	} else {
		log.Print(util.DEVTRON, "ci complete event notification done")
	}
	//Drain the connection
	if client.NatsClient != nil {
		err = client.NatsClient.Conn.Drain()
		if err != nil {
			log.Fatal("Error while draining the connection", "error", err)
		}
	}

	log.Print(util.DEVTRON, " housekeeping done. exiting now")
	return nil

}
