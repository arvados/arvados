// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

func main() {
	err := generate()
	if err != nil {
		log.Fatal(err)
	}
}

func generate() error {
	outfn := "generated_config.go"
	tmpfile, err := ioutil.TempFile(".", "."+outfn+".")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())

	gofmt := exec.Command("gofmt", "-s")
	gofmt.Stdout = tmpfile
	gofmt.Stderr = os.Stderr
	w, err := gofmt.StdinPipe()
	if err != nil {
		return err
	}
	gofmt.Start()

	// copyright header: same as this file
	cmd := exec.Command("head", "-n", "4", "generate.go")
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile("config.default.yml")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "package config\nvar DefaultYAML = []byte(`%s`)", bytes.Replace(data, []byte{'`'}, []byte("`+\"`\"+`"), -1))
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	err = gofmt.Wait()
	if err != nil {
		return err
	}
	err = tmpfile.Close()
	if err != nil {
		return err
	}
	return os.Rename(tmpfile.Name(), outfn)
}
