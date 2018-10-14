package main

import (
	"github.com/labstack/gommon/log"
	"net"
	"time"
)

func main() {

	listen, err := net.Listen("tcp4", ":65001")

	if err != nil {
		log.Fatal(err)
	}

	defer listen.Close()

	for {

		conn, err := listen.Accept()

		if err != nil {
			log.Printf("accept error: %v", err)
			break
		}

		// log.Print("Client Connected!")

		go HandleConn(conn)

		time.Sleep(10000)
	}

}
