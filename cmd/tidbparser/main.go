package main

import (
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	_ "github.com/pingcap/tidb/types/parser_driver"
	"log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	p := parser.New()
	a, _, c := p.Parse("show databases;show tables;", "", "")
	if c != nil {
		log.Panicln(c)
	}
	for _, v := range a {
		switch v.(type) {
		case *ast.ShowStmt:
			v1 := v.(*ast.ShowStmt)
			switch v1.Tp {
			case ast.ShowDatabases:
				log.Println("show databases")
			case ast.ShowTables:
				log.Println("show tables")
			default:
				log.Println("unknown")
			}
		default:
			log.Println(v)
		}
	}
}
