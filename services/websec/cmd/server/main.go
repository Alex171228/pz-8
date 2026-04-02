package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"pz1.2/services/websec/internal/auth"
	"pz1.2/services/websec/internal/httpapi"
	"pz1.2/services/websec/internal/store"
)

func main() {
	httpsFlag := flag.Bool("https", false, "enable HTTPS with self-signed certificate")
	flag.Parse()

	st := store.New()

	exe, _ := os.Executable()
	baseDir := filepath.Dir(exe)
	templatesDir := filepath.Join(baseDir, "services", "websec", "templates")
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		templatesDir = filepath.Join(cwd, "services", "websec", "templates")
	}

	handler, err := httpapi.NewHandler(st, templatesDir)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/login", handler.Login)
	mux.HandleFunc("/logout", handler.Logout)
	mux.HandleFunc("/profile", handler.Profile)
	mux.HandleFunc("/hello", handler.Hello)
	mux.HandleFunc("/comments", handler.Comments)

	if *httpsFlag {
		auth.SecureMode = true
		cert, err := generateSelfSignedCert()
		if err != nil {
			log.Fatal("failed to generate certificate:", err)
		}

		srv := &http.Server{
			Addr:    ":8443",
			Handler: mux,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
			},
		}

		log.Println("HTTPS server started on https://localhost:8443")
		log.Println("open https://localhost:8443/login")
		log.Println("cookie Secure=true, SameSite=Strict")
		if err := srv.ListenAndServeTLS("", ""); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Println("HTTP server started on http://localhost:8080")
		log.Println("open http://localhost:8080/login")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatal(err)
		}
	}
}

func generateSelfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}, nil
}
