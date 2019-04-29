package main

import (
	"errors"
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
	Dev         bool           `yaml:"dev"`
	Hostname    string         `yaml:"hostname"`
	Bind        string         `yaml:"bind"`
	HostRsa     string         `yaml:"host_rsa"`
	HostEcdsa   string         `yaml:"host_ecdsa"`
	HostEd25519 string         `yaml:"host_ed25519"`
	Endpoint    EndpointConfig `yaml:"endpoint"`
}

type EndpointConfig struct {
	URL  string `yaml:"url"`
	CA   string `yaml:"ca"`
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
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
	defaultStr(&o.HostRsa, "host_rsa")
	resolveRelative(&o.HostRsa, file)
	defaultStr(&o.HostEcdsa, "host_ecdsa")
	resolveRelative(&o.HostEcdsa, file)
	defaultStr(&o.HostEd25519, "host_ed25519")
	resolveRelative(&o.HostEd25519, file)
	defaultStr(&o.Endpoint.URL, "http://127.0.0.1:2223")
	resolveRelative(&o.Endpoint.CA, file)
	resolveRelative(&o.Endpoint.Cert, file)
	resolveRelative(&o.Endpoint.Key, file)
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
