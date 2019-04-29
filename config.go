package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var (
	hostname string
)

func init() {
	hostname, _ = os.Hostname()
}

type Config struct {
	Dev      bool           `yaml:"dev"`
	Hostname string         `yaml:"hostname"`
	Bind     string         `yaml:"bind"`
	HostKeys ConfigHostKeys `yaml:"host_keys"`
	Endpoint ConfigEndpoint `yaml:"endpoint"`
}

type ConfigEndpoint struct {
	URL  string `yaml:"url"`
	CA   string `yaml:"ca"`
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

type ConfigHostKeys struct {
	RSA     string `yaml:"rsa"`
	ECDSA   string `yaml:"ecdsa"`
	ED25519 string `yaml:"ed25519"`
}

func LoadConfigFile(file string) (o Config, err error) {
	var buf []byte
	if buf, err = ioutil.ReadFile(file); err != nil {
		return
	}
	if err = yaml.Unmarshal(buf, &o); err != nil {
		return
	}
	defaultStr(&o.Hostname, hostname)
	if len(o.Hostname) == 0 {
		err = errors.New("failed to get hostname")
		return
	}
	defaultStr(&o.Bind, "0.0.0.0:2222")
	defaultStr(&o.HostKeys.RSA, "host_rsa")
	resolveRelative(&o.HostKeys.RSA, file)
	defaultStr(&o.HostKeys.ECDSA, "host_ecdsa")
	resolveRelative(&o.HostKeys.ECDSA, file)
	defaultStr(&o.HostKeys.ED25519, "host_ed25519")
	resolveRelative(&o.HostKeys.ED25519, file)
	defaultStr(&o.Endpoint.URL, "http://127.0.0.1:2223")
	resolveRelative(&o.Endpoint.CA, file)
	resolveRelative(&o.Endpoint.Cert, file)
	resolveRelative(&o.Endpoint.Key, file)
	return
}

func (c Config) CreateEndpoint() (*Endpoint, error) {
	return NewEndpoint(EndpointOptions{
		Hostname:       c.Hostname,
		URL:            c.Endpoint.URL,
		ExtraCAFile:    c.Endpoint.CA,
		ClientCertFile: c.Endpoint.Cert,
		ClientKeyFile:  c.Endpoint.Key,
	})
}

func (c Config) CreateSigners() (ss []ssh.Signer, err error) {
	ss = make([]ssh.Signer, 0, 3)
	var s ssh.Signer
	var g bool
	if s, g, err = LoadOrGenerateRSASigner(c.HostKeys.RSA); err != nil {
		log.Error().Err(err).Str("alg", "rsa").Str("file", c.HostKeys.RSA).Bool("generated", g).Msg("host key failed to load")
		return
	}
	log.Info().Str("alg", "rsa").Str("file", c.HostKeys.RSA).Bool("generated", g).Msg("host key loaded")
	ss = append(ss, s)
	if s, g, err = LoadOrGenerateECDSASigner(c.HostKeys.ECDSA); err != nil {
		log.Error().Err(err).Str("alg", "ecdsa").Str("file", c.HostKeys.ECDSA).Bool("generated", g).Msg("host key failed to load")
		return
	}
	log.Info().Str("alg", "ecdsa").Str("file", c.HostKeys.ECDSA).Bool("generated", g).Msg("host key loaded")
	ss = append(ss, s)
	if s, g, err = LoadOrGenerateEd25519Signer(c.HostKeys.ED25519); err != nil {
		log.Error().Err(err).Str("alg", "ed25519").Str("file", c.HostKeys.ED25519).Bool("generated", g).Msg("host key failed to load")
		return
	}
	log.Info().Str("alg", "ed25519").Str("file", c.HostKeys.ED25519).Bool("generated", g).Msg("host key loaded")
	ss = append(ss, s)
	return
}

// load or generate a pkcs8 compatible RSA private key file
func LoadOrGenerateRSASigner(f string) (s ssh.Signer, g bool, err error) {
	var buf []byte
	if buf, err = ioutil.ReadFile(f); err != nil {
		// if error occurred
		if !os.IsNotExist(err) {
			return
		}
		err = nil

		g = true

		// generate rsa key
		var privateKey *rsa.PrivateKey
		if privateKey, err = rsa.GenerateKey(rand.Reader, 4096); err != nil {
			return
		}

		// encode private key with pem
		var pkcs []byte
		if pkcs, err = x509.MarshalPKCS8PrivateKey(privateKey); err != nil {
			return
		}
		buf = pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: pkcs,
		})

		// write file
		if err = ioutil.WriteFile(f, buf, 0600); err != nil {
			return
		}
	}

	// load pem encoded private key
	if s, err = ssh.ParsePrivateKey(buf); err != nil {
		return
	}
	return
}

// load or generate a pkcs8 compatible RSA private key file
func LoadOrGenerateECDSASigner(f string) (s ssh.Signer, g bool, err error) {
	var buf []byte
	if buf, err = ioutil.ReadFile(f); err != nil {
		// if error occurred
		if !os.IsNotExist(err) {
			return
		}
		err = nil

		g = true

		// generate ecdsa key
		var privateKey *ecdsa.PrivateKey
		if privateKey, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader); err != nil {
			return
		}

		// encode private key with pkcs8
		var pkcs []byte
		if pkcs, err = x509.MarshalPKCS8PrivateKey(privateKey); err != nil {
			return
		}
		buf = pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: pkcs,
		})

		// write file
		if err = ioutil.WriteFile(f, buf, 0600); err != nil {
			return
		}
	}

	// load pem encoded private key
	if s, err = ssh.ParsePrivateKey(buf); err != nil {
		return
	}
	return
}

// load or generate a custom formatted Ed25519 private key file
func LoadOrGenerateEd25519Signer(f string) (s ssh.Signer, g bool, err error) {
	var buf []byte
	if buf, err = ioutil.ReadFile(f); err != nil {
		// if error occurred
		if !os.IsNotExist(err) {
			return
		}
		err = nil

		g = true

		// generate ecdsa key
		var privateKey ed25519.PrivateKey
		if _, privateKey, err = ed25519.GenerateKey(rand.Reader); err != nil {
			return
		}

		buf = pem.EncodeToMemory(&pem.Block{
			Type:  "ED25519 PRIVATE KEY",
			Bytes: privateKey,
		})

		// write file
		if err = ioutil.WriteFile(f, buf, 0600); err != nil {
			return
		}
	}

	var b *pem.Block
	if b, _ = pem.Decode(buf); len(b.Bytes) == 0 {
		err = errors.New("failed to load ed25519 key")
		return
	}
	if b.Type != "ED25519 PRIVATE KEY" {
		err = errors.New("not a ed25519 private key")
		return
	}

	// load pem encoded private key
	if s, err = ssh.NewSignerFromKey(ed25519.PrivateKey(b.Bytes)); err != nil {
		return
	}
	return
}

func defaultStr(v *string, defaultValue string) {
	*v = strings.TrimSpace(*v)
	if len(*v) == 0 {
		*v = defaultValue
	}
}

func resolveRelative(v *string, base string) {
	if len(*v) != 0 {
		*v = filepath.Join(base, "../", *v)
	}
}
