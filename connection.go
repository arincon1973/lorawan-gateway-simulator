package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func (g *Gateway) Connect() error {
	addr := fmt.Sprintf("%s:%d", g.ServerAddr, g.ServerPort)

	if g.UseTLS {
		return g.connectTLS(addr)
	}

	conn, err := net.Dial("udp", addr)
	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}
	g.conn = conn
	log.Printf("Connected to network server at %s", addr)
	return nil
}

func (g *Gateway) connectTLS(addr string) error {
	// Load client certificate and key
	cert, err := tls.LoadX509KeyPair(g.CertFile, g.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load client certificate: %v", err)
	}

	// Load CA certificate
	caCert, err := os.ReadFile(g.CAFile)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to parse CA certificate")
	}

	// Configure TLS
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		ServerName:   g.ServerAddr, // For SNI
		MinVersion:   tls.VersionTLS12,
	}

	// For UDP over DTLS, we need to use a regular UDP connection
	// and wrap it with TLS for AWS IoT Core
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to resolve address: %v", err)
	}

	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("failed to dial UDP: %v", err)
	}

	// Wrap UDP connection with TLS (DTLS)
	// Note: For production, you may need a proper DTLS library
	// This is a simplified version for demonstration
	tlsConn := tls.Client(udpConn, tlsConfig)

	// Perform TLS handshake with timeout
	if err := tlsConn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		udpConn.Close()
		return fmt.Errorf("failed to set deadline: %v", err)
	}

	if err := tlsConn.Handshake(); err != nil {
		tlsConn.Close()
		return fmt.Errorf("TLS handshake failed: %v", err)
	}

	// Clear deadline after handshake
	if err := tlsConn.SetDeadline(time.Time{}); err != nil {
		tlsConn.Close()
		return fmt.Errorf("failed to clear deadline: %v", err)
	}

	g.conn = tlsConn
	log.Printf("Connected to network server at %s with TLS", addr)
	log.Printf("TLS version: %s", tls.VersionName(tlsConn.ConnectionState().Version))

	return nil
}
