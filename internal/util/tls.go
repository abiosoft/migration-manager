package util

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	incusTLS "github.com/lxc/incus/v7/shared/tls"
)

// TLSClientConfig returns a TLS configuration for outgoing connections, trusting the system
// certificates, the given PEM encoded CA certificates and, if set, the pinned server certificate.
func TLSClientConfig(serverCert *x509.Certificate, caCertificates []string) (*tls.Config, error) {
	tlsConfig := &tls.Config{}
	incusTLS.TLSConfigWithTrustedCert(tlsConfig, serverCert)

	if len(caCertificates) == 0 {
		return tlsConfig, nil
	}

	if tlsConfig.RootCAs == nil {
		tlsConfig.RootCAs = x509.NewCertPool()
	} else {
		// Don't modify a potentially shared certificate pool.
		tlsConfig.RootCAs = tlsConfig.RootCAs.Clone()
	}

	for i, caCert := range caCertificates {
		if !tlsConfig.RootCAs.AppendCertsFromPEM([]byte(caCert)) {
			return nil, fmt.Errorf("Failed to parse PEM encoded CA certificate at index %d", i)
		}
	}

	return tlsConfig, nil
}
