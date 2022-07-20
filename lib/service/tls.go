// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/acme/autocert"
)

func makeTLSConfig(cluster *arvados.Cluster, logger logrus.FieldLogger) (*tls.Config, error) {
	if cluster.TLS.ACME.Server != "" {
		return makeAutocertConfig(cluster, logger)
	} else {
		return makeFileLoaderConfig(cluster, logger)
	}
}

var errCertUnavailable = errors.New("certificate unavailable, waiting for supervisor to update cache")

type readonlyDirCache autocert.DirCache

func (c readonlyDirCache) Get(ctx context.Context, name string) ([]byte, error) {
	data, err := autocert.DirCache(c).Get(ctx, name)
	if err != nil {
		// Returning an error other than autocert.ErrCacheMiss
		// causes GetCertificate() to fail early instead of
		// trying to obtain a certificate itself (which
		// wouldn't work because we're not in a position to
		// answer challenges).
		return nil, errCertUnavailable
	}
	return data, nil
}

func (c readonlyDirCache) Put(ctx context.Context, name string, data []byte) error {
	return fmt.Errorf("(bug?) (readonlyDirCache)Put(%s) called", name)
}

func (c readonlyDirCache) Delete(ctx context.Context, name string) error {
	return nil
}

func makeAutocertConfig(cluster *arvados.Cluster, logger logrus.FieldLogger) (*tls.Config, error) {
	mgr := &autocert.Manager{
		Cache:  readonlyDirCache("/var/lib/arvados/tmp/autocert"),
		Prompt: autocert.AcceptTOS,
		// HostPolicy accepts all names because this Manager
		// doesn't request certs. Whoever writes certs to our
		// cache is effectively responsible for HostPolicy.
		HostPolicy: func(ctx context.Context, host string) error { return nil },
		// Keep using whatever's in the cache as long as
		// possible. Assume some other process (see lib/boot)
		// handles renewals.
		RenewBefore: time.Second,
	}
	return mgr.TLSConfig(), nil
}

func makeFileLoaderConfig(cluster *arvados.Cluster, logger logrus.FieldLogger) (*tls.Config, error) {
	currentCert := make(chan *tls.Certificate, 1)
	loaded := false

	key := strings.TrimPrefix(cluster.TLS.Key, "file://")
	cert := strings.TrimPrefix(cluster.TLS.Certificate, "file://")

	update := func() error {
		cert, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return fmt.Errorf("error loading X509 key pair: %s", err)
		}
		if loaded {
			// Throw away old cert
			<-currentCert
		}
		currentCert <- &cert
		loaded = true
		return nil
	}
	err := update()
	if err != nil {
		return nil, err
	}

	reload := make(chan os.Signal, 1)
	signal.Notify(reload, syscall.SIGHUP)
	go func() {
		for range time.NewTicker(time.Hour).C {
			reload <- nil
		}
	}()
	go func() {
		for range reload {
			err := update()
			if err != nil {
				logger.WithError(err).Warn("error updating TLS certificate")
			}
		}
	}()

	// https://blog.gopheracademy.com/advent-2016/exposing-go-on-the-internet/
	return &tls.Config{
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			cert := <-currentCert
			currentCert <- cert
			return cert, nil
		},
	}, nil
}
