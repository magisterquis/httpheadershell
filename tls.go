package main

/*
 * tls.go
 * Roll a TLS config
 * By J. Stuart McMurray
 * Created 20230713
 * Last Modified 20230713
 */

import (
	"crypto/tls"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

const (
	// stagingURL is the Let's Encrypt staging environment URL.
	stagingURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
	// stagingSubdir is a subdirectory in the cert cache directory for
	// staging certs.
	stagingSubdir = "staging"
)

// MustTLSConfig either generates a TLS config from the Let's Encrypt info or
// using the cert and key file, or terminates the program.
func MustTLSConfig(
	domains string,
	staging bool,
	leEmail, certDir string,
	certFile, keyFile string,
) *tls.Config {
	/* If we're not let's encrypting, we'll need a cert and key. */
	if "" == domains {
		/* Make sure the user gave us something to work with. */
		if "" == certFile || "" == keyFile {
			log.Fatalf(
				"Need a Let's Encrypt domains " +
					"(-letsencrypt), certificate and " +
					"key files (-cert/-key), or to use " +
					"plaintext HTTP (-plaintext)",
			)
		}
		conf, err := tlsFileConfig(certFile, keyFile)
		if nil != err {
			log.Fatalf(
				"Error generating TLS config from "+
					"%s and %s: %s",
				certFile,
				keyFile,
				err,
			)
		}
		return conf
	}

	/* Work out the domains to use. */
	ds := strings.Split(domains, ",")
	last := 0
	for _, d := range ds {
		d = strings.TrimSpace(d)
		if "" == d {
			continue
		}
		ds[last] = d
		last++
	}
	ds = ds[:last]

	/* Work out the server URL and cert directory. */
	var client *acme.Client
	if "" == certDir {
		certDir = "."
	}
	if staging {
		certDir = filepath.Join(certDir, stagingSubdir)
		client = &acme.Client{DirectoryURL: stagingURL}
	}

	/* Get a TLS config. */
	return (&autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache(certDir),
		HostPolicy: autocert.HostWhitelist(ds...),
		Client:     client,
		Email:      leEmail,
	}).TLSConfig()
}

// tlsFileConfig rolls a TLS config which uses the given cert and key file.
func tlsFileConfig(certFile, keyFile string) (*tls.Config, error) {
	/* Load the cert and key. */
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if nil != err {
		return nil, fmt.Errorf("loading keypair: %w", err)
	}

	/* Roll into a TLS config. */
	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}
