package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
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
		log.Fatalf("tcp listen failed: %s", err)
	}
	defer tcpl.Close()
	var wg sync.WaitGroup
	open := true
	go accepter(&wg, &open, tcpl)

	var l net.Listener
	l, err = net.Listen("unix", socket)
	if err != nil {
		log.Fatal("unix listen failed: %s\n", err)
	}
	defer l.Close()

	var a net.Conn
	a, err = l.Accept()
	if err != nil {
		log.Fatalf("unix accept failed: %s\n", err)
	}
	defer a.Close()

	listenConn := a.(*net.UnixConn)

	var f *os.File
	f, err = tcpl.(*net.TCPListener).File()
	if err != nil {
		log.Fatal("tcp File() failed: %s\n", err)
	}
	if err = fd.Put(listenConn, f); err != nil {
		log.Fatal("Put failed: %s\n", err)
	}
	log.Println("socket transfered, closing tcpl and waiting for termination")
	open = false
	tcpl.Close()
	f.Close()
	wg.Wait()
	log.Println("drain complete, closing")
}

func accepter(wg *sync.WaitGroup, open *bool, tcpl net.Listener) {
	for *open {
		var c net.Conn
		var err error
		log.Println("accepting...")
		c, err = tcpl.Accept()
		if !*open {
			log.Println("stopping accept loop")
			return
		}
		if err != nil {
			log.Fatalf("TCP accept failed: %s\n", err)
		}
		log.Println("accepted connection")
		wg.Add(1)
		go echo(wg, c)
	}
}

func echo(wg *sync.WaitGroup, c net.Conn) {
	for {
		if _, err := io.WriteString(c, "hello from sender\n"); err != nil {
			log.Printf("write failed: %s\n", err.Error())
			c.Close()
			wg.Done()
			return
		}
		time.Sleep(1 * time.Second)
	}
}
