// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

//go:build ignore
// +build ignore

// This file is compiled by docker_test.go to build a test client.
// It's not part of the pam module itself.

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/msteinert/pam"
	"github.com/sirupsen/logrus"
)

func main() {
	if len(os.Args) != 4 || os.Args[1] != "try" {
		logrus.Print("usage: testclient try 'username' 'password'")
		os.Exit(1)
	}
	username := os.Args[2]
	password := os.Args[3]

	// Configure PAM to use arvados token auth by default.
	cmd := exec.Command("pam-auth-update", "--force", "arvados", "--remove", "unix")
	cmd.Env = append([]string{"DEBIAN_FRONTEND=noninteractive"}, os.Environ()...)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logrus.WithError(err).Error("pam-auth-update failed")
		os.Exit(1)
	}

	// Check that pam-auth-update actually added arvados config.
	cmd = exec.Command("grep", "-Hn", "arvados", "/etc/pam.d/common-auth")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		panic(err)
	}

	logrus.Debugf("starting pam: username=%q password=%q", username, password)

	sentPassword := false
	errorMessage := ""
	tx, err := pam.StartFunc("default", username, func(style pam.Style, message string) (string, error) {
		logrus.Debugf("pam conversation: style=%v message=%q", style, message)
		switch style {
		case pam.ErrorMsg:
			logrus.WithField("Message", message).Info("pam.ErrorMsg")
			errorMessage = message
			return "", nil
		case pam.TextInfo:
			logrus.WithField("Message", message).Info("pam.TextInfo")
			errorMessage = message
			return "", nil
		case pam.PromptEchoOn, pam.PromptEchoOff:
			sentPassword = true
			return password, nil
		default:
			return "", fmt.Errorf("unrecognized message style %d", style)
		}
	})
	if err != nil {
		logrus.WithError(err).Print("StartFunc failed")
		os.Exit(1)
	}
	err = tx.Authenticate(pam.DisallowNullAuthtok)
	if err != nil {
		err = fmt.Errorf("PAM: %s (message = %q, sentPassword = %v)", err, errorMessage, sentPassword)
		logrus.WithError(err).Print("authentication failed")
		os.Exit(1)
	}
	logrus.Print("authentication succeeded")
}
