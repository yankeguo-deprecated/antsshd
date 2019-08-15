package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/antssh/types"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	Addr   string `yaml:"addr"`
	Secure bool   `yaml:"secure"`
	CA     string `yaml:"ca"`
	Cert   string `yaml:"cert"`
	Key    string `yaml:"key"`
}

type ConfigHostKeys struct {
	RSA     string `yaml:"rsa"`
	ECDSA   string `yaml:"ecdsa"`
	ED25519 string `yaml:"ed25519"`
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

func createSigner(
	f string,
	allowG bool,
	funcGen func() (buf []byte, err error),
	funcLoad func(buf []byte) (s ssh.Signer, err error),
) (s ssh.Signer, g bool, err error) {
	var buf []byte
	if buf, err = ioutil.ReadFile(f); err != nil {
		if !allowG || !os.IsNotExist(err) {
			return
		}
		g, err = true, nil
		if buf, err = funcGen(); err != nil {
			return
		}
		if err = ioutil.WriteFile(f, buf, 0600); err != nil {
			return
		}
	}
	s, err = funcLoad(buf)
	return
}

func createRSASigner(f string, allowG bool) (ssh.Signer, bool, error) {
	return createSigner(
		f,
		allowG,
		func() (buf []byte, err error) {
			var privateKey *rsa.PrivateKey
			if privateKey, err = rsa.GenerateKey(rand.Reader, 4096); err != nil {
				return
			}
			var pkcs []byte
			if pkcs, err = x509.MarshalPKCS8PrivateKey(privateKey); err != nil {
				return
			}
			buf = pem.EncodeToMemory(&pem.Block{
				Type:  "PRIVATE KEY",
				Bytes: pkcs,
			})
			return
		},
		func(buf []byte) (s ssh.Signer, err error) {
			s, err = ssh.ParsePrivateKey(buf)
			return
		},
	)
}

func createECDSASigner(f string, allowG bool) (ssh.Signer, bool, error) {
	return createSigner(
		f,
		allowG,
		func() (buf []byte, err error) {
			var privateKey *ecdsa.PrivateKey
			if privateKey, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader); err != nil {
				return
			}
			var pkcs []byte
			if pkcs, err = x509.MarshalPKCS8PrivateKey(privateKey); err != nil {
				return
			}
			buf = pem.EncodeToMemory(&pem.Block{
				Type:  "PRIVATE KEY",
				Bytes: pkcs,
			})
			return
		},
		func(buf []byte) (s ssh.Signer, err error) {
			s, err = ssh.ParsePrivateKey(buf)
			return
		},
	)
}

func createEd25519Signer(f string, allowG bool) (ssh.Signer, bool, error) {
	return createSigner(
		f,
		allowG,
		func() (buf []byte, err error) {
			var privateKey ed25519.PrivateKey
			if _, privateKey, err = ed25519.GenerateKey(rand.Reader); err != nil {
				return
			}
			buf = pem.EncodeToMemory(&pem.Block{
				Type:  "ED25519 PRIVATE KEY",
				Bytes: privateKey,
			})
			return
		},
		func(buf []byte) (s ssh.Signer, err error) {
			var b *pem.Block
			if b, _ = pem.Decode(buf); len(b.Bytes) == 0 {
				err = errors.New("failed to load ed25519 key")
				return
			}
			if b.Type != "ED25519 PRIVATE KEY" {
				err = errors.New("not a ed25519 private key")
				return
			}
			if s, err = ssh.NewSignerFromKey(ed25519.PrivateKey(b.Bytes)); err != nil {
				return
			}
			return
		},
	)
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
	defaultStr(&o.Endpoint.Addr, "127.0.0.1:2223")
	resolveRelative(&o.Endpoint.CA, file)
	resolveRelative(&o.Endpoint.Cert, file)
	resolveRelative(&o.Endpoint.Key, file)
	return
}

func (c Config) createTLSConfig() (tlsCfg *tls.Config, err error) {
	tlsCfg = &tls.Config{}
	// load system ca
	if tlsCfg.RootCAs, err = x509.SystemCertPool(); err != nil {
		return
	}
	// append custom ca
	if len(c.Endpoint.CA) > 0 {
		log.Debug().Str("file", c.Endpoint.CA).Msg("custom endpoint ca found, loading...")
		var buf []byte
		if buf, err = ioutil.ReadFile(c.Endpoint.CA); err != nil {
			return
		}
		if !tlsCfg.RootCAs.AppendCertsFromPEM(buf) {
			err = errors.New("failed to load custom ca")
			return
		}
	}
	// load client cert
	if len(c.Endpoint.Cert) > 0 && len(c.Endpoint.Key) > 0 {
		log.Debug().Str("cert-file", c.Endpoint.Cert).Str("cert-key", c.Endpoint.Key).Msg("client certificate found, loading...")
		var cert tls.Certificate
		if cert, err = tls.LoadX509KeyPair(c.Endpoint.Cert, c.Endpoint.Key); err != nil {
			return
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	return
}

func (c Config) CreateAgentClient() (client types.AgentControllerClient, err error) {
	var opts []grpc.DialOption
	if c.Endpoint.Secure {
		var tlsCfg *tls.Config
		if tlsCfg, err = c.createTLSConfig(); err != nil {
			return
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	var conn *grpc.ClientConn
	if conn, err = grpc.Dial(c.Endpoint.Addr, opts...); err != nil {
		return
	}
	client = types.NewAgentControllerClient(conn)
	return
}

func (c Config) CreateSigners(allowG bool) (ss []ssh.Signer, err error) {
	ss = make([]ssh.Signer, 0, 3)
	var s ssh.Signer
	var g bool
	if s, g, err = createRSASigner(c.HostKeys.RSA, allowG); err != nil {
		log.Error().Err(err).Str("alg", "rsa").Str("file", c.HostKeys.RSA).Bool("generated", g).Msg("host key failed to load")
		return
	}
	log.Info().Str("alg", "rsa").Str("file", c.HostKeys.RSA).Bool("generated", g).Msg("host key loaded")
	ss = append(ss, s)
	if s, g, err = createECDSASigner(c.HostKeys.ECDSA, allowG); err != nil {
		log.Error().Err(err).Str("alg", "ecdsa").Str("file", c.HostKeys.ECDSA).Bool("generated", g).Msg("host key failed to load")
		return
	}
	log.Info().Str("alg", "ecdsa").Str("file", c.HostKeys.ECDSA).Bool("generated", g).Msg("host key loaded")
	ss = append(ss, s)
	if s, g, err = createEd25519Signer(c.HostKeys.ED25519, allowG); err != nil {
		log.Error().Err(err).Str("alg", "ed25519").Str("file", c.HostKeys.ED25519).Bool("generated", g).Msg("host key failed to load")
		return
	}
	log.Info().Str("alg", "ed25519").Str("file", c.HostKeys.ED25519).Bool("generated", g).Msg("host key loaded")
	ss = append(ss, s)
	return
}
