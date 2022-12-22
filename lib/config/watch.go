// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	_ "net/http/pprof"
	"reflect"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

// WatchConfig starts a goroutine that calls fn when the config file
// changes in such a way that the new content is loadable and
// semantically different from the previous config.
//
// After fn is called, the next call to Load() will return the updated
// config.
//
// If cfgPath is empty or "-", or the config has AutoReloadConfig
// turned off, then WatchConfig does nothing.
func (ldr *Loader) WatchConfig(ctx context.Context, fn func()) error {
	if ldr.Path == "" || ldr.Path == "-" {
		ldr.Logger.Debugf("AutoReloadConfig is disabled because config %q is not a regular file", ldr.Path)
		return nil
	}
	cfg, err := ldr.Load()
	if err != nil {
		return err
	}
	if !cfg.AutoReloadConfig {
		ldr.Logger.Debug("AutoReloadConfig is disabled, not watching config")
		return nil
	}
	var copyerr error
	pr, pw := io.Pipe()
	go func() {
		err := json.NewEncoder(pw).Encode(cfg)
		if err != nil {
			copyerr = err
		}
		err = pw.Close()
		if copyerr == nil {
			copyerr = err
		}
	}()
	cfg2 := new(arvados.Config)
	err = json.NewDecoder(pr).Decode(cfg2)
	if err != nil {
		return fmt.Errorf("error copying config: %w", err)
	}
	err = pr.Close()
	if err == nil {
		err = copyerr
	}
	if err != nil {
		return fmt.Errorf("error copying config: %w", err)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fsnotify setup failed: %w", err)
	}
	go watchConfig(ctx, ldr.Logger, ldr.Path, cfg2, watcher, func(configdata []byte) {
		ldr.configdata = configdata
		fn()
	})
	return nil
}

func watchConfig(ctx context.Context, logger logrus.FieldLogger, cfgPath string, prevcfg *arvados.Config, watcher *fsnotify.Watcher, fn func([]byte)) {
	defer watcher.Close()
	rewatch := func() {
		watcher.Remove(cfgPath)
		for delay := time.Second / 10; ; {
			err := watcher.Add(cfgPath)
			if err != nil {
				logger.WithError(err).WithField("file", cfgPath).Warn("fsnotify watch failed")
				time.Sleep(delay)
				if delay < time.Minute {
					delay = delay * 2
				}
				continue
			}
			break
		}
	}
	rewatch()
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.WithError(err).Warn("fsnotify watcher reported error")
		case _, ok := <-watcher.Events:
			if !ok {
				return
			}
			for len(watcher.Events) > 0 {
				<-watcher.Events
			}

			// We remove and re-add our watcher here so
			// that, if someone renames a new config file
			// into place (as they should), we receive
			// events about the *new* file.
			//
			// Setting up the watcher here (before reading
			// the new file) ensures we will get notified
			// in the next loop iteration if the new file
			// is changed or replaced before we even
			// finish reading it.
			rewatch()

			loader := NewLoader(&bytes.Buffer{}, &logrus.Logger{Out: ioutil.Discard})
			loader.Path = cfgPath
			loader.SkipAPICalls = true
			cfg, err := loader.Load()
			if err != nil {
				logger.WithError(err).Warn("error reloading config file after change detected; ignoring new config for now")
			} else if reflect.DeepEqual(cfg, prevcfg) {
				logger.Debug("config file changed but is still DeepEqual to the existing config")
			} else {
				logger.Debug("config changed, notifying")
				fn(loader.configdata)
				prevcfg = cfg
			}
		}
	}
}
