package store

import (
	"database/sql"
	"fmt"
	"log"
	"time"

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

type bt struct {
	Hashinfo string `db:"hashinfo"`
	Name     string `db:"name"`
	Length   int64  `db:"length"`
}

var MysqlDb *sql.DB
var MysqlDbErr error

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
