package store

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/elastic/go-elasticsearch"
	_ "github.com/go-sql-driver/mysql"
)

const (
	USER_NAME = "root"
	PASS_WORD = "ghr246810"
	HOST      = "124.70.94.103"
	PORT      = "3306"
	DATABASE  = "bt"
	CHARSET   = "utf8"
)

var addresses = []string{"http://127.0.0.1:9200", "http://127.0.0.1:9201"}

type bt struct {
	Hashinfo string `db:"hashinfo"`
	Name     string `db:"name"`
	Length   int64  `db:"length"`
}

var MysqlDb *sql.DB
var MysqlDbErr error
var es *elasticsearch.Client

func init() {
	dbDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", USER_NAME, PASS_WORD, HOST, PORT, DATABASE)

	// 打开连接失败
	MysqlDb, MysqlDbErr = sql.Open("mysql", dbDSN)
	//defer MysqlDb.Close();
	if MysqlDbErr != nil {
		log.Println("dbDSN: " + dbDSN)
		panic("数据源配置不正确: " + MysqlDbErr.Error())
	}

	// 最大连接数
	MysqlDb.SetMaxOpenConns(100)
	// 闲置连接数
	MysqlDb.SetMaxIdleConns(20)
	// 最大连接周期
	MysqlDb.SetConnMaxLifetime(100 * time.Second)

	if MysqlDbErr = MysqlDb.Ping(); nil != MysqlDbErr {
		panic("数据库链接失败: " + MysqlDbErr.Error())
	}
	config := elasticsearch.Config{
		Addresses: addresses,
		// Username:  "",
		// Password:  "",
		// CloudID:   "",
		// APIKey:    "",
	}

	es, _ = elasticsearch.NewClient(config)
}

func insertBt(hashInfo string, name string, length int64) {
	if MysqlDb == nil {
		log.Println("DB is nil")
		return
	}
	result, err := MysqlDb.Exec("insert into bt(hashinfo, name, length) values(?,?,?)", hashInfo, name, length)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(result.LastInsertId())
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func Create(doc map[string]interface{}) {

	// new client
	// Create creates a new document in the index.
	// Returns a 409 response when a document with a same ID already exists in the index.
	var buf bytes.Buffer
	// doc := map[string]interface{}{
	// 	"title":   "你看到外面的世界是什么样的？",
	// 	"content": "外面的世界真的很精彩",
	// }
	if err := json.NewEncoder(&buf).Encode(doc); err != nil {
		failOnError(err, "Error encoding doc")
	}
	res, err := es.Create("demo", "esd", &buf, es.Create.WithDocumentType("doc"))
	if err != nil {
		failOnError(err, "Error create response")
	}
	defer res.Body.Close()
	fmt.Println(res.String())
}
