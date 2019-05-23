package sqllayer

import "github.com/totalgo/mysqlgogogo/engine"

type SQLLayer struct {
	db *engine.DB
}

func NewLayer(db *engine.DB) *SQLLayer {
	s := &SQLLayer{db: db}
	return s
}

