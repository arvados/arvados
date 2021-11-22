// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package cloudtest

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/dispatchcloud"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"golang.org/x/crypto/ssh"
)

var Command command

type command struct{}

func (command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var err error
	defer func() {
		if err != nil {
			fmt.Fprintf(stderr, "%s\n", err)
		}
	}()

	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configFile := flags.String("config", arvados.DefaultConfigFile, "Site configuration `file`")
	instanceSetID := flags.String("instance-set-id", "zzzzz-zzzzz-zzzzzzcloudtest", "InstanceSetID tag `value` to use on the test instance")
	imageID := flags.String("image-id", "", "Image ID to use when creating the test instance (if empty, use cluster config)")
	instanceType := flags.String("instance-type", "", "Instance type to create (if empty, use cheapest type in config)")
	destroyExisting := flags.Bool("destroy-existing", false, "Destroy any existing instances tagged with our InstanceSetID, instead of erroring out")
	shellCommand := flags.String("command", "", "Run an interactive shell command on the test instance when it boots")
	pauseBeforeDestroy := flags.Bool("pause-before-destroy", false, "Prompt and wait before destroying the test instance")
	if ok, code := cmd.ParseFlags(flags, prog, args, "", stderr); !ok {
		return code
	}
	logger := ctxlog.New(stderr, "text", "info")
	defer func() {
		if err != nil {
			logger.WithError(err).Error("fatal")
			// suppress output from the other error-printing func
			err = nil
		}
		logger.Info("exiting")
	}()

	loader := config.NewLoader(stdin, logger)
	loader.Path = *configFile
	cfg, err := loader.Load()
	if err != nil {
		return 1
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return 1
	}
	key, err := ssh.ParsePrivateKey([]byte(cluster.Containers.DispatchPrivateKey))
	if err != nil {
		err = fmt.Errorf("error parsing configured Containers.DispatchPrivateKey: %s", err)
		return 1
	}
	driver, ok := dispatchcloud.Drivers[cluster.Containers.CloudVMs.Driver]
	if !ok {
		err = fmt.Errorf("unsupported cloud driver %q", cluster.Containers.CloudVMs.Driver)
		return 1
	}
	if *imageID == "" {
		*imageID = cluster.Containers.CloudVMs.ImageID
	}
	it, err := chooseInstanceType(cluster, *instanceType)
	if err != nil {
		return 1
	}
	tags := cloud.SharedResourceTags(cluster.Containers.CloudVMs.ResourceTags)
	tagKeyPrefix := cluster.Containers.CloudVMs.TagKeyPrefix
	tags[tagKeyPrefix+"CloudTestPID"] = fmt.Sprintf("%d", os.Getpid())
	if !(&tester{
		Logger:           logger,
		Tags:             tags,
		TagKeyPrefix:     tagKeyPrefix,
		SetID:            cloud.InstanceSetID(*instanceSetID),
		DestroyExisting:  *destroyExisting,
		ProbeInterval:    cluster.Containers.CloudVMs.ProbeInterval.Duration(),
		SyncInterval:     cluster.Containers.CloudVMs.SyncInterval.Duration(),
		TimeoutBooting:   cluster.Containers.CloudVMs.TimeoutBooting.Duration(),
		Driver:           driver,
		DriverParameters: cluster.Containers.CloudVMs.DriverParameters,
		ImageID:          cloud.ImageID(*imageID),
		InstanceType:     it,
		SSHKey:           key,
		SSHPort:          cluster.Containers.CloudVMs.SSHPort,
		BootProbeCommand: cluster.Containers.CloudVMs.BootProbeCommand,
		ShellCommand:     *shellCommand,
		PauseBeforeDestroy: func() {
			if *pauseBeforeDestroy {
				logger.Info("waiting for operator to press Enter")
				fmt.Fprint(stderr, "Press Enter to continue: ")
				bufio.NewReader(stdin).ReadString('\n')
			}
		},
	}).Run() {
		return 1
	}
	return 0
}

// Return the named instance type, or the cheapest type if name=="".
func chooseInstanceType(cluster *arvados.Cluster, name string) (arvados.InstanceType, error) {
	if len(cluster.InstanceTypes) == 0 {
		return arvados.InstanceType{}, errors.New("no instance types are configured")
	} else if name == "" {
		first := true
		var best arvados.InstanceType
		for _, it := range cluster.InstanceTypes {
			if first || best.Price > it.Price {
				best = it
				first = false
			}
		}
		return best, nil
	} else if it, ok := cluster.InstanceTypes[name]; !ok {
		return it, fmt.Errorf("requested instance type %q is not configured", name)
	} else {
		return it, nil
	}
}
