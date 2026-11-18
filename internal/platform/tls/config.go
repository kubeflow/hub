package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
)

type TLSConfig struct {
	CertPath         string
	KeyPath          string
	RootCertPath     string
	CAPath           string
	Cipher           string
	VerifyServerCert bool
}

func NewTLSConfig(certPath, keyPath, rootCertPath, caPath, cipher string, verifyServerCert bool) *TLSConfig {
	return &TLSConfig{
		CertPath:         certPath,
		KeyPath:          keyPath,
		RootCertPath:     rootCertPath,
		CAPath:           caPath,
		Cipher:           cipher,
		VerifyServerCert: verifyServerCert,
	}
}

func (c *TLSConfig) BuildTLSConfig() (*tls.Config, error) {
	var rootCAs *x509.CertPool
	var err error

	if c.RootCertPath != "" || c.CAPath != "" {
		rootCAs = x509.NewCertPool()

		if c.RootCertPath != "" {
			rootCert, err := os.ReadFile(c.RootCertPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read SSL root certificate from %s: %w", c.RootCertPath, err)
			}
			if !rootCAs.AppendCertsFromPEM(rootCert) {
				return nil, fmt.Errorf("failed to parse SSL root certificate from %s", c.RootCertPath)
			}
		}

		if c.CAPath != "" && c.CAPath != c.RootCertPath {
			caCert, err := os.ReadFile(c.CAPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read SSL CA certificate from %s: %w", c.CAPath, err)
			}
			if !rootCAs.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse SSL CA certificate from %s", c.CAPath)
			}
		}
	} else if c.VerifyServerCert {
		rootCAs, err = x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("failed to get system cert pool: %w", err)
		}
	}

	tlsConfig := &tls.Config{
		RootCAs:            rootCAs,
		InsecureSkipVerify: !c.VerifyServerCert,
		MinVersion:         tls.VersionTLS12,
	}

	// InsecureSkipVerify turns off chain verification too, so a configured CA
	// would otherwise be ignored. VerifyServerCert only controls the host name
	// check: verify the chain against the CA here, like the PostgreSQL
	// connector does for sslmode=require with an sslrootcert (verify-ca).
	if rootCAs != nil && !c.VerifyServerCert {
		tlsConfig.VerifyConnection = verifyPeerChain(rootCAs)
	}

	if c.CertPath != "" && c.KeyPath != "" {
		cert, err := tls.LoadX509KeyPair(c.CertPath, c.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load SSL certificate pair (cert: %s, key: %s): %w",
				c.CertPath, c.KeyPath, err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if c.Cipher != "" {
		cipherSuites, err := parseCipherSuites(c.Cipher)
		if err != nil {
			return nil, err
		}

		if len(cipherSuites) > 0 {
			tlsConfig.CipherSuites = cipherSuites
		}
	}

	return tlsConfig, nil
}

// verifyPeerChain returns a VerifyConnection callback that checks the server's
// certificate chain against roots without checking the host name.
func verifyPeerChain(roots *x509.CertPool) func(tls.ConnectionState) error {
	return func(cs tls.ConnectionState) error {
		if len(cs.PeerCertificates) == 0 {
			return fmt.Errorf("server did not present a certificate")
		}
		opts := x509.VerifyOptions{
			Roots:         roots,
			Intermediates: x509.NewCertPool(),
		}
		for _, cert := range cs.PeerCertificates[1:] {
			opts.Intermediates.AddCert(cert)
		}
		if _, err := cs.PeerCertificates[0].Verify(opts); err != nil {
			return fmt.Errorf("failed to verify server certificate: %w", err)
		}
		return nil
	}
}

// parseCipherSuites parses a colon-separated list of cipher suite names
// and returns the corresponding cipher suite IDs.
func parseCipherSuites(cipherStr string) ([]uint16, error) {
	if cipherStr == "" {
		return nil, nil
	}

	cipherMap := map[string]uint16{
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":       tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_AES_128_GCM_SHA256":                        tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384":                        tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256":                  tls.TLS_CHACHA20_POLY1305_SHA256,
	}

	var cipherSuites []uint16

	for cipher := range strings.SplitSeq(strings.TrimSpace(cipherStr), ":") {
		cipher = strings.TrimSpace(cipher)

		for _, insecureCipher := range tls.InsecureCipherSuites() {
			if insecureCipher.Name == cipher {
				return nil, fmt.Errorf("selected cipher suite is insecure: %s", insecureCipher.Name)
			}
		}

		if cipherID, exists := cipherMap[cipher]; exists {
			cipherSuites = append(cipherSuites, cipherID)
		} else {
			return nil, fmt.Errorf("invalid cipher suite: %s", cipher)
		}
	}

	return cipherSuites, nil
}
