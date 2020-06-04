// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package undelete

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	"time"

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
			uuid, err := und.RecoverManifest(string(mtxt))
			if err != nil {
				logger.WithError(err).Error("recovery failed")
				exitcode = 1
				continue
			}
			logger.WithField("UUID", uuid).Info("recovery succeeded")
			fmt.Fprintln(stdout, uuid)
		}
	}
	return exitcode
}

type undeleter struct {
	client  *arvados.Client
	cluster *arvados.Cluster
	logger  logrus.FieldLogger
}

var errNotFound = errors.New("not found")

// Return the timestamp of the newest copy of blk on svc. Second
// return value is false if blk is not on svc at all, or an error
// occurs.
func (und undeleter) newestMtime(logger logrus.FieldLogger, blk string, svc arvados.KeepService) (time.Time, error) {
	found, err := svc.Index(und.client, blk)
	if err != nil {
		logger.WithError(err).Warn("error getting index")
		return time.Time{}, err
	} else if len(found) == 0 {
		return time.Time{}, errNotFound
	}
	var latest time.Time
	for _, ent := range found {
		t := time.Unix(0, ent.Mtime)
		if t.After(latest) {
			latest = t
		}
	}
	logger.WithField("latest", latest).Debug("found")
	return latest, nil
}

var errTouchIneffective = errors.New("(BUG?) touch succeeded but had no effect -- reported timestamp is still too old")

// Ensure the given block exists on the given server and won't be
// eligible for trashing until after our chosen deadline (blobsigexp).
// Returns an error if the block doesn't exist on the given server, or
// has an old timestamp and can't be updated.  Reports errors via
// logger.
//
// After we decide a block is "safe" (whether or not we had to untrash
// it), keep-balance might notice that it's currently unreferenced and
// decide to trash it, all before our recovered collection gets
// saved. But if the block's timestamp is more recent than blobsigttl,
// keepstore will refuse to trash it even if told to by keep-balance.
func (und undeleter) ensureSafe(ctx context.Context, logger logrus.FieldLogger, blk string, svc arvados.KeepService, blobsigttl time.Duration, blobsigexp time.Time) error {
	if latest, err := und.newestMtime(logger, blk, svc); err != nil {
		return err
	} else if latest.Add(blobsigttl).After(blobsigexp) {
		return nil
	}
	if err := svc.Touch(ctx, und.client, blk); err != nil {
		return fmt.Errorf("error updating timestamp: %s", err)
	}
	logger.Debug("updated timestamp")
	if latest, err := und.newestMtime(logger, blk, svc); err == errNotFound {
		return fmt.Errorf("(BUG?) touch succeeded, but then block did not appear in index")
	} else if err != nil {
		return err
	} else if latest.Add(blobsigttl).After(blobsigexp) {
		return nil
	} else {
		return errTouchIneffective
	}
}

// Untrash and update GC timestamps (as needed) on blocks referenced
// by the given manifest, save a new collection and return the new
// collection's UUID.
func (und undeleter) RecoverManifest(mtxt string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	coll := arvados.Collection{ManifestText: mtxt}
	blks, err := coll.SizedDigests()
	if err != nil {
		return "", err
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
		return "", fmt.Errorf("error getting list of keep services: %s", err)
	}
	und.logger.WithField("services", services).Debug("got list of services")

	// Choose a deadline for saving a rescued collection.
	blobsigttl := und.cluster.Collections.BlobSigningTTL.Duration()
	blobsigexp := time.Now().Add(blobsigttl / 2)
	und.logger.WithField("blobsigexp", blobsigexp).Debug("chose save deadline")

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
				for _, untrashing := range []bool{false, true} {
					for _, svc := range services {
						logger := logger.WithField("service", fmt.Sprintf("%s:%d", svc.ServiceHost, svc.ServicePort))
						if untrashing {
							if err := svc.Untrash(ctx, und.client, blk); err != nil {
								logger.WithError(err).Debug("untrash failed")
								continue
							}
							logger.Info("untrashed")
						}
						err := und.ensureSafe(ctx, logger, blk, svc, blobsigttl, blobsigexp)
						if err == errNotFound {
							logger.Debug(err)
						} else if err != nil {
							logger.Error(err)
						} else {
							blkFound[idx] = true
							continue nextblk
						}
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
		return "", fmt.Errorf("unable to recover %d of %d blocks", havenot, have+havenot)
	}

	if und.cluster.Collections.BlobSigning {
		key := []byte(und.cluster.Collections.BlobSigningKey)
		coll.ManifestText = arvados.SignManifest(coll.ManifestText, und.client.AuthToken, blobsigexp, blobsigttl, key)
	}
	und.logger.WithField("manifest", coll.ManifestText).Debug("updated blob signatures in manifest")
	err = und.client.RequestAndDecodeContext(ctx, &coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]interface{}{
			"manifest_text": coll.ManifestText,
		},
	})
	if err != nil {
		return "", fmt.Errorf("error saving new collection: %s", err)
	}
	und.logger.WithField("UUID", coll.UUID).Debug("created new collection")
	return coll.UUID, nil
}
