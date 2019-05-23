package server

import (
	"errors"
	"fmt"
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/types/parser_driver"
	_ "github.com/pingcap/tidb/types/parser_driver"
	"log"
	"reflect"
	"strconv"
)

func (c *ClientConn) handleUseDB(db string) (err error) {
	if v, ok := c.s.schemas[db]; ok {
		c.s.schema = v
		return nil
	}
	return errors.New(fmt.Sprintf("schema %s not exist", db))

}

func (c *ClientConn) handleQuery(sql string) (err error) {
	log.Println(sql)
	p := parser.New()
	stmts, _, err := p.Parse(sql, "", "")
	if err != nil {
		c.writeError(err)
	}

	for _, stmt := range stmts {
		switch stmt1 := stmt.(type) {
		case *ast.UseStmt:
			c.writeError(c.handleUseDB(stmt1.DBName))
		case *ast.ShowStmt:
			switch stmt1.Tp {
			case ast.ShowDatabases:
				//show databases;
				var schemas = [][]interface{}{}
				for _, v := range c.s.schemas {
					schemas = append(schemas, []interface{}{v.Name})
				}
				c.writeData([]string{"Databases"}, schemas)
			case ast.ShowVariables:
				data := [][]interface{}{}
				iter := c.s.schema.Db.NewIterator(nil, nil)
				data = append(data, []interface{}{"1", "2"})
				for iter.Next() {
					data1 := []interface{}{string(iter.Key()), string(iter.Value())}
					data = append(data, data1)
				}
				c.writeData([]string{"Key", "value"}, data)
			default:
				c.writeError(errors.New(fmt.Sprintf("%s statement not supported show", reflect.TypeOf(stmt))))
			}
		case *ast.SetStmt:
			log.Printf("%#v\n", stmt1.Variables[0].Value)
			if v, ok := stmt1.Variables[0].Value.(*driver.ValueExpr); ok {
				log.Println(stmt1.Variables[0].Name, v.Datum.GetString())
				switch v.Kind() {
				case types.KindInt64:
					c.s.schema.Db.Put([]byte(stmt1.Variables[0].Name), []byte(strconv.Itoa(int(v.Datum.GetInt64()))), nil)
				default:
					c.s.schema.Db.Put([]byte(stmt1.Variables[0].Name), []byte(v.Datum.GetString()), nil)
				}
			}
			c.writeOK(nil)
		case *ast.CreateDatabaseStmt:
			log.Println("do create database ", stmt1.Name)
			c.writeError(c.s._createDatabase(stmt1.Name))
		case *ast.CreateTableStmt:
			if c.s.schema == nil {
				c.writeError(errors.New(fmt.Sprintf("none database selected")))
			} else {
				c.writeError(c.s.schema._createTable(stmt1))
			}
		default:
			c.writeError(errors.New(fmt.Sprintf("%s statement not supported show", reflect.TypeOf(stmt))))
		}
	}
	return nil

	//switch v := stmt.(type) {
	//case *sqlparser.Show:
	//	c.writeError(errors.New("lalala"))
	//case *sqlparser.Select:
	//	c.writeOK(nil)
	//	return c.handleSelect(v, nil)
	//case *sqlparser.Insert:
	//	return c.handleExec(stmt, nil)
	//case *sqlparser.Update:
	//	return c.handleExec(stmt, nil)
	//case *sqlparser.Delete:
	//	return c.handleExec(stmt, nil)
	//case *sqlparser.Replace:
	//	return c.handleExec(stmt, nil)
	//case *sqlparser.Set:
	//	return c.handleSet(v, sql)
	//case *sqlparser.Begin:
	//	return c.handleBegin()
	//case *sqlparser.Commit:
	//	return c.handleCommit()
	//case *sqlparser.Rollback:
	//	return c.handleRollback()
	//case *sqlparser.Admin:
	//	if c.user == "root" {
	//		return c.handleAdmin(v)
	//	}
	//	return fmt.Errorf("statement %T not support now", stmt)
	//case *sqlparser.AdminHelp:
	//	if c.user == "root" {
	//		return c.handleAdminHelp(v)
	//	}
	//	return fmt.Errorf("statement %T not support now", stmt)
	//case *sqlparser.UseDB:
	//	return c.handleUseDB(v.DB)
	//case *sqlparser.SimpleSelect:
	//	return c.handleSimpleSelect(v)
	//case *sqlparser.Truncate:
	//	return c.handleExec(stmt, nil)
	//default:
	//	return fmt.Errorf("statement %T not support now", stmt)
	//}

}
