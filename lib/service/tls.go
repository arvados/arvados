// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package service

import (
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
)

func tlsConfigWithCertUpdater(cluster *arvados.Cluster, logger logrus.FieldLogger) (*tls.Config, error) {
	currentCert := make(chan *tls.Certificate, 1)
	loaded := false

	key, cert := cluster.TLS.Key, cluster.TLS.Certificate
	if !strings.HasPrefix(key, "file://") || !strings.HasPrefix(cert, "file://") {
		return nil, errors.New("cannot use TLS certificate: TLS.Key and TLS.Certificate must be specified as file://...")
	}
	key, cert = key[7:], cert[7:]

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

	go func() {
		reload := make(chan os.Signal, 1)
		signal.Notify(reload, syscall.SIGHUP)
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
