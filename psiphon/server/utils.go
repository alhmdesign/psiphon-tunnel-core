/*
 * Copyright (c) 2016, Psiphon Inc.
 * All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/Psiphon-Labs/psiphon-tunnel-core/psiphon/common"
)

// GenerateWebServerCertificate creates a self-signed web server certificate,
// using the specified host name (commonName).
// This is primarily intended for use by MeekServer to generate on-the-fly,
// self-signed TLS certificates for fronted HTTPS mode. In this case, the nature
// of the certificate is non-circumvention; it only has to be acceptable to the
// front CDN making connections to meek.
// The same certificates are used for unfronted HTTPS meek. In this case, the
// certificates may be a fingerprint used to detect Psiphon servers or traffic.
// TODO: more effort to mitigate fingerprinting these certificates.
//
// In addition, GenerateWebServerCertificate is used by GenerateConfig to create
// Psiphon web server certificates for test/example configurations. If these Psiphon
// web server certificates are used in production, the same caveats about
// fingerprints apply.
func GenerateWebServerCertificate(commonName string) (string, string, error) {

	// Based on https://golang.org/src/crypto/tls/generate_cert.go
	// TODO: use other key types: anti-fingerprint by varying params

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", common.ContextError(err)
	}

	// Validity period is ~10 years, starting some number of ~months
	// back in the last year.

	age, err := common.MakeSecureRandomInt(12)
	if err != nil {
		return "", "", common.ContextError(err)
	}
	age += 1
	validityPeriod := 10 * 365 * 24 * time.Hour
	notBefore := time.Now().Add(time.Duration(-age) * 30 * 24 * time.Hour).UTC()
	notAfter := notBefore.Add(validityPeriod).UTC()

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return "", "", common.ContextError(err)
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(rsaKey.Public())
	if err != nil {
		return "", "", common.ContextError(err)
	}
	// as per RFC3280 sec. 4.2.1.2
	subjectKeyID := sha1.Sum(publicKeyBytes)

	var subject pkix.Name
	if commonName != "" {
		subject = pkix.Name{CommonName: commonName}
	}

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               subject,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:         true,
		SubjectKeyId: subjectKeyID[:],
		MaxPathLen:   1,
		Version:      2,
	}

	derCert, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		rsaKey.Public(),
		rsaKey)
	if err != nil {
		return "", "", common.ContextError(err)
	}

	webServerCertificate := pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: derCert,
		},
	)

	webServerPrivateKey := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
		},
	)

	return string(webServerCertificate), string(webServerPrivateKey), nil
}

// IntentionalPanicError is an error type that is used
// when calling panic() in a situation where recovers
// should propagate the panic.
type IntentionalPanicError struct {
	message string
}

// NewIntentionalPanicError creates a new IntentionalPanicError.
func NewIntentionalPanicError(message string) error {
	return IntentionalPanicError{message: message}
}

// Error implements the error interface.
func (err IntentionalPanicError) Error() string {
	return err.message
}

// AddStack creates a new IntentionalPanicError which
// records the given stack. When a IntentionalPanicError is
// recovered, call AddStack with the debug.Stack() at the
// point of recovery, and panic with the resulting
// IntentionalPanicError.
func (err IntentionalPanicError) AddStack(debugStack []byte) error {
	return NewIntentionalPanicError(
		fmt.Sprintf("intentional panic error: %s\nstack: %s\n",
			err.Error(),
			string(debugStack)))
}

// PanickingLogWriter wraps an io.Writer and intentionally
// panics when a Write() fails.
type PanickingLogWriter struct {
	name   string
	writer io.Writer
}

// NewPanickingLogWriter creates a new PanickingLogWriter.
func NewPanickingLogWriter(
	name string, writer io.Writer) *PanickingLogWriter {

	return &PanickingLogWriter{
		name:   name,
		writer: writer,
	}
}

// Write implements the io.Writer interface.
func (w *PanickingLogWriter) Write(p []byte) (n int, err error) {
	n, err = w.writer.Write(p)
	if err != nil {
		panic(
			NewIntentionalPanicError(
				fmt.Sprintf("fatal write to %s failed: %s", w.name, err)))
	}
	return
}
