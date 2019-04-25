package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type Config struct {
	Dev         bool              `yaml:"dev"`
	Bind        string            `yaml:"bind"`
	HostRsa     string            `yaml:"host_rsa"`
	HostEcdsa   string            `yaml:"host_ecdsa"`
	HostEd25519 string            `yaml:"host_ed25519"`
	Endpoint    string            `yaml:"endpoint"`
	EndpointTLS EndpointTLSConfig `yaml:"endpoint_tls"`
}

type EndpointTLSConfig struct {
	ServerCACert string `yaml:"server_ca_cert"`
	ClientCert   string `yaml:"client_cert"`
	ClientKey    string `yaml:"client_key"`
}

func LoadConfigFile(file string) (o Config, err error) {
	var buf []byte
	if buf, err = ioutil.ReadFile(file); err != nil {
		return
	}
	if err = yaml.Unmarshal(buf, &o); err != nil {
		return
	}
	defaultStr(&o.Bind, "0.0.0.0:2222")
	defaultStr(&o.HostRsa, "host_rsa")
	resolveRelative(&o.HostRsa, file)
	defaultStr(&o.HostEcdsa, "host_ecdsa")
	resolveRelative(&o.HostEcdsa, file)
	defaultStr(&o.HostEd25519, "host_ed25519")
	resolveRelative(&o.HostEd25519, file)
	defaultStr(&o.Endpoint, "http://127.0.0.1:2223")
	defaultStr(&o.EndpointTLS.ServerCACert, "server_ca_cert")
	resolveRelative(&o.EndpointTLS.ServerCACert, file)
	defaultStr(&o.EndpointTLS.ClientCert, "client_cert")
	resolveRelative(&o.EndpointTLS.ClientCert, file)
	defaultStr(&o.EndpointTLS.ClientKey, "client_key")
	resolveRelative(&o.EndpointTLS.ClientKey, file)
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
