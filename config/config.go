package config

import (
	"crypto/tls"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

// GetHTTPClient returns an HTTP client configured with proxy if needed
func GetHTTPClient() *http.Client {
	// Настройка SOCKS5 прокси
	dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:26001", nil, proxy.Direct)
	if err != nil {
		log.Printf("Error creating SOCKS5 dialer: %v", err)
		return &http.Client{Timeout: 15 * time.Second}
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		Dial:                  dialer.Dial,
		ResponseHeaderTimeout: 15 * time.Second,
		IdleConnTimeout:       15 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}

	return client
}
