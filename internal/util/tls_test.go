package util_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FuturFusion/migration-manager/internal/util"
)

func TestTLSClientConfig(t *testing.T) {
	caPEM, caCert := generateCACertificate(t)

	tests := []struct {
		name           string
		caCertificates []string
		assertErr      require.ErrorAssertionFunc
		wantTrusted    bool
	}{
		{
			name:      "no CA certificates",
			assertErr: require.NoError,
		},
		{
			name:           "valid CA certificate",
			caCertificates: []string{caPEM},
			assertErr:      require.NoError,
			wantTrusted:    true,
		},
		{
			name:           "invalid CA certificate",
			caCertificates: []string{"not a certificate"},
			assertErr:      require.Error,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tlsConfig, err := util.TLSClientConfig(nil, tc.caCertificates)
			tc.assertErr(t, err)

			if err != nil {
				return
			}

			_, err = caCert.Verify(x509.VerifyOptions{Roots: tlsConfig.RootCAs})
			if tc.wantTrusted {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func generateCACertificate(t *testing.T) (string, *x509.Certificate) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Migration Manager Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), cert
}
