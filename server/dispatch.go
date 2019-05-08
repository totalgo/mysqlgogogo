package server

import (
	"fmt"
	"github.com/flike/kingshard/mysql"
	"log"
)

func (c *ClientConn) dispatch(data []byte) error {
	cmd := data[0]
	data = data[1:]

	log.Println(cmd, string(data))
	switch cmd {
	case mysql.COM_QUIT:
		//	c.handleRollback()
		c.Close()
		return nil
	case mysql.COM_QUERY:
		return c.handleQuery(string(data))
	case mysql.COM_PING:
		return c.writeOK(nil)
	case mysql.COM_INIT_DB:
		return c.writeError(c.handleUseDB(string(data)))
	//case mysql.COM_FIELD_LIST:
	//	return c.handleFieldList(data)
	//case mysql.COM_STMT_PREPARE:
	//	return c.handleStmtPrepare(hack.String(data))
	//case mysql.COM_STMT_EXECUTE:
	//	return c.handleStmtExecute(data)
	//case mysql.COM_STMT_CLOSE:
	//	return c.handleStmtClose(data)
	//case mysql.COM_STMT_SEND_LONG_DATA:
	//	return c.handleStmtSendLongData(data)
	//case mysql.COM_STMT_RESET:
	//	return c.handleStmtReset(data)
	case mysql.COM_SET_OPTION:
		return c.writeEOF(0)
	default:
		msg := fmt.Sprintf("你说你妈呢？command %d not supported now", cmd)
		return mysql.NewError(mysql.ER_UNKNOWN_ERROR, msg)
	}

	return nil
}
