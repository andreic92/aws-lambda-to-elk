package elastic

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/olivere/elastic"
)

const mapping = `
{
	"mappings":{
		"logs":{
			"properties":{
				"message":{
					"type":"text"
				},
				"date":{
					"type":"date"
				},
				"cmdName":{
					"type":"keyword"
				},
				"cmdLine":{
					"type":"keyword"
				},
				"hostname":{
					"type":"keyword"
				},
				"transport":{
					"type":"keyword"
				},
				"priority":{
					"type":"keyword"
				}
			}
		}
	}
}`

type ElasticSearchClient interface {
	AddEvent(Event) error
}

func NewClient(esURL string, esPort string, indexName string) (ElasticSearchClient, error) {
	url := fmt.Sprintf("%s:%s", strings.Trim(esURL, "/"), esPort)
	fmt.Println(url)
	client, err := elastic.NewClient(
		elastic.SetURL(url),
		elastic.SetSniff(false),
	)
	if err != nil {
		// Handle error
		return nil, err
	}
	if err := initializeElasticSearch(client, url, indexName); err != nil {
		return nil, err
	}
	return &defaultElasticClient{
		client:    client,
		indexName: indexName,
	}, nil
}

type defaultElasticClient struct {
	client    *elastic.Client
	indexName string
}

func initializeElasticSearch(client *elastic.Client, url string, indexName string) error {
	ctx := context.Background()

	// Ping the Elasticsearch server to get e.g. the version number
	_, _, err := client.Ping(url).Do(ctx)
	if err != nil {
		// Handle error
		return err
	}

	// Getting the ES version number is quite common, so there's a shortcut
	_, err = client.ElasticsearchVersion(url)
	if err != nil {
		// Handle error
		return err
	}

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists(indexName).Do(ctx)
	if err != nil {
		// Handle error
		return err
	}
	if !exists {
		// Create a new index.
		_, err := client.CreateIndex(indexName).BodyString(mapping).Do(ctx)
		if err != nil {
			// Handle error
			return err
		}
	}
	return nil
}

func (d *defaultElasticClient) AddEvent(ev Event) error {
	uuid, err := uuid.NewV4()
	if err != nil {
		return err
	}

	ctx := context.Background()
	_, err = d.client.Index().
		Index(d.indexName).
		Type("log").
		Id(uuid.String()).
		BodyJson(ev).
		Do(ctx)
	if err != nil {
		return err
	}
	return nil
}

type Event struct {
	CmdName   string `json:"cmdName"`
	CmdLine   string `json:"cmdLine"`
	Hostname  string `json:"hostname"`
	Transport string `json:"transport"`
	Priority  string `json:"priority"`
	Message   string `json:"message"`
	Date      string `json:"date"`
}
