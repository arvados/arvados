// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package recovercollection

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

	"git.arvados.org/arvados.git/lib/cmd"
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
	loader.SkipLegacy = true

	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), `Usage:
	%s [options ...] { /path/to/manifest.txt | log-or-collection-uuid } [...]

	This program recovers deleted collections. Recovery is
	possible when the collection's manifest is still available and
	all of its data blocks are still available or recoverable
	(e.g., garbage collection is not enabled, the blocks are too
	new for garbage collection, the blocks are referenced by other
	collections, or the blocks have been trashed but not yet
	deleted).

	There are multiple ways to specify a collection to recover:

        * Path to a local file containing a manifest with the desired
	  data

	* UUID of an Arvados log entry, typically a "delete" or
	  "update" event, whose "old attributes" have a manifest with
	  the desired data

	* UUID of an Arvados collection whose most recent log entry,
          typically a "delete" or "update" event, has the desired
          data in its "old attributes"

	For each provided collection manifest, once all data blocks
	are recovered/protected from garbage collection, a new
	collection is saved and its UUID is printed on stdout.

	Restored collections will belong to the system (root) user.

	Exit status will be zero if recovery is successful, i.e., a
	collection is saved for each provided manifest.
Options:
`, prog)
		flags.PrintDefaults()
	}
	loader.SetupFlags(flags)
	loglevel := flags.String("log-level", "info", "logging level (debug, info, ...)")
	if ok, code := cmd.ParseFlags(flags, prog, args, "source [...]", stderr); !ok {
		return code
	} else if flags.NArg() == 0 {
		fmt.Fprintf(stderr, "missing required arguments (try -help)\n")
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
	rcvr := recoverer{
		client:  client,
		cluster: cluster,
		logger:  logger,
	}

	exitcode := 0
	for _, src := range flags.Args() {
		logger := logger.WithField("src", src)
		var mtxt string
		if !strings.Contains(src, "/") && len(src) == 27 && src[5] == '-' && src[11] == '-' {
			var filters []arvados.Filter
			if src[5:12] == "-57u5n-" {
				filters = []arvados.Filter{{"uuid", "=", src}}
			} else if src[5:12] == "-4zz18-" {
				filters = []arvados.Filter{{"object_uuid", "=", src}}
			} else {
				logger.Error("looks like a UUID but not a log or collection UUID (if it's really a file, prepend './')")
				exitcode = 1
				continue
			}
			var resp struct {
				Items []struct {
					UUID       string    `json:"uuid"`
					EventType  string    `json:"event_type"`
					EventAt    time.Time `json:"event_at"`
					ObjectUUID string    `json:"object_uuid"`
					Properties struct {
						OldAttributes struct {
							ManifestText string `json:"manifest_text"`
						} `json:"old_attributes"`
					} `json:"properties"`
				}
			}
			err = client.RequestAndDecode(&resp, "GET", "arvados/v1/logs", nil, arvados.ListOptions{
				Limit:   1,
				Order:   []string{"event_at desc"},
				Filters: filters,
			})
			if err != nil {
				logger.WithError(err).Error("error looking up log entry")
				exitcode = 1
				continue
			} else if len(resp.Items) == 0 {
				logger.Error("log entry not found")
				exitcode = 1
				continue
			}
			logent := resp.Items[0]
			logger.WithFields(logrus.Fields{
				"uuid":                logent.UUID,
				"old_collection_uuid": logent.ObjectUUID,
				"logged_event_type":   logent.EventType,
				"logged_event_time":   logent.EventAt,
				"logged_object_uuid":  logent.ObjectUUID,
			}).Info("loaded log entry")
			mtxt = logent.Properties.OldAttributes.ManifestText
			if mtxt == "" {
				logger.Error("log entry properties.old_attributes.manifest_text missing or empty")
				exitcode = 1
				continue
			}
		} else {
			buf, err := ioutil.ReadFile(src)
			if err != nil {
				logger.WithError(err).Error("failed to load manifest data from file")
				exitcode = 1
				continue
			}
			mtxt = string(buf)
		}
		uuid, err := rcvr.RecoverManifest(string(mtxt))
		if err != nil {
			logger.WithError(err).Error("recovery failed")
			exitcode = 1
			continue
		}
		logger.WithField("UUID", uuid).Info("recovery succeeded")
		fmt.Fprintln(stdout, uuid)
	}
	return exitcode
}

type recoverer struct {
	client  *arvados.Client
	cluster *arvados.Cluster
	logger  logrus.FieldLogger
}

var errNotFound = errors.New("not found")

