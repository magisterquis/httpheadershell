// Program httpheadershell - Shell over two HTTP streams
package main

/*
 * httpheadershell.go
 * Shell over two HTTP streams
 * By J. Stuart McMurray
 * Created 20230713
 * Last Modified 20230713
 */

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/http2"
)

// URL suffixes, for connecting shell's stdio to our stdio.
const (
	stdinSuffix  = "i"
	stdoutSuffix = "o"
	stderrSuffix = "e"
	bidirSuffix  = "b"
)

func main() {
	/* Command-line flags. */
	var (
		domain = flag.String(
			"letsencrypt",
			"",
			"Comma-separated Let's Encrypt certificate "+
				"provisoning `whitelist`",
		)
		staging = flag.Bool(
			"staging",
			false,
			"Use Let's Encrypt's staging server",
		)
		certDir = flag.String(
			"lecache",
			".httpheadershell.certs",
			"Let's Encrypt certificate cache `directory`",
		)
		lAddr = flag.String(
			"listen",
			"0.0.0.0:443",
			"Listen `address`",
		)
		path = flag.String(
			"path",
			"/httpheadershell-",
			"Stdio proxy URL path `prefix`",
		)
		plaintext = flag.Bool(
			"plaintext",
			false,
			"Serve plaintext HTTP, not HTTPS",
		)
		certFile = flag.String(
			"cert",
			"",
			"Optional TLS certificate `file`",
		)
		keyFile = flag.String(
			"key",
			"",
			"Optional TLS key `file`",
		)
		leEmail = flag.String(
			"email",
			"",
			"Optional email `address` for Let's Encrypt "+
				"certificate provisioning",
		)
	)
	flag.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			`Usage: %s [options]

Proxies stdio to HTTP clients, meant for catching shells but with real HTTP
headers on the wire.

URL Paths:

%s - Stdin will be sent via the body of the response
%s - Request body will be written to stdout
%s - Request body will be written to stderr
%s - Stdin will be sent via the body of the response AND
%s   Request body will be written to stdout

Use of -letsencrypt constitutes accepting Let's Encrypt's terms and conditions.

Options:
`,
			os.Args[0],
			*path+stdinSuffix,
			*path+stdoutSuffix,
			*path+stderrSuffix,
			*path+bidirSuffix,
			strings.Repeat(" ", len(*path+bidirSuffix)),
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	/* Set up HTTP handling. */
	*path = "/" + strings.TrimPrefix(*path, "/")
	http.HandleFunc(*path+stdinSuffix, HandleIn)
	http.HandleFunc(*path+stdoutSuffix, HandleOut)
	http.HandleFunc(*path+stderrSuffix, HandleErr)
	http.HandleFunc(*path+bidirSuffix, HandleBidir)

	/* Roll a TLS config, if we're TLSing. */
	var tlsConfig *tls.Config
	if !*plaintext {
		tlsConfig = MustTLSConfig(
			*domain,
			*staging,
			*certDir,
			*leEmail,
			*certFile,
			*keyFile,
		)
	}

	/* Start listening. */
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", *lAddr); nil != err {
		log.Fatalf("Unable to listen on %s: %s", *lAddr, err)
	}
	if !*plaintext {
		listener = tls.NewListener(listener, tlsConfig)
	}
	log.Printf("Listening on %s", listener.Addr())

	/* Start reading from stdin. */
	go ReadStdin()

	/* Start HTTP service. */
	server := new(http.Server)
	if err := http2.ConfigureServer(server, nil); nil != err {
		log.Fatalf("Error configuring HTTP/2: %s", err)
	}
	log.Fatalf("Fatal error: %s", server.Serve(listener))
}
