package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadOptionsFile(t *testing.T) {
	c, err := LoadConfigFile("testdata/config.yml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.CreateSigners(); err != nil {
		t.Fatal(err)
	}
	fmt.Println(c)
}

func TestLoadOrGenerateRSASigner(t *testing.T) {
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Unix()))
	_ = os.MkdirAll(dir, 755)
	f := filepath.Join(dir, "id_rsa")

	var s ssh.Signer
	var err error
	var g bool

	if s, g, err = LoadOrGenerateRSASigner(f); err != nil {
		t.Fatal(err)
	}
	if !g {
		t.Fatal("should g")
	}

	fp := ssh.FingerprintSHA256(s.PublicKey())
	t.Logf("fp: %s\n", fp)

	if s, g, err = LoadOrGenerateRSASigner(f); err != nil {
		t.Fatal(err)
	}
	if g {
		t.Fatal("should not g")
	}
	if fp != ssh.FingerprintSHA256(s.PublicKey()) {
		t.Fatal("not same")
	}
}

func TestLoadOrGenerateECDSASigner(t *testing.T) {
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Unix()))
	_ = os.MkdirAll(dir, 755)
	f := filepath.Join(dir, "id_ecdsa")

	var s ssh.Signer
	var err error
	var g bool

	if s, g, err = LoadOrGenerateECDSASigner(f); err != nil {
		t.Fatal(err)
	}
	if !g {
		t.Fatal("should g")
	}

	fp := ssh.FingerprintSHA256(s.PublicKey())
	t.Logf("fp: %s\n", fp)

	if s, g, err = LoadOrGenerateECDSASigner(f); err != nil {
		t.Fatal(err)
	}
	if g {
		t.Fatal("should not g")
	}
	if fp != ssh.FingerprintSHA256(s.PublicKey()) {
		t.Fatal("not same")
	}
}

func TestLoadOrGenerateEd25519Signer(t *testing.T) {
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().Unix()))
	_ = os.MkdirAll(dir, 755)
	f := filepath.Join(dir, "id_ed25519")

	var s ssh.Signer
	var err error
	var g bool

	if s, g, err = LoadOrGenerateEd25519Signer(f); err != nil {
		t.Fatal(err)
	}
	if !g {
		t.Fatal("should g")
	}

	fp := ssh.FingerprintSHA256(s.PublicKey())
	t.Logf("fp: %s\n", fp)

	if s, g, err = LoadOrGenerateEd25519Signer(f); err != nil {
		t.Fatal(err)
	}
	if g {
		t.Fatal("should not g")
	}
	if fp != ssh.FingerprintSHA256(s.PublicKey()) {
		t.Fatal("not same")
	}
}
