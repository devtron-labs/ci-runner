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

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/caarlos0/env"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"gopkg.in/go-resty/resty.v2"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func SendCDEvent(cdRequest *CdRequest) error {

	event := CdStageCompleteEvent{
		CiProjectDetails: cdRequest.CiProjectDetails,
		CdPipelineId:     cdRequest.CdPipelineId,
		WorkflowId:       cdRequest.WorkflowId,
		WorkflowRunnerId: cdRequest.WorkflowRunnerId,
		CiArtifactDTO:    cdRequest.CiArtifactDTO,
		TriggeredBy:      cdRequest.TriggeredBy,
	}
	err := SendCdCompleteEvent(cdRequest, event)
	if err != nil {
		log.Println(devtron, "err", err)
		return err
	}
	return nil
}

func SendEvents(ciRequest *CiRequest, digest string, image string) error {

	event := CiCompleteEvent{
		CiProjectDetails: ciRequest.CiProjectDetails,
		DockerImage:      image,
		Digest:           digest,
		PipelineId:       ciRequest.PipelineId,
		PipelineName:     ciRequest.PipelineName,
		DataSource:       "CI-RUNNER",
		WorkflowId:       ciRequest.WorkflowId,
		TriggeredBy:      ciRequest.TriggeredBy,
		MaterialType:     "git",
	}
	err := SendCiCompleteEvent(event)
	if err != nil {
		log.Println(devtron, "err", err)
		return err
	}
	log.Println(devtron, " housekeeping done. exiting now")
	return nil
}

func NewPubSubClient() (*PubSubClient, error) {
	cfg := &PubSubConfig{}
	err := env.Parse(cfg)
	if err != nil {
		return &PubSubClient{}, err
	}
	nc, err := nats.Connect(cfg.NatsServerHost)
	if err != nil {
		log.Println(devtron, "err", err)
		os.Exit(1)
	}
	s := rand.NewSource(time.Now().UnixNano())
	uuid := rand.New(s)
	uniqueClienId := "CI-RUNNER-" + strconv.Itoa(uuid.Int())

	sc, err := stan.Connect(cfg.ClusterId, uniqueClienId, stan.NatsConn(nc))
	if err != nil {
		log.Fatal(devtron, "err", err)
	}
	natsClient := &PubSubClient{
		Conn: sc,
	}
	return natsClient, nil
}

func SendCiCompleteEvent(event CiCompleteEvent) error {
	jsonBody, err := json.Marshal(event)
	if err != nil {
		log.Println(devtron, "err", err)
		return err
	}
	err = PublishEvent(jsonBody, CI_COMPLETE_TOPIC)
	log.Println(devtron, "ci complete event notification done")
	return err
}

func SendCdCompleteEvent(cdRequest *CdRequest, event CdStageCompleteEvent) error {
	jsonBody, err := json.Marshal(event)
	if err != nil {
		log.Println(devtron, "err", err)
		return err
	}
	err = PublishCDEvent(jsonBody, CD_COMPLETE_TOPIC, cdRequest)
	log.Println(devtron, "cd stage complete event notification done")
	return err
}

func PublishCDEvent(jsonBody []byte, topic string, cdRequest *CdRequest) error {
	if cdRequest.IsExtRun {
		return PublishEventsOnRest(jsonBody, topic, cdRequest)
	}
	return PublishEventsOnNats(jsonBody, topic)
}

func PublishEvent(jsonBody []byte, topic string) error {
	return PublishEventsOnNats(jsonBody, topic)
}

func PublishEventsOnNats(jsonBody []byte, topic string) error {
	client, err := NewPubSubClient()
	if err != nil {
		log.Println(devtron, "err", err)
		os.Exit(1)
	}

	var reqBody = []byte(jsonBody)
	err = client.Conn.Publish(topic, reqBody)
	if err != nil {
		log.Println(devtron, "publish err", "err", err)
		return err
	}
	log.Println(devtron, "ci complete event notification done")
	nc := client.Conn.NatsConn()

	err = client.Conn.Close()
	if err != nil {
		log.Println(devtron, " error in closing stan", "err", err)
	}

	err = nc.Drain()
	if err != nil {
		log.Println(devtron, " error in draining nats", "err", err)
	}
	nc.Close()
	log.Println(devtron, " housekeeping done. exiting now")
	return nil
}

type PublishRequest struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

func PublishEventsOnRest(jsonBody []byte, topic string, cdRequest *CdRequest) error {
	publishRequest := &PublishRequest{
		Topic:   topic,
		Payload: jsonBody,
	}
	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	resp, err := client.SetRetryCount(4).R().
		SetHeader("Content-Type", "application/json").
		SetBody(publishRequest).
		SetAuthToken(cdRequest.OrchestratorToken).
		//SetResult().    // or SetResult(AuthSuccess{}).
		Post(cdRequest.OrchestratorHost)
	if err != nil {
		log.Println(devtron, "err in publishing over rest", err)
		return err
	}
	log.Println(devtron, "res ", string(resp.Body()))
	return nil
}

func SendEventToClairUtility(event *ScanEvent) error {
	jsonBody, err := json.Marshal(event)
	if err != nil {
		log.Println(devtron, "err", err)
		return err
	}
	/*
		err = PublishEventsOnNats(jsonBody, TOPIC_CI_SCAN)
		if err != nil {
			log.Println(devtron, "err", err)
			return err
		}
	*/

	cfg := &PubSubConfig{}
	err = env.Parse(cfg)
	if err != nil {
		return err
	}

	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	//client.SetDebug(true)
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(jsonBody).
		Post(fmt.Sprintf("%s/%s", cfg.ImageScannerEndpoint, "scanner/image"))
	if err != nil {
		log.Println(devtron, "err in image scanner app over rest", err)
		return err
	}
	log.Println(devtron, resp.StatusCode())
	log.Println(devtron, resp)
	return nil
}

type ScanEvent struct {
	Image            string `json:"image"`
	ImageDigest      string `json:"imageDigest"`
	AppId            int    `json:"appId"`
	EnvId            int    `json:"envId"`
	PipelineId       int    `json:"pipelineId"`
	CiArtifactId     int    `json:"ciArtifactId"`
	UserId           int    `json:"userId"`
	AccessKey        string `json:"accessKey"`
	SecretKey        string `json:"secretKey"`
	Token            string `json:"token"`
	AwsRegion        string `json:"awsRegion"`
	DockerRegistryId string `json:"dockerRegistryId"`
}
