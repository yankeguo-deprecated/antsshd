package main

import (
	"context"
	"errors"
	"flag"
	"github.com/antssh/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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
	var optDev bool
	var optDir string
	var optWorker bool

	flag.BoolVar(&optDev, "dev", false, "enable dev mode")
	flag.StringVar(&optDir, "dir", "/etc/antsshd", "config base directory, a 'config.yml' file is required")
	flag.BoolVar(&optWorker, "worker", false, "start as a worker process (internal only)")
	flag.Parse()

	// load options
	if cfg, err = LoadConfigFile(filepath.Join(optDir, "config.yml")); err != nil {
		return
	}

	// apply dev from cli
	if cfg.Dev = cfg.Dev || optDev; cfg.Dev {
		setupZerolog(true)
	}

	// start master / worker
	if optWorker {
		err = workerMain()
	} else {
		err = masterMain()
	}
}

func masterMain() (err error) {
	log.Info().Msg("master started")
	// preload host keys
	if _, err = cfg.CreateSigners(true); err != nil {
		return
	}
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
	cmd.Dir, _ = os.Getwd()
	cmd.Env = os.Environ()
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

	// ignore SIGINT/SIGTERM
	discardSignals()

	// create ssh config
	sshCfg := &ssh.ServerConfig{}

	// create signers
	var signers []ssh.Signer
	if signers, err = cfg.CreateSigners(false); err != nil {
		return
	}

	for _, s := range signers {
		sshCfg.AddHostKey(s)
	}

	// create endpoint
	var client types.AgentClient
	if client, err = cfg.CreateAgentClient(); err != nil {
		return
	}

	sshCfg.PublicKeyCallback = func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		fp := ssh.FingerprintSHA256(key)
		if resp, err := client.Auth(context.Background(), &types.AgentAuthReq{
			Fingerprint: fp,
			User:        conn.User(),
			Hostname:    cfg.Hostname,
			Action:      types.AgentAuthReq_CONNECT,
		}); err != nil {
			return nil, err
		} else if !resp.Success {
			return nil, errors.New("auth not success: " + resp.Message)
		} else {
			return &ssh.Permissions{
				Extensions: map[string]string{
					"Fingerprint": fp,
					"RecordID":    resp.RecordId,
				},
			}, nil
		}
	}

	sshCfg.BannerCallback = func(conn ssh.ConnMetadata) string {
		return "welcome to antsshd"
	}

	// obtain connection
	var c net.Conn
	if c, err = net.FileConn(os.NewFile(3, "connection")); err != nil {
		log.Error().Err(err).Msg("failed to obtain connection")
		return
	}
	defer c.Close()

	// upgrade connection
	var sc *ssh.ServerConn
	var nc <-chan ssh.NewChannel
	var gr <-chan *ssh.Request
	if sc, nc, gr, err = ssh.NewServerConn(c, sshCfg); err != nil {
		return
	}
	defer sc.Close()

	// TODO: handle nc, gr
	_ = nc
	_ = gr

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
