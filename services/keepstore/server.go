// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type server struct {
	http.Server

	// channel (size=1) with the current keypair
	currentCert chan *tls.Certificate
}

func (srv *server) Serve(l net.Listener) error {
	if theConfig.TLSCertificateFile == "" && theConfig.TLSKeyFile == "" {
		return srv.Server.Serve(l)
	}
	// https://blog.gopheracademy.com/advent-2016/exposing-go-on-the-internet/
	srv.TLSConfig = &tls.Config{
		GetCertificate:           srv.getCertificate,
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
	}
	srv.currentCert = make(chan *tls.Certificate, 1)
	go srv.refreshCertificate(theConfig.TLSCertificateFile, theConfig.TLSKeyFile)
	return srv.Server.ServeTLS(l, "", "")
}

func (srv *server) refreshCertificate(certfile, keyfile string) {
	cert, err := tls.LoadX509KeyPair(certfile, keyfile)
	if err != nil {
		log.WithError(err).Fatal("error loading X509 key pair")
	}
	srv.currentCert <- &cert

	reload := make(chan os.Signal, 1)
	signal.Notify(reload, syscall.SIGHUP)
	for range reload {
		cert, err := tls.LoadX509KeyPair(certfile, keyfile)
		if err != nil {
			log.WithError(err).Warn("error loading X509 key pair")
			continue
		}
		// Throw away old cert and start using new one
		<-srv.currentCert
		srv.currentCert <- &cert
	}
}

func (srv *server) getCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	if srv.currentCert == nil {
		panic("srv.currentCert not initialized")
	}
	cert := <-srv.currentCert
	srv.currentCert <- cert
	return cert, nil
}
