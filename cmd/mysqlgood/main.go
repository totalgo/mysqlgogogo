package main

import (
	"github.com/totalgo/mysqlgogogo/server"
	"log"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	s := server.Server{Addr: ":9998"}
	log.Println(s.Run())
}
