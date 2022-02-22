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

	"github.com/devtron-labs/ci-runner/util"
	"github.com/nats-io/nats.go"
)

func PublishEventsOnNats(jsonBody []byte, topic string) error {
	client, err := NewPubSubClient()
	if err != nil {
		log.Println(util.DEVTRON, "err", err)
		os.Exit(1)
	}

	var reqBody = []byte(jsonBody)

	streamInfo, err := client.JetStrCtxt.StreamInfo(topic)
	if err != nil {
		client.Logger.Errorw("Error while getting stream info", "topic", topic, "error", err)
	}
	if streamInfo == nil {
		//Stream doesn't already exist. Create a new stream from jetStreamContext
		_, error := client.JetStrCtxt.AddStream(&nats.StreamConfig{
			Name:     topic,
			Subjects: []string{topic + ".*"},
		})
		if error != nil {
			client.Logger.Errorw("Error while creating stream", "topic", topic, "error", error)
		}
	}

	//Generate random string for passing as Header Id in message
	randString := "MsgHeaderId-" + util.Generate(10)
	_, err = client.JetStrCtxt.Publish(topic, reqBody, nats.MsgId(randString))
	if err != nil {
		client.Logger.Errorw("Error while publishing Request", "topic", topic, "body", string(reqBody), "err", err)
	}

	err = client.Conn.Publish(topic, reqBody)
	if err != nil {
		log.Println(util.DEVTRON, "publish err", "err", err)
		return err
	}
	log.Println(util.DEVTRON, "ci complete event notification done")

	err = client.Conn.Drain()
	if err != nil {
		log.Println(util.DEVTRON, " error in closing nats", "err", err)
	}

	log.Println(util.DEVTRON, " housekeeping done. exiting now")
	return nil
}
