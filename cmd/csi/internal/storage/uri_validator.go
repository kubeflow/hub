package storage

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var (
	ErrArtifactURIBlocked    = errors.New("artifact URI targets a blocked endpoint")
	ErrArtifactURINotAllowed = errors.New("artifact URI does not match any allowed prefix")
)

var blockedHosts = []string{
	"169.254.169.254",
	"fd00:ec2::254",
	"metadata.google.internal",
	"metadata.goog",
}

type URIValidator struct {
	AllowedPrefixes []string
}

func NewURIValidator(allowedPrefixes []string) *URIValidator {
	var filtered []string
	for _, p := range allowedPrefixes {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	return &URIValidator{
		AllowedPrefixes: filtered,
	}
}

func (v *URIValidator) Validate(artifactURI string) error {
	parsed, err := url.Parse(artifactURI)
	if err != nil {
		return fmt.Errorf("%w: unable to parse URI %q: %v", ErrArtifactURIBlocked, artifactURI, err)
	}

	host := parsed.Host
	if host == "" {
		host = parsed.Opaque
		if idx := strings.Index(host, "/"); idx != -1 {
			host = host[:idx]
		}
	}

	if host != "" && isBlockedHost(host) {
		return fmt.Errorf("%w: host %q is a known cloud metadata or link-local endpoint", ErrArtifactURIBlocked, host)
	}

	if len(v.AllowedPrefixes) == 0 {
		return nil
	}

	for _, prefix := range v.AllowedPrefixes {
		if strings.HasPrefix(artifactURI, prefix) {
			return nil
		}
	}

	return fmt.Errorf("%w: URI %q does not match any entry in allowlist %v", ErrArtifactURINotAllowed, artifactURI, v.AllowedPrefixes)
}

func isBlockedHost(host string) bool {
	hostname := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostname = h
	}
	hostname = strings.TrimPrefix(strings.TrimSuffix(hostname, "]"), "[")
	hostname = strings.TrimSuffix(hostname, ".")
	hostname = strings.ToLower(hostname)

	for _, blocked := range blockedHosts {
		if hostname == blocked {
			return true
		}
	}

	if ip := net.ParseIP(hostname); ip != nil {
		if ip.IsLinkLocalUnicast() {
			return true
		}
	}

	return false
}
