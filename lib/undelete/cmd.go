// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package undelete

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/sirupsen/logrus"
)

var Command command

type command struct{}

func (command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var err error
	logger := ctxlog.New(stderr, "text", "info")
	defer func() {
		if err != nil {
			logger.WithError(err).Error("fatal")
		}
		logger.Info("exiting")
	}()

	loader := config.NewLoader(stdin, logger)

	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(stderr)
	loader.SetupFlags(flags)
	loglevel := flags.String("log-level", "info", "logging level (debug, info, ...)")
	err = flags.Parse(args)
	if err == flag.ErrHelp {
		err = nil
		return 0
	} else if err != nil {
		return 2
	}

	if len(flags.Args()) == 0 {
		fmt.Fprintf(stderr, "Usage: %s [options] uuid_or_file ...\n", prog)
		flags.PrintDefaults()
		return 2
	}

	lvl, err := logrus.ParseLevel(*loglevel)
	if err != nil {
		return 2
	}
	logger.SetLevel(lvl)

	cfg, err := loader.Load()
	if err != nil {
		return 1
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return 1
	}
	client, err := arvados.NewClientFromConfig(cluster)
	if err != nil {
		return 1
	}
	client.AuthToken = cluster.SystemRootToken
	und := undeleter{
		client:  client,
		cluster: cluster,
		logger:  logger,
	}

	exitcode := 0
	for _, src := range flags.Args() {
		logger := logger.WithField("src", src)
		if len(src) == 27 && src[5:12] == "-57u5n-" {
			logger.Error("log entry lookup not implemented")
			exitcode = 1
			continue
		} else {
			mtxt, err := ioutil.ReadFile(src)
			if err != nil {
				logger.WithError(err).Error("error loading manifest data")
				exitcode = 1
				continue
			}
			err = und.RecoverManifest(string(mtxt))
			if err != nil {
				logger.WithError(err).Error("recovery failed")
				exitcode = 1
				continue
			}
			logger.WithError(err).Info("recovery succeeded")
		}
	}
	return exitcode
}

type undeleter struct {
	client  *arvados.Client
	cluster *arvados.Cluster
	logger  logrus.FieldLogger
}

func (und undeleter) RecoverManifest(mtxt string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	coll := arvados.Collection{ManifestText: mtxt}
	blks, err := coll.SizedDigests()
	if err != nil {
		return err
	}
	todo := make(chan int, len(blks))
	for idx := range blks {
		todo <- idx
	}
	go close(todo)

	var services []arvados.KeepService
	err = und.client.EachKeepService(func(svc arvados.KeepService) error {
		if svc.ServiceType == "proxy" {
			und.logger.WithField("service", svc).Debug("ignore proxy service")
		} else {
			services = append(services, svc)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error getting list of keep services: %s", err)
	}
	und.logger.WithField("services", services).Debug("got list of services")

	blkFound := make([]bool, len(blks))
	var wg sync.WaitGroup
	for i := 0; i < 2*len(services); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
		nextblk:
			for idx := range todo {
				blk := strings.SplitN(string(blks[idx]), "+", 2)[0]
				logger := und.logger.WithField("block", blk)
				for _, svc := range services {
					logger := logger.WithField("service", fmt.Sprintf("%s:%d", svc.ServiceHost, svc.ServicePort))
					if found, err := svc.Index(und.client, blk); err != nil {
						logger.WithError(err).Warn("error getting index")
					} else if len(found) > 0 {
						blkFound[idx] = true
						logger.Debug("found")
						continue nextblk
					} else {
						logger.Debug("not found")
					}
				}
				for _, svc := range services {
					logger := logger.WithField("service", fmt.Sprintf("%s:%d", svc.ServiceHost, svc.ServicePort))
					if err := svc.Untrash(ctx, und.client, blk); err != nil {
						logger.WithError(err).Debug("untrash failed")
					} else {
						blkFound[idx] = true
						logger.Info("untrashed")
						continue nextblk
					}
				}
				logger.Debug("unrecoverable")
			}
		}()
	}
	wg.Wait()

	var have, havenot int
	for _, ok := range blkFound {
		if ok {
			have++
		} else {
			havenot++
		}
	}
	if havenot > 0 {
		if have > 0 {
			und.logger.Warn("partial recovery is not implemented")
		}
		return fmt.Errorf("unable to recover %d of %d blocks", havenot, have+havenot)
	}

	if und.cluster.Collections.BlobSigning {
		ttl := und.cluster.Collections.BlobSigningTTL.Duration()
		key := []byte(und.cluster.Collections.BlobSigningKey)
		coll.ManifestText = arvados.SignManifest(coll.ManifestText, und.client.AuthToken, ttl, key)
	}
	und.logger.Info(coll.ManifestText)
	err = und.client.RequestAndDecodeContext(ctx, &coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]interface{}{
			"manifest_text": coll.ManifestText,
		},
	})
	if err != nil {
		return fmt.Errorf("error saving new collection: %s", err)
	}
	und.logger.WithField("UUID", coll.UUID).Info("created new collection")
	return nil
}
