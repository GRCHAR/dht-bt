package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"

	"github.com/elastic/go-elasticsearch/v8"
	_ "github.com/go-sql-driver/mysql"
)

var addresses = []string{"http://124.70.94.103:9200"}

var es *elasticsearch.Client

func init() {

	config := elasticsearch.Config{
		Addresses: addresses,
		// Username:  "",
		// Password:  "",
		// CloudID:   "",
		// APIKey:    "",
	}

	locales, err := elasticsearch.NewClient(config)
	if err != nil {
		log.Println("connect es fail!", err)
		return
	}
	es = locales
	log.Println("connect es success")

}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func Create(hashInfo string, name string, length int64, files []interface{}) {

	// new client
	// Create creates a new document in the index.
	// Returns a 409 response when a document with a same ID already exists in the index.
	var buf bytes.Buffer
	doc := map[string]interface{}{
		"hashInfo": hashInfo,
		"name":     name,
		"length":   length,
		"files":    files,
	}
	if err := json.NewEncoder(&buf).Encode(doc); err != nil {
		failOnError(err, "Error encoding doc")
	}
	res, err := es.Index("bt", &buf)
	if err != nil {
		failOnError(err, "Error Index response")
	}
	defer res.Body.Close()
	fmt.Println(res.String())
}
