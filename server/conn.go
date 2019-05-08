package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/flike/kingshard/mysql"
	"log"
	"net"
	"strconv"
	"sync"
)

type ClientConn struct {
	sync.Mutex

	pkg *mysql.PacketIO

	c net.Conn

	s *Server

	capability uint32

	connectionId uint32

	status    uint16
	collation mysql.CollationId
	charset   string

	user string

	salt []byte

	closed bool
}

var DEFAULT_CAPABILITY uint32 = mysql.CLIENT_LONG_PASSWORD | mysql.CLIENT_LONG_FLAG |
	mysql.CLIENT_CONNECT_WITH_DB | mysql.CLIENT_PROTOCOL_41 |
	mysql.CLIENT_TRANSACTIONS | mysql.CLIENT_SECURE_CONNECTION

var baseConnId uint32 = 10000

func (c *ClientConn) Handshake() error {
	if err := c.writeInitialHandshake(); err != nil {
		log.Println(err)
		return err
	}

	if err := c.readHandshakeResponse(); err != nil {
		log.Println(err)
		return err
	}

	if err := c.writeOK(nil); err != nil {
		log.Println(err)
		return err
	}

	c.pkg.Sequence = 0
	return nil
}

func (c *ClientConn) Close() error {
	if c.closed {
		return nil
	}

	c.c.Close()

	c.closed = true

	return nil
}

func (c *ClientConn) writeInitialHandshake() error {
	data := make([]byte, 4, 128)

	//min version 10
	data = append(data, 10)

	//server version[00]
	data = append(data, mysql.ServerVersion...)
	data = append(data, 0)

	//connection id
	data = append(data, byte(c.connectionId), byte(c.connectionId>>8), byte(c.connectionId>>16), byte(c.connectionId>>24))

	//auth-plugin-data-part-1
	data = append(data, c.salt[0:8]...)

	//filter [00]
	data = append(data, 0)

	//capability flag lower 2 bytes, using default capability here
	data = append(data, byte(DEFAULT_CAPABILITY), byte(DEFAULT_CAPABILITY>>8))

	//charset, utf-8 default
	data = append(data, uint8(mysql.DEFAULT_COLLATION_ID))

	//status
	data = append(data, byte(c.status), byte(c.status>>8))

	//below 13 byte may not be used
	//capability flag upper 2 bytes, using default capability here
	data = append(data, byte(DEFAULT_CAPABILITY>>16), byte(DEFAULT_CAPABILITY>>24))

	//filter [0x15], for wireshark dump, value is 0x15
	data = append(data, 0x15)

	//reserved 10 [00]
	data = append(data, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)

	//auth-plugin-data-part-2
	data = append(data, c.salt[8:]...)

	//filter [00]
	data = append(data, 0)

	return c.writePacket(data)
}

func (c *ClientConn) readPacket() ([]byte, error) {
	return c.pkg.ReadPacket()
}

func (c *ClientConn) writePacket(data []byte) error {
	return c.pkg.WritePacket(data)
}

func (c *ClientConn) writePacketBatch(total, data []byte, direct bool) ([]byte, error) {
	return c.pkg.WritePacketBatch(total, data, direct)
}

func (c *ClientConn) readHandshakeResponse() error {
	data, err := c.readPacket()

	if err != nil {
		return err
	}

	pos := 0

	//capability
	c.capability = binary.LittleEndian.Uint32(data[:4])
	pos += 4

	//skip max packet size
	pos += 4

	//charset, skip, if you want to use another charset, use set names
	//c.collation = CollationId(data[pos])
	pos++

	//skip reserved 23[00]
	pos += 23

	//user name
	c.user = string(data[pos : pos+bytes.IndexByte(data[pos:], 0)])

	pos += len(c.user) + 1

	//auth length and auth
	authLen := int(data[pos])
	pos++
	auth := data[pos : pos+authLen]

	//check password
	checkAuth := mysql.CalcPassword(c.salt, []byte(c.user))
	if !bytes.Equal(auth, checkAuth) {
		return mysql.NewDefaultError(mysql.ER_ACCESS_DENIED_ERROR, c.user, c.c.RemoteAddr().String(), "Yes")
	}

	pos += authLen

	var db string
	if c.capability&mysql.CLIENT_CONNECT_WITH_DB > 0 {
		if len(data[pos:]) == 0 {
			return nil
		}

		db = string(data[pos : pos+bytes.IndexByte(data[pos:], 0)])
		c.handleUseDB(db)
	}
	//todo 这里还能取出来客户端版本之类的，但是我不想做了

	return nil
}

