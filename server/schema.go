package server

import (
	"github.com/pingcap/parser/ast"
	"github.com/syndtr/goleveldb/leveldb"
)

type Schema struct {
	Name string
	Db   *leveldb.DB
}

func (s *Schema) _createTable(stmt *ast.CreateTableStmt) error {
	return nil
}
