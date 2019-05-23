package sqllayer

import (
	"encoding/json"
	"strings"
)

func BuildKey(schema, table string, column, pk string) []byte {
	//构建key,用来映射到底层数据库
	return []byte(
		strings.Join([]string{
			"data", schema, "table", table, "column", column, "pk", pk,
		}, "%"))
}

func BuildUniqueIndexKey(schema, table string, column, data string) []byte {
	//构建唯一索引的key，参照tidb，
	return []byte(
		strings.Join([]string{
			"unique", schema, "table", table, "column", column, "data", data,
		}, "%"))
}

func BuildIndexKey(schema, table string, column, data, pk string) []byte {
	return []byte(
		strings.Join([]string{
			"unique", schema, "table", table, "column", column, "data", data,
		}, "%"))
}

func BuildValue(data interface{}) []byte {
	d, _ := json.Marshal(data)
	return d
}
