package main

import (
	"github.com/flike/kingshard/sqlparser"
	"log"
)

func main() {
	//testParse("show tables;")
	TestMixer()
}

func testParse(sql string) {
	_, err := sqlparser.Parse(sql)
	if err != nil {
		log.Println(sql)
		log.Println(err)
	}

}

func TestSet() {
	sql := "set names gbk"
	testParse(sql)
}

func TestSimpleSelect() {
	sql := "select last_insert_id() as a"
	testParse(sql)
}

func TestMixer() {
	sql := ``

	sql = "show databases"
	testParse(sql)

	sql = "show tables from abc"
	testParse(sql)

	sql = "show tables from abc like a"
	testParse(sql)

	sql = "show tables from abc where a = 1"
	testParse(sql)

	sql = "show proxy abc"
	testParse(sql)
}
