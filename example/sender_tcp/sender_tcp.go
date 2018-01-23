package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/ftrvxmtrx/fd"
)

var (
	port   int
	socket string
)

func init() {
	flag.IntVar(&port, "p", 1234, "listen port")
	flag.StringVar(&socket, "s", "/tmp/sendfd.sock", "socket")
}

func main() {
	flag.Parse()

	if !flag.Parsed() || socket == "" {
		flag.Usage()
		os.Exit(1)
	}

	tcpl, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Fatal(err)
	}
	defer tcpl.Close()
	go accepter(tcpl)

	var f *os.File
	f, err = tcpl.(*net.TCPListener).File()
	if err != nil {
		log.Fatal(err)
	}

	var l net.Listener
	l, err = net.Listen("unix", socket)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	var a net.Conn
	a, err = l.Accept()
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()

	listenConn := a.(*net.UnixConn)
	if err = fd.Put(listenConn, f); err != nil {
		log.Fatal(err)
	}
}

func echo(c net.Conn) {
	for {
		if _, err := fmt.Fprintf(c, "hello from sender\n"); err != nil {
			fmt.Printf("connection closed: %s\n", err.Error())
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func accepter(tcpl net.Listener) {
	for {
		var c net.Conn
		var err error
		c, err = tcpl.Accept()
		if err != nil {
			log.Fatal(err)
		}
		defer c.Close()
		go echo(c)
	}
}
