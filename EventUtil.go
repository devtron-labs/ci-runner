package main

import (
	"encoding/json"
	"github.com/caarlos0/env"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func SendEvents(ciRequest *CiRequest, digest string, image string) error {
	client, err := NewPubSubClient()
	if err != nil {
		log.Println("err", err)
		os.Exit(1)
	}
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
	err = SendCiCompleteEvent(client, event)
	nc := client.Conn.NatsConn()

	err = client.Conn.Close()
	if err != nil {
		log.Println(devtron, " error in closing stan", "err", err)
	}

	err = nc.Drain()
	if err != nil {
		log.Println(devtron," error in draining nats", "err", err)
	}
	nc.Close()
	log.Println(devtron," housekeeping done. exiting now")
	return err
}

func NewPubSubClient() (*PubSubClient, error) {
	cfg := &PubSubConfig{}
	err := env.Parse(cfg)
	if err != nil {
		return &PubSubClient{}, err
	}
	nc, err := nats.Connect(cfg.NatsServerHost)
	if err != nil {
		log.Println("err", err)
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

func SendCiCompleteEvent(client *PubSubClient, event CiCompleteEvent) error {
	jsonBody, err := json.Marshal(event)
	if err != nil {
		log.Println(devtron, "err", err)
		return err
	}
	var reqBody = []byte(jsonBody)
	err = client.Conn.Publish(CI_COMPLETE_TOPIC, reqBody) // does not return until an ack has been received from NATS Streaming
	if err != nil {
		log.Println(devtron, "publish err", "err", err)
		return err
	}
	log.Println(devtron, "ci complete event notification done")
	return nil
}
