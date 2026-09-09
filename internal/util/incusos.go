package util

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"

	incusOSAPI "github.com/lxc/incus-os/incus-osd/api"
	incusAPI "github.com/lxc/incus/v7/shared/api"
)

// incusOSTimeout is the maximum time to wait for a response from the Incus OS API.
const incusOSTimeout = 5 * time.Second

// incusOSResponseSizeLimit is the maximum amount of Incus OS response data we accept.
const incusOSResponseSizeLimit = 1024 * 1024

// incusOSCACertificatesTTL is how long the CA certificates fetched from Incus OS remain valid.
const incusOSCACertificatesTTL = time.Minute

var incusOSCACertificates struct {
	mu      sync.Mutex
	certs   []string
	expires time.Time
}

// SystemCACertificates returns the PEM encoded CA certificates trusted at the system level, which
// aren't part of our own trust store. Currently this only covers the custom CA certificates
// configured in Incus OS, so nothing is returned when we aren't running on Incus OS.
func SystemCACertificates() []string {
	if !IsIncusOS() {
		return nil
	}

	incusOSCACertificates.mu.Lock()
	defer incusOSCACertificates.mu.Unlock()

	if time.Now().Before(incusOSCACertificates.expires) {
		return slices.Clone(incusOSCACertificates.certs)
	}

	certs, err := fetchIncusOSCACertificates()
	if err != nil {
		// Keep using the certificates we last fetched, if any, but don't retry until the next interval.
		slog.Warn("Failed to fetch CA certificates from Incus OS", slog.Any("err", err))
	} else {
		incusOSCACertificates.certs = certs
	}

	incusOSCACertificates.expires = time.Now().Add(incusOSCACertificatesTTL)

	return slices.Clone(incusOSCACertificates.certs)
}

// fetchIncusOSCACertificates queries the Incus OS security API over its unix socket.
func fetchIncusOSCACertificates() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), incusOSTimeout)
	defer cancel()

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _ string, _ string) (net.Conn, error) {
				var d net.Dialer

				return d.DialContext(ctx, "unix", IncusOSSocket)
			},
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://incus-os/1.0/system/security", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Incus OS returned %q when fetching the system security configuration", resp.Status)
	}

	var apiResp incusAPI.Response

	err = json.NewDecoder(io.LimitReader(resp.Body, incusOSResponseSizeLimit)).Decode(&apiResp)
	if err != nil {
		return nil, err
	}

	var security incusOSAPI.SystemSecurity

	err = apiResp.MetadataAsStruct(&security)
	if err != nil {
		return nil, err
	}

	return security.Config.CustomCACerts, nil
}
