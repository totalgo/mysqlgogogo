package server

import (
	"encoding/json"
	"github.com/pingcap/parser/ast"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
)

type Schema struct {
	Name string
	Db   *leveldb.DB
}

func (s *Schema) _createTable(stmt *ast.CreateTableStmt) error {
	x, _ := json.Marshal(stmt)
	log.Printf("%s\n", string(x))

	return nil
}
