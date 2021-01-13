// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package selfsigned

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"time"
)

type CertGenerator struct {
	Bits  int
	Hosts []string
	IsCA  bool
}

func (gen CertGenerator) Generate() (cert tls.Certificate, err error) {
	keyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	if gen.IsCA {
		keyUsage |= x509.KeyUsageCertSign
	}
	notBefore := time.Now()
	notAfter := time.Now().Add(time.Hour * 24 * 365)
	snMax := new(big.Int).Lsh(big.NewInt(1), 128)
	sn, err := rand.Int(rand.Reader, snMax)
	if err != nil {
		err = fmt.Errorf("Failed to generate serial number: %w", err)
		return
	}
	template := x509.Certificate{
		SerialNumber: sn,
		Subject: pkix.Name{
			Organization: []string{"N/A"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  gen.IsCA,
	}
	for _, h := range gen.Hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}
	bits := gen.Bits
	if bits == 0 {
		bits = 4096
	}
	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		err = fmt.Errorf("error generating key: %w", err)
		return
	}
	certder, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		err = fmt.Errorf("error creating certificate: %w", err)
		return
	}
	cert = tls.Certificate{
		Certificate: [][]byte{certder},
		PrivateKey:  priv,
	}
	return
}
