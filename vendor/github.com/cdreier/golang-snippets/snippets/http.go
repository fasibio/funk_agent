package snippets

import (
	"crypto/tls"
	"net/http"
	"time"
)

// CreateHTTPServer creates a http server, based on the recommendations from https://blog.cloudflare.com/exposing-go-on-the-internet/
func CreateHTTPServer(addr string, hand http.Handler) *http.Server {
	return &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		TLSConfig: &tls.Config{
			PreferServerCipherSuites: true,
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				tls.X25519,
			},
		},
		Addr:    addr,
		Handler: hand,
	}
}
