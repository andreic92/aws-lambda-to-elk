package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/sirupsen/logrus"
	"github.com/yottta/aws-lambda-to-elk/elastic"
)

const (
	expectedCmdName = "dockerd"

	envESHost  = "ES_HOST"
	envESPort  = "ES_PORT"
	envESIndex = "ES_INDEX"
)

var client elastic.ElasticSearchClient

func HandleRequest(ctx context.Context, ev CWEvent) error {
	if ev.Logs == nil {
		return fmt.Errorf("non existing logs in the received event: %v", ev)
	}

	if len(ev.Logs.Data) == 0 {
		return fmt.Errorf("no logs data in the reiceved event: %v", ev)
	}

	data, err := base64.StdEncoding.DecodeString(ev.Logs.Data)
	if err != nil {
		return err
	}

	var buf2 bytes.Buffer
	err = gunzipWrite(&buf2, data)
	if err != nil {
		log.WithError(err).WithField("EncodedEvent", ev.Logs.Data).Error("Error during unziping the event")
	}

	var logEvents DecodedEvent
	err = json.Unmarshal(buf2.Bytes(), &logEvents)
	if err != nil {
		return err
	}

	for _, event := range logEvents.Events {
		var myEvent elastic.Event
		err = json.Unmarshal([]byte(event.Message), &myEvent)
		if err != nil {
			return err
		}
		if myEvent.CmdName != expectedCmdName {
			return fmt.Errorf("Invalid cmdLine %s", myEvent.CmdLine)
		}

		unixTimeUTC := time.Unix(event.Timestamp/1000, 0)
		myEvent.Date = unixTimeUTC.Format(time.RFC3339)

		if err := client.AddEvent(myEvent); err != nil {
			log.WithError(err).Error("Error during adding event to ES")
		}
	}

	return nil
}

// Write gunzipped data to a Writer
func gunzipWrite(w io.Writer, data []byte) error {
	// Write gzipped data to the client
	gr, err := gzip.NewReader(bytes.NewBuffer(data))
	defer gr.Close()
	data, err = ioutil.ReadAll(gr)
	if err != nil {
		return err
	}
	w.Write(data)
	return nil
}

func main() {
	elasticHost := os.Getenv(envESHost)
	elasticPort := os.Getenv(envESPort)
	elasticIndex := os.Getenv(envESIndex)

	esClient, err := elastic.NewClient(elasticHost, elasticPort, elasticIndex)
	if err != nil {
		panic(err)
	}
	client = esClient
	lambda.Start(HandleRequest)
}

type CWEvent struct {
	Logs *AWSLogs `json:"awslogs"`
}

type AWSLogs struct {
	Data string `json:"data"`
}

type DecodedEvent struct {
	Events []LogEvent `json:"logEvents"`
}
type LogEvent struct {
	ID        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}
