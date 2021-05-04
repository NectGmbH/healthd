package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// APPVERSION contains the version of the tool, injected by make
var APPVERSION string

// StringSlice is a typed slice of strings
type StringSlice []string

// String returns a string representation of the current string slice.
func (i *StringSlice) String() string {
	return strings.Join(*i, " ")
}

// Set appends a entry to the string slice (used for flags)
func (i *StringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	logrus.Infof("healthd v%s", APPVERSION)

	var caPath string
	var crtPath string
	var keyPath string
	var etcds StringSlice
	var httpsPort int

	flag.Var(&etcds, "etcd", "etcd endpoint where status should be persisted. Multiple can be given, e.g.: -etcd localhost:2379 -etcd localhost:22379")
	flag.StringVar(&caPath, "ca", "", "path to the ca.crt")
	flag.StringVar(&crtPath, "crt", "", "path to the client.crt")
	flag.StringVar(&keyPath, "key", "", "path to the client.key")
	flag.IntVar(&httpsPort, "port", 443, "port on which the https server should listen")
	flag.Parse()

	if len(etcds) == 0 {
		logrus.Fatal("no etcds given, pass them using -etcd")
	}

	if caPath == "" {
		logrus.Fatal("no ca certificate given, pass it using -ca")
	}

	if crtPath == "" {
		logrus.Fatal("no client certificate given, pass it using -crt")
	}

	if keyPath == "" {
		logrus.Fatal("no client key given, pass it using -key")
	}

	if httpsPort <= 0 {
		logrus.Fatalf("Invalid https port `%d` specified, pass using -port", httpsPort)
	}

	// Create a CA certificate pool and add cert.pem to it
	caCert, err := ioutil.ReadFile(caPath)
	if err != nil {
		logrus.Fatalf("couldn't read `%s`, see: %v", caPath, err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Create the TLS Config with the CA pool and enable Client certificate validation
	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	tlsConfig.BuildNameToCertificate()

	h, err := NewHealthd(etcds)
	if err != nil {
		logrus.Fatalf("couldn't create healthd service, see: %v", err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		agentName := r.Header.Get("X-Agent-Name")
		if agentName == "" {
			errStr := "request missing X-Agent-Name header, ignoring."
			logrus.Error(errStr)
			http.Error(w, errStr, http.StatusBadRequest)
			return
		}

		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			errStr := "missing tls informations"
			logrus.Error(errStr)
			http.Error(w, errStr, http.StatusBadRequest)
			return
		}

		foundName := false
		if r.TLS.PeerCertificates[0].Subject.CommonName == agentName {
			foundName = true
		} else {
			for _, n := range r.TLS.PeerCertificates[0].DNSNames {
				if n == agentName {
					foundName = true
					break
				}
			}
		}

		if !foundName {
			errStr := fmt.Sprintf("invalid agent name `%s` (not in CN or SANs)", agentName)
			logrus.Error(errStr)
			http.Error(w, errStr, http.StatusForbidden)
			return
		}

		if r.Method == http.MethodGet {
			h.GETMonitorsHandler(w, r)
		} else if r.Method == http.MethodPost {
			h.POSTStatusHandler(w, r)
		} else {
			errStr := fmt.Sprintf("invalid method `%s` expected GET or POST", r.Method)
			logrus.Error(errStr)
			http.Error(w, errStr, http.StatusBadRequest)
		}
	}

	// Set up a /hello resource handler
	http.HandleFunc("/", handler)

	// Create a Server instance to listen with the TLS config
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", httpsPort),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		TLSConfig:      tlsConfig,
	}

	// Listen to HTTPS connections with the server certificate and wait
	err = server.ListenAndServeTLS(crtPath, keyPath)
	logrus.Fatalf("received error in https server, see: %v", err)
}
