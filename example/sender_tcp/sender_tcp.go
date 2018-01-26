package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/ftrvxmtrx/fd"
)

const (
	port   = 1234
	socket = "/tmp/sendfd.sock"
)

var (
	bind     bool
	takeover bool
)

func init() {
	flag.BoolVar(&bind, "bind", false, "bind on :1234")
	flag.BoolVar(&takeover, "takeover", false, "take fd from "+socket)
}

func main() {
	if flag.Parse(); !flag.Parsed() {
		flag.Usage()
		os.Exit(1)
	}

	if bind == takeover {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "either bind or takeover is required and are mutually exclusive")
		os.Exit(1)
	}

	var (
		sockL  net.Listener
		tcpL   net.Listener
		wg     sync.WaitGroup
		closed bool
		err    error
	)

	if bind {
		sockL, err = net.Listen("unix", socket)
		if err != nil {
			log.Fatalf("unix listen failed: %s\n", err)
		}
		if tcpL, err = net.Listen("tcp", ":"+strconv.Itoa(port)); err != nil {
			log.Fatalf("tcp listen failed: %s\n", err)
		}
		defer tcpL.Close()
	} else {
		c, err := net.Dial("unix", socket)
		if err != nil {
			log.Fatal("net.Dial failed: %s\n", err)
		}
		defer c.Close()

		var fs []*os.File
		fs, err = fd.Get(c.(*net.UnixConn), 2, []string{"f1", "f2"})
		if err != nil {
			log.Fatalf("GET failed: %s\n", err)
		}
		log.Println("fds received")
		defer fs[0].Close()
		defer fs[1].Close()

		if tcpL, err = net.FileListener(fs[0]); err != nil {
			log.Fatalf("error converting tcpL: %s\n", err)
		}

		if sockL, err = net.FileListener(fs[1]); err != nil {
			log.Fatalf("error converting sockL: %s\n", err)
		}
	}
	go accepter(&wg, &closed, tcpL)

	var sockA net.Conn
	sockA, err = sockL.Accept()
	if err != nil {
		log.Fatalf("unix accept failed: %s\n", err)
	}
	defer sockA.Close()

	var tcpF *os.File // will be closed immediately after Put
	if tcpF, err = tcpL.(*net.TCPListener).File(); err != nil {
		log.Fatalf("tcp File() failed: %s\n", err)
	}

	var sockLF *os.File // will be closed immediately after Put
	if sockLF, err = sockL.(*net.UnixListener).File(); err != nil {
		log.Fatalf("unix socket File() failed: %s\n", err)
	}
	if err = fd.Put(sockA.(*net.UnixConn), tcpF, sockLF); err != nil {
		log.Fatalf("Put failed: %s\n", err)
	}
	sockLF.Close()
	tcpF.Close()

	log.Println("sockets transfered, draining")
	closed = true
	wg.Wait()
	log.Println("drain complete, quitting")
}

func accepter(wg *sync.WaitGroup, closed *bool, tcpL net.Listener) {
	for !*closed {
		var c net.Conn
		var err error
		log.Println("accepting...")
		c, err = tcpL.Accept()
		if *closed {
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
		if _, err := fmt.Fprintf(c, "hello from %d\n", os.Getpid()); err != nil {
			log.Printf("write failed: %s\n", err.Error())
			c.Close()
			wg.Done()
			return
		}
		time.Sleep(1 * time.Second)
	}
}