func (c *ClientConn) Run() {
	for {
		data, err := c.readPacket()

		if err != nil {
			return
		}

		if err := c.dispatch(data); err != nil {
			c.writeError(err)
			if err == mysql.ErrBadConn {
				c.Close()
			}
		}

		if c.closed {
			return
		}

		c.pkg.Sequence = 0
	}
}


func (c *ClientConn) buildResultset(fields []*mysql.Field, names []string, values [][]interface{}) (*mysql.Resultset, error) {
	var ExistFields bool
	r := new(mysql.Resultset)

	r.Fields = make([]*mysql.Field, len(names))
	r.FieldNames = make(map[string]int, len(names))

	//use the field def that get from true database
	if len(fields) != 0 {
		if len(r.Fields) == len(fields) {
			ExistFields = true
		} else {
			return nil, errors.New("invalid argument")
		}
	}

	var b []byte
	var err error

	for i, vs := range values {
		if len(vs) != len(r.Fields) {
			return nil, fmt.Errorf("row %d has %d column not equal %d", i, len(vs), len(r.Fields))
		}

		var row []byte
		for j, value := range vs {
			//列的定义
			if i == 0 {
				if ExistFields {
					r.Fields[j] = fields[j]
					r.FieldNames[string(r.Fields[j].Name)] = j
				} else {
					field := &mysql.Field{}
					r.Fields[j] = field
					r.FieldNames[string(r.Fields[j].Name)] = j
					field.Name = []byte(names[j])
					if err = formatField(field, value); err != nil {
						return nil, err
					}
				}

			}
			b, err = formatValue(value)
			if err != nil {
				return nil, err
			}

			row = append(row, mysql.PutLengthEncodedString(b)...)
		}

		r.RowDatas = append(r.RowDatas, row)
	}
	//assign the values to the result
	r.Values = values

	return r, nil
}

func (c *ClientConn) writeData(columns []string, data [][]interface{}) error {
	rs, err := c.buildResultset(nil, columns, data)
	if err != nil {
		return c.writeError(err)
	}
	return c.writeResultset(c.status, rs)
}

func (c *ClientConn) writeResultset(status uint16, r *mysql.Resultset) error {
	total := make([]byte, 0, 4096)
	data := make([]byte, 4, 512)
	var err error

	columnLen := mysql.PutLengthEncodedInt(uint64(len(r.Fields)))

	data = append(data, columnLen...)
	total, err = c.writePacketBatch(total, data, false)
	if err != nil {
		return err
	}

	for _, v := range r.Fields {
		data = data[0:4]
		data = append(data, v.Dump()...)
		total, err = c.writePacketBatch(total, data, false)
		if err != nil {
			return err
		}
	}

	total, err = c.writeEOFBatch(total, status, false)
	if err != nil {
		return err
	}

	for _, v := range r.RowDatas {
		data = data[0:4]
		data = append(data, v...)
		total, err = c.writePacketBatch(total, data, false)
		if err != nil {
			return err
		}
	}

	total, err = c.writeEOFBatch(total, status, true)
	total = nil
	if err != nil {
		return err
	}

	return nil
}

func (c *ClientConn) writeOK(r *mysql.Result) error {
	if r == nil {
		r = &mysql.Result{Status: c.status}
	}
	data := make([]byte, 4, 32)

	data = append(data, mysql.OK_HEADER)

	data = append(data, mysql.PutLengthEncodedInt(r.AffectedRows)...)
	data = append(data, mysql.PutLengthEncodedInt(r.InsertId)...)

	if c.capability&mysql.CLIENT_PROTOCOL_41 > 0 {
		data = append(data, byte(r.Status), byte(r.Status>>8))
		data = append(data, 0, 0)
	}

	return c.writePacket(data)
}

