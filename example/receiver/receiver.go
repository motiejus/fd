package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/ftrvxmtrx/fd"
)

var (
	socket string
)

func init() {
	flag.StringVar(&socket, "s", "/tmp/sendfd.sock", "socket")
}

func main() {
	flag.Parse()

	if !flag.Parsed() || socket == "" {
		flag.Usage()
		os.Exit(1)
	}

	c, err := net.Dial("unix", socket)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	fdConn := c.(*net.UnixConn)

	log.Println("waiting for an fd...")
	var fs []*os.File
	fs, err = fd.Get(fdConn, 1, []string{"a file"})
	if err != nil {
		log.Fatal(err)
	}
	f := fs[0]
	log.Println("fd received")

	tcpl, err := net.FileListener(f)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
	accepter(tcpl)
}

func accepter(tcpl net.Listener) {
	for {
		var c net.Conn
		var err error
		c, err = tcpl.Accept()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("accepted connection")
		go echo(c)
	}
}

func echo(c net.Conn) {
	for {
		if _, err := fmt.Fprintf(c, "hello from receiver\n"); err != nil {
			log.Printf("write error: %s\n", err.Error())
			c.Close()
			return
		}
		time.Sleep(1 * time.Second)
	}
}
