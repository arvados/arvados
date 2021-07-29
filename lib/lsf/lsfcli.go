// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package lsf

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
)

type bjobsEntry struct {
	id   int
	name string
	stat string
}

type lsfcli struct {
	logger logrus.FieldLogger
	// (for testing) if non-nil, call stubCommand() instead of
	// exec.Command() when running lsf command line programs.
	stubCommand func(string, ...string) *exec.Cmd
}

func (cli lsfcli) command(prog string, args ...string) *exec.Cmd {
	if f := cli.stubCommand; f != nil {
		return f(prog, args...)
	} else {
		return exec.Command(prog, args...)
	}
}

func (cli lsfcli) Bsub(script []byte, args []string, arv *arvados.Client) error {
	cli.logger.Infof("bsub command %q script %q", args, script)
	cmd := cli.command(args[0], args[1:]...)
	cmd.Env = append([]string(nil), os.Environ()...)
	cmd.Env = append(cmd.Env, "ARVADOS_API_HOST="+arv.APIHost)
	cmd.Env = append(cmd.Env, "ARVADOS_API_TOKEN="+arv.AuthToken)
	if arv.Insecure {
		cmd.Env = append(cmd.Env, "ARVADOS_API_HOST_INSECURE=1")
	}
	cmd.Stdin = bytes.NewReader(script)
	out, err := cmd.Output()
	cli.logger.WithField("stdout", string(out)).Infof("bsub finished")
	return errWithStderr(err)
}

func (cli lsfcli) Bjobs() ([]bjobsEntry, error) {
	cli.logger.Debugf("Bjobs()")
	cmd := cli.command("bjobs", "-u", "all", "-noheader", "-o", "jobid stat job_name:30")
	buf, err := cmd.Output()
	if err != nil {
		return nil, errWithStderr(err)
	}
	var bjobs []bjobsEntry
	for _, line := range strings.Split(string(buf), "\n") {
		if line == "" {
			continue
		}
		var ent bjobsEntry
		if _, err := fmt.Sscan(line, &ent.id, &ent.stat, &ent.name); err != nil {
			cli.logger.Warnf("ignoring unparsed line in bjobs output: %q", line)
			continue
		}
		bjobs = append(bjobs, ent)
	}
	return bjobs, nil
}

func (cli lsfcli) Bkill(id int) error {
	cli.logger.Infof("Bkill(%d)", id)
	cmd := cli.command("bkill", fmt.Sprintf("%d", id))
	buf, err := cmd.CombinedOutput()
	if err == nil || strings.Index(string(buf), "already finished") >= 0 {
		return nil
	} else {
		return fmt.Errorf("%s (%q)", err, buf)
	}
}

func errWithStderr(err error) error {
	if err, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("%s (%q)", err, err.Stderr)
	}
	return err
}