// Finds the timestamp of the newest copy of blk on svc. Returns
// errNotFound if blk is not on svc at all.
func (rcvr recoverer) newestMtime(ctx context.Context, logger logrus.FieldLogger, blk string, svc arvados.KeepService) (time.Time, error) {
	found, err := svc.Index(ctx, rcvr.client, blk)
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

// Ensures the given block exists on the given server and won't be
// eligible for trashing until after our chosen deadline (blobsigexp).
// Returns an error if the block doesn't exist on the given server, or
// has an old timestamp and can't be updated.
//
// After we decide a block is "safe" (whether or not we had to untrash
// it), keep-balance might notice that it's currently unreferenced and
// decide to trash it, all before our recovered collection gets
// saved. But if the block's timestamp is more recent than blobsigttl,
// keepstore will refuse to trash it even if told to by keep-balance.
func (rcvr recoverer) ensureSafe(ctx context.Context, logger logrus.FieldLogger, blk string, svc arvados.KeepService, blobsigttl time.Duration, blobsigexp time.Time) error {
	if latest, err := rcvr.newestMtime(ctx, logger, blk, svc); err != nil {
		return err
	} else if latest.Add(blobsigttl).After(blobsigexp) {
		return nil
	}
	if err := svc.Touch(ctx, rcvr.client, blk); err != nil {
		return fmt.Errorf("error updating timestamp: %s", err)
	}
	logger.Debug("updated timestamp")
	if latest, err := rcvr.newestMtime(ctx, logger, blk, svc); err == errNotFound {
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
func (rcvr recoverer) RecoverManifest(mtxt string) (string, error) {
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
	err = rcvr.client.EachKeepService(func(svc arvados.KeepService) error {
		if svc.ServiceType == "proxy" {
			rcvr.logger.WithField("service", svc).Debug("ignore proxy service")
		} else {
			services = append(services, svc)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("error getting list of keep services: %s", err)
	}
	rcvr.logger.WithField("services", services).Debug("got list of services")

	// blobsigexp is our deadline for saving the rescued
	// collection. This must be less than BlobSigningTTL
	// (otherwise our rescued blocks could be garbage collected
	// again before we protect them by saving the collection) but
	// the exact value is somewhat arbitrary. If it's too soon, it
	// will arrive before we're ready to save, and save will
	// fail. If it's too late, we'll needlessly update timestamps
	// on some blocks that were recently written/touched (e.g., by
	// a previous attempt to rescue this same collection) and
	// would have lived long enough anyway if left alone.
	// BlobSigningTTL/2 (typically around 1 week) is much longer
	// than than we need to recover even a very large collection.
	blobsigttl := rcvr.cluster.Collections.BlobSigningTTL.Duration()
	blobsigexp := time.Now().Add(blobsigttl / 2)
	rcvr.logger.WithField("blobsigexp", blobsigexp).Debug("chose save deadline")

	// We'll start a number of threads, each working on
	// checking/recovering one block at a time. The threads
	// themselves don't need much CPU/memory, but to avoid hitting
	// limits on keepstore connections, backend storage bandwidth,
	// etc., we limit concurrency to 2 per keepstore node.
	workerThreads := 2 * len(services)

	blkFound := make([]bool, len(blks))
	var wg sync.WaitGroup
	for i := 0; i < workerThreads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
		nextblk:
			for idx := range todo {
				blk := strings.SplitN(string(blks[idx]), "+", 2)[0]
				logger := rcvr.logger.WithField("block", blk)
				for _, untrashing := range []bool{false, true} {
					for _, svc := range services {
						logger := logger.WithField("service", fmt.Sprintf("%s:%d", svc.ServiceHost, svc.ServicePort))
						if untrashing {
							if err := svc.Untrash(ctx, rcvr.client, blk); err != nil {
								logger.WithError(err).Debug("untrash failed")
								continue
							}
							logger.Info("untrashed")
						}
						err := rcvr.ensureSafe(ctx, logger, blk, svc, blobsigttl, blobsigexp)
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
			rcvr.logger.Warn("partial recovery is not implemented")
		}
		return "", fmt.Errorf("unable to recover %d of %d blocks", havenot, have+havenot)
	}

	if rcvr.cluster.Collections.BlobSigning {
		key := []byte(rcvr.cluster.Collections.BlobSigningKey)
		coll.ManifestText = arvados.SignManifest(coll.ManifestText, rcvr.client.AuthToken, blobsigexp, blobsigttl, key)
	}
	rcvr.logger.WithField("manifest", coll.ManifestText).Debug("updated blob signatures in manifest")
	err = rcvr.client.RequestAndDecodeContext(ctx, &coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]interface{}{
			"manifest_text": coll.ManifestText,
		},
	})
	if err != nil {
		return "", fmt.Errorf("error saving new collection: %s", err)
	}
	rcvr.logger.WithField("UUID", coll.UUID).Debug("created new collection")
	return coll.UUID, nil
}
