package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type Endpoint struct {
	hostname string
	url      string
	client   *http.Client
}

type EndpointOptions struct {
	Hostname       string
	URL            string
	ExtraCAFile    string
	ClientCertFile string
	ClientKeyFile  string
}

func NewEndpoint(opts EndpointOptions) (e *Endpoint, err error) {
	// parse url
	var u *url.URL
	if u, err = url.Parse(opts.URL); err != nil {
		return
	}

	// endpoint
	e = &Endpoint{
		url:      opts.URL,
		hostname: opts.Hostname,
	}

	// tls setup
	if strings.ToLower(u.Scheme) == "https" {
		tlsCfg := &tls.Config{}

		// load system ca
		if tlsCfg.RootCAs, err = x509.SystemCertPool(); err != nil {
			return
		}

		// append custom ca
		if len(opts.ExtraCAFile) > 0 {
			log.Debug().Str("file", opts.ExtraCAFile).Msg("custom endpoint ca found, loading...")
			var buf []byte
			if buf, err = ioutil.ReadFile(opts.ExtraCAFile); err != nil {
				return
			}
			if !tlsCfg.RootCAs.AppendCertsFromPEM(buf) {
				err = errors.New("failed to load custom ca")
				return
			}
		}

		// load client cert
		if len(opts.ClientCertFile) > 0 && len(opts.ClientKeyFile) > 0 {
			log.Debug().Str("cert-file", opts.ClientCertFile).Str("cert-key", opts.ClientKeyFile).Msg("client certificate found, loading...")
			var cert tls.Certificate
			if cert, err = tls.LoadX509KeyPair(opts.ClientCertFile, opts.ClientKeyFile); err != nil {
				return
			}
			tlsCfg.Certificates = []tls.Certificate{cert}
		}

		// create the client
		e.client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsCfg,
			},
		}
	} else {
		e.client = http.DefaultClient
	}
	return
}

func (e *Endpoint) CanConnect(user, pk string) error {
	return e.Can(EndpointRequest{
		Hostname:  e.hostname,
		User:      user,
		PublicKey: pk,
		Type:      "connect",
	})
}

func (e *Endpoint) CanExecute(user, pk string) error {
	return e.Can(EndpointRequest{
		Hostname:  e.hostname,
		User:      user,
		PublicKey: pk,
		Type:      "execute",
	})
}

func (e *Endpoint) CanProxy(user, pk, host string, port int) error {
	return e.Can(EndpointRequest{
		Hostname:  e.hostname,
		User:      user,
		PublicKey: pk,
		Type:      "proxy",
		Proxy: EndpointRequestProxy{
			Host: host,
			Port: port,
		},
	})
}

func (e *Endpoint) CanForward(user, pk, host string, port int) error {
	return e.Can(EndpointRequest{
		Hostname:  e.hostname,
		User:      user,
		PublicKey: pk,
		Type:      "forward",
		Forward: EndpointRequestForward{
			Host: host,
			Port: port,
		},
	})
}

func (e *Endpoint) Can(req EndpointRequest) (err error) {
	var buf []byte
	if buf, err = json.Marshal(req); err != nil {
		return
	}
	var res *http.Response
	if res, err = e.client.Post(e.url, "application/json", bytes.NewReader(buf)); err != nil {
		return
	}
	defer res.Body.Close()
	var body []byte
	if body, err = ioutil.ReadAll(res.Body); err != nil {
		return
	}
	if res.StatusCode != http.StatusOK {
		err = errors.New(fmt.Sprintf("status %d, %s", res.StatusCode, body))
		return
	}
	return
}
