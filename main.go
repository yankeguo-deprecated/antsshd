package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var (
	err error

	exe string
)

func exit() {
	if err != nil {
		log.Println("exited with error:", err)
		os.Exit(1)
	} else {
		log.Println("exited")
	}
}

func main() {
	defer exit()

	var isWorker bool

	flag.BoolVar(&isWorker, "as-worker", false, "start as a worker process (internal only)")
	flag.Parse()

	if isWorker {
		err = workerMain()
	} else {
		err = masterMain()
	}
}

func masterMain() (err error) {
	// determine executable
	if exe, err = os.Executable(); err != nil {
		log.Println("failed to determine executable:", err)
		return
	}
	// TODO: implements options
	// start listener
	var l net.Listener
	if l, err = net.Listen("tcp", "127.0.0.1:8800"); err != nil {
		return
	}
	defer l.Close()
	// close chan
	cc := make(chan error, 1)
	// signal chan
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	// start loop
	go masterServ(l, cc)
	// wait for close or signal
	select {
	case s := <-sc:
		log.Println("signal caught:", s)
	case err = <-cc:
		log.Println("listener close unexpected:", err)
	}
	return
}

func masterServ(l net.Listener, sc chan error) {
	var err error
	for {
		// accept connection
		var c net.Conn
		if c, err = l.Accept(); err != nil {
			break
		}
		// handle connection
		if err := masterExec(c); err != nil {
			log.Println("failed to handle connection:", err)
		}
	}
	sc <- err
}

func masterExec(c net.Conn) (err error) {
	defer c.Close()
	// obtain fd
	var f *os.File
	if f, err = c.(*net.TCPConn).File(); err != nil {
		log.Println("failed to retrieve connection fd:", err)
		return
	}
	defer f.Close()
	// spawn worker
	cmd := exec.Command(exe, "-as-worker")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{f}
	if err = cmd.Start(); err != nil {
		return
	}
	return
}

func workerMain() (err error) {
	// retrieve connection
	var c net.Conn
	if c, err = net.FileConn(os.NewFile(3, "connection")); err != nil {
		log.Println("failed to retrieve connection:", err)
		return
	}
	defer c.Close()

	// ignore SIGINT/SIGTERM
	discardSignals()

	// TODO: implements worker
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)
		if _, err = c.Write([]byte(strconv.Itoa(i) + "\r\n")); err != nil {
			log.Println("failed to write connection:", err)
			return
		}
	}
	return
}

func discardSignals() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			<-sc
		}
	}()
}