func (c *ClientConn) writeError(e error) error {
	if e == nil {
		return c.writeOK(nil)
	}
	var m *mysql.SqlError
	var ok bool
	if m, ok = e.(*mysql.SqlError); !ok {
		m = mysql.NewError(mysql.ER_UNKNOWN_ERROR, e.Error())
	}

	data := make([]byte, 4, 16+len(m.Message))

	data = append(data, mysql.ERR_HEADER)
	data = append(data, byte(m.Code), byte(m.Code>>8))

	if c.capability&mysql.CLIENT_PROTOCOL_41 > 0 {
		data = append(data, '#')
		data = append(data, m.State...)
	}

	data = append(data, m.Message...)

	return c.writePacket(data)
}

func (c *ClientConn) writeEOF(status uint16) error {
	data := make([]byte, 4, 9)

	data = append(data, mysql.EOF_HEADER)
	if c.capability&mysql.CLIENT_PROTOCOL_41 > 0 {
		data = append(data, 0, 0)
		data = append(data, byte(status), byte(status>>8))
	}

	return c.writePacket(data)
}

func (c *ClientConn) writeEOFBatch(total []byte, status uint16, direct bool) ([]byte, error) {
	data := make([]byte, 4, 9)

	data = append(data, mysql.EOF_HEADER)
	if c.capability&mysql.CLIENT_PROTOCOL_41 > 0 {
		data = append(data, 0, 0)
		data = append(data, byte(status), byte(status>>8))
	}

	return c.writePacketBatch(total, data, direct)
}
func formatValue(value interface{}) ([]byte, error) {
	if value == nil {
		return []byte("NULL"), nil
	}
	switch v := value.(type) {
	case int8:
		return strconv.AppendInt(nil, int64(v), 10), nil
	case int16:
		return strconv.AppendInt(nil, int64(v), 10), nil
	case int32:
		return strconv.AppendInt(nil, int64(v), 10), nil
	case int64:
		return strconv.AppendInt(nil, int64(v), 10), nil
	case int:
		return strconv.AppendInt(nil, int64(v), 10), nil
	case uint8:
		return strconv.AppendUint(nil, uint64(v), 10), nil
	case uint16:
		return strconv.AppendUint(nil, uint64(v), 10), nil
	case uint32:
		return strconv.AppendUint(nil, uint64(v), 10), nil
	case uint64:
		return strconv.AppendUint(nil, uint64(v), 10), nil
	case uint:
		return strconv.AppendUint(nil, uint64(v), 10), nil
	case float32:
		return strconv.AppendFloat(nil, float64(v), 'f', -1, 64), nil
	case float64:
		return strconv.AppendFloat(nil, float64(v), 'f', -1, 64), nil
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		return nil, fmt.Errorf("invalid type %T", value)
	}
}

func formatField(field *mysql.Field, value interface{}) error {
	switch value.(type) {
	case int8, int16, int32, int64, int:
		field.Charset = 63
		field.Type = mysql.MYSQL_TYPE_LONGLONG
		field.Flag = mysql.BINARY_FLAG | mysql.NOT_NULL_FLAG
	case uint8, uint16, uint32, uint64, uint:
		field.Charset = 63
		field.Type = mysql.MYSQL_TYPE_LONGLONG
		field.Flag = mysql.BINARY_FLAG | mysql.NOT_NULL_FLAG | mysql.UNSIGNED_FLAG
	case float32, float64:
		field.Charset = 63
		field.Type = mysql.MYSQL_TYPE_DOUBLE
		field.Flag = mysql.BINARY_FLAG | mysql.NOT_NULL_FLAG
	case string, []byte:
		field.Charset = 33
		field.Type = mysql.MYSQL_TYPE_VAR_STRING
	default:
		return fmt.Errorf("unsupport type %T for resultset", value)
	}
	return nil
}
