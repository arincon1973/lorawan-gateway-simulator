package main

import (
	"log"
)

func main() {
	// AWS IoT Core for LoRaWAN Configuration
	// Replace with your actual AWS LNS endpoint
	// Format: <account-specific-prefix>.lns.iot.<region>.amazonaws.com
	gateway := &Gateway{
		GatewayEUI: "AA555A0000000101", // Replace with your gateway EUI
		ServerAddr: "your-account-id.lns.iot.us-east-1.amazonaws.com", // AWS LNS endpoint
		ServerPort: 8887, // AWS LNS TLS port (use 1700 for non-TLS)
		UseTLS:     true, // Enable TLS for production
		CertFile:   "./certs/gateway-cert.pem",  // Client certificate
		KeyFile:    "./certs/gateway-key.pem",   // Client private key
		CAFile:     "./certs/AmazonRootCA1.pem", // AWS Root CA
	}

	log.Printf("Starting LoRaWAN Gateway Simulator")
	log.Printf("Gateway EUI: %s", gateway.GatewayEUI)
	log.Printf("Connecting to AWS LNS: %s:%d (TLS: %v)", gateway.ServerAddr, gateway.ServerPort, gateway.UseTLS)

	if err := gateway.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer gateway.conn.Close()

	// Start routines
	go gateway.SendPullData()
	go gateway.SendStats()
	go gateway.ReceiveMessages()

	// Simulate uplink messages
	gateway.SimulateUplinks()
}
