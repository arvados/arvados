// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
)

func main() {
	checkOnly := false
	if len(os.Args) == 2 && os.Args[1] == "-check" {
		checkOnly = true
	} else if len(os.Args) != 1 {
		panic("usage: go run generate.go [-check]")
	}

	in, err := os.Open("list.go")
	if err != nil {
		panic(err)
	}
	buf, err := ioutil.ReadAll(in)
	if err != nil {
		panic(err)
	}
	orig := regexp.MustCompile(`(?ms)\nfunc [^\n]*generated_CollectionList\(.*?\n}\n`).Find(buf)
	if len(orig) == 0 {
		panic("can't find CollectionList func")
	}

	outfile, err := os.OpenFile("generated.go~", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		panic(err)
	}

	gofmt := exec.Command("goimports")
	gofmt.Stdout = outfile
	gofmt.Stderr = os.Stderr
	out, err := gofmt.StdinPipe()
	if err != nil {
		panic(err)
	}
	go func() {
		defer out.Close()
		out.Write(regexp.MustCompile(`(?ms)^.*package .*?import.*?\n\)\n`).Find(buf))
		io.WriteString(out, "//\n// -- this file is auto-generated -- do not edit -- edit list.go and run \"go generate\" instead --\n//\n\n")
		for _, t := range []string{"Container", "ContainerRequest", "Group", "Specimen", "User", "Link"} {
			_, err := out.Write(bytes.ReplaceAll(orig, []byte("Collection"), []byte(t)))
			if err != nil {
				panic(err)
			}
		}
	}()
	err = gofmt.Run()
	if err != nil {
		panic(err)
	}
	err = outfile.Close()
	if err != nil {
		panic(err)
	}
	if checkOnly {
		diff := exec.Command("diff", "-u", "/dev/fd/3", "/dev/fd/4")
		for _, fnm := range []string{"generated.go", "generated.go~"} {
			f, err := os.Open(fnm)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			diff.ExtraFiles = append(diff.ExtraFiles, f)
		}
		diff.Stdout = os.Stdout
		diff.Stderr = os.Stderr
		err = diff.Run()
		if err != nil {
			os.Exit(1)
		}
	} else {
		err = os.Rename("generated.go~", "generated.go")
		if err != nil {
			panic(err)
		}
	}
}
