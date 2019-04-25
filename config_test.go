package main

import (
	"fmt"
	"testing"
)

func TestLoadOptionsFile(t *testing.T) {
	c, err := LoadConfigFile("testdata/config.yml")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(c)
}
