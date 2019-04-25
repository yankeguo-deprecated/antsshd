package main

import (
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

var (
	err error

	exe string
	cfg Config
)

func exit() {
	if err != nil {
		log.Error().Err(err).Msg("exited")
		os.Exit(1)
	} else {
		log.Info().Msg("exited")
	}
}

func setupZerolog(dev bool) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, NoColor: !dev, TimeFormat: time.RFC3339})
	if dev {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

func main() {
	defer exit()

	// pre-setup zerolog
	setupZerolog(false)

	// determine executable
	if exe, err = os.Executable(); err != nil {
		log.Error().Err(err).Msg("failed to determine executable")
		return
	}

	// flags
	var dev bool
	var dir string
	var worker bool

	flag.BoolVar(&dev, "dev", false, "enable dev mode")
	flag.StringVar(&dir, "dir", "/etc/antsshd", "config base directory, a 'config.yml' file is required")
	flag.BoolVar(&worker, "worker", false, "start as a worker process (internal only)")
	flag.Parse()

	// load options
	if cfg, err = LoadConfigFile(filepath.Join(dir, "config.yml")); err != nil {
		return
	}

	// apply dev from cli
	if dev {
		cfg.Dev = true
		setupZerolog(true)
	}

	// start master / worker
	if worker {
		err = workerMain()
	} else {
		err = masterMain()
	}
}

func masterMain() (err error) {
	log.Info().Msg("master started")
	// start listener
	var l net.Listener
	if l, err = net.Listen("tcp", cfg.Bind); err != nil {
		return
	}
	defer l.Close()
	// chan-close
	cc := make(chan error, 1)
	// chan-signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	// start loop
	go masterServ(l, cc)
	// wait for close or signal
	select {
	case s := <-sc:
		log.Info().Str("signal", s.String()).Msg("signal caught")
	case err = <-cc:
		log.Error().Err(err).Msg("listener closed unexpected")
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
		if err := masterFork(c); err != nil {
			log.Error().Err(err).Msg("failed to handle connection")
		}
	}
	sc <- err
}

func masterFork(c net.Conn) (err error) {
	defer c.Close()
	// obtain fd
	var f *os.File
	if f, err = c.(*net.TCPConn).File(); err != nil {
		log.Error().Err(err).Msg("failed to obtain connection fd")
		return
	}
	defer f.Close()
	// spawn worker
	cmd := exec.Command(exe, append([]string{"-worker"}, os.Args[1:]...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{f}
	if err = cmd.Start(); err != nil {
		return
	}
	return
}

func workerMain() (err error) {
	log.Info().Msg("worker started")
	// obtain connection
	var c net.Conn
	if c, err = net.FileConn(os.NewFile(3, "connection")); err != nil {
		log.Error().Err(err).Msg("failed to obtain connection")
		return
	}
	defer c.Close()

	// ignore SIGINT/SIGTERM
	discardSignals()

	// TODO: implements worker
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)
		if _, err = c.Write([]byte(strconv.Itoa(i) + "\r\n")); err != nil {
			log.Error().Err(err).Msg("failed to write connection")
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
