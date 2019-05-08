package server

import (
	"github.com/flike/kingshard/mysql"
	"github.com/syndtr/goleveldb/leveldb"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync/atomic"
)


type Server struct {
	Addr     string
	listener net.Listener
	running  bool
	datadir  string
	schemas  map[string]*Schema
	schema   *Schema
}

func (s *Server) _createDatabase(schema string) error {
	err := os.Mkdir("./data/"+schema, 0755)
	if err != nil {
		return err
	}
	db, err := leveldb.OpenFile(s.datadir+"/"+schema, nil)
	if err != nil {
		return err
	}
	s.schemas[schema] = &Schema{schema, db}
	return nil
}

func (s *Server) Run() error {
	s.schemas = map[string]*Schema{}
	var err error
	s.datadir = "./data"
	if _, err := os.Stat(s.datadir); os.IsNotExist(err) {
		os.Mkdir("./data", 0755)
	}
	dirs, _ := ioutil.ReadDir(s.datadir)
	for _, info := range dirs {
		if info.IsDir() {
			db, err := leveldb.OpenFile(s.datadir+"/"+info.Name(), nil)
			if err != nil {
				return err
			}
			s.schemas[info.Name()] = &Schema{info.Name(), db}
		}
	}

	s.running = true

	s.listener, err = net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	for s.running {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go s.onConn(conn)
	}

	return nil
}

func (s *Server) onConn(c net.Conn) {
	conn := s.newClientConn(c) //新建一个conn

	if err := conn.Handshake(); err != nil {
		conn.writeError(err)
		conn.Close()
		return
	}

	conn.Run()
}

func (s *Server) newClientConn(co net.Conn) *ClientConn {
	c := new(ClientConn)
	tcpConn := co.(*net.TCPConn)

	tcpConn.SetNoDelay(false)
	c.c = tcpConn

	c.pkg = mysql.NewPacketIO(tcpConn)
	c.s = s

	c.pkg.Sequence = 0

	c.connectionId = atomic.AddUint32(&baseConnId, 1)

	c.status = mysql.SERVER_STATUS_AUTOCOMMIT

	c.salt, _ = mysql.RandomBuf(20)

	c.closed = false

	c.charset = mysql.DEFAULT_CHARSET
	c.collation = mysql.DEFAULT_COLLATION_ID

	return c
}
