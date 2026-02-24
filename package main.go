package main

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

// Gateway configuration
type Gateway struct {
	GatewayEUI string
	ServerAddr string
	ServerPort int
	UseTLS     bool
	CertFile   string // Path to client certificate
	KeyFile    string // Path to client private key
	CAFile     string // Path to CA certificate
	conn       net.Conn
}

// Semtech UDP Protocol structures
type PushData struct {
	ProtocolVersion uint8
	RandomToken     uint16
	Identifier      uint8
	GatewayEUI      []byte
	Payload         json.RawMessage
}

type PullData struct {
	ProtocolVersion uint8
	RandomToken     uint16
	Identifier      uint8
	GatewayEUI      []byte
}

// RXPK represents a received packet
type RXPK struct {
	Time string  `json:"time"`
	Tmst uint32  `json:"tmst"`
	Freq float64 `json:"freq"`
	Chan uint8   `json:"chan"`
	RFch uint8   `json:"rfch"`
	Stat int8    `json:"stat"`
	Modu string  `json:"modu"`
	Datr string  `json:"datr"`
	Codr string  `json:"codr"`
	Rssi int16   `json:"rssi"`
	Lsnr float64 `json:"lsnr"`
	Size uint16  `json:"size"`
	Data string  `json:"data"`
}

// Stat represents gateway statistics
type Stat struct {
	Time string  `json:"time"`
	Lati float64 `json:"lati"`
	Long float64 `json:"long"`
	Alti int16   `json:"alti"`
	Rxnb uint32  `json:"rxnb"`
	Rxok uint32  `json:"rxok"`
	Rxfw uint32  `json:"rxfw"`
	Ackr float64 `json:"ackr"`
	Dwnb uint32  `json:"dwnb"`
	Txnb uint32  `json:"txnb"`
}

// Message payload structures
type RXPKPayload struct {
	RXPK []RXPK `json:"rxpk"`
}

type StatPayload struct {
	Stat Stat `json:"stat"`
}

func main() {
	// AWS IoT Core for LoRaWAN Configuration
	// Replace with your actual AWS LNS endpoint
	// Format: <account-specific-prefix>.lns.iot.<region>.amazonaws.com
	gateway := &Gateway{
		GatewayEUI: "AA555A0000000101", // Replace with your gateway EUI
		ServerAddr: "A1FVQWQLWY45AS.lns.lorawan.us-east-1.amazonaws.com",
		ServerPort: 8887, // AWS LNS TLS port (use 1700 for non-TLS)
		UseTLS:     true, // Enable TLS for production
		CertFile:   "./certs/gateway-cert.pem",    // Client certificate
		KeyFile:    "./certs/gateway-key.pem",     // Client private key
		CAFile:     "./certs/AmazonRootCA1.pem",   // AWS Root CaA
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

func (g *Gateway) SendPullData() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		token := g.generateRandomToken()
		gatewayEUI := g.parseGatewayEUI()

		packet := []byte{
			0x02,                // Protocol version
			byte(token >> 8),    // Random token MSB
			byte(token & 0xFF),  // Random token LSB
			0x02,                // PULL_DATA identifier
		}
		packet = append(packet, gatewayEUI...)

		if _, err := g.conn.Write(packet); err != nil {
			log.Printf("Error sending PULL_DATA: %v", err)
			continue
		}
		log.Printf("Sent PULL_DATA (token: %04x)", token)
	}
}

func (g *Gateway) SendStats() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	rxnb := uint32(0)
	txnb := uint32(0)

	for range ticker.C {
		token := g.generateRandomToken()
		gatewayEUI := g.parseGatewayEUI()

		stat := StatPayload{
			Stat: Stat{
				Time: time.Now().UTC().Format("2006-01-02 15:04:05 MST"),
				Lati: 37.7749,
				Long: -122.4194,
				Alti: 15,
				Rxnb: rxnb,
				Rxok: rxnb,
				Rxfw: rxnb,
				Ackr: 100.0,
				Dwnb: 0,
				Txnb: txnb,
			},
		}

		payload, err := json.Marshal(stat)
		if err != nil {
			log.Printf("Error marshaling stat: %v", err)
			continue
		}

		packet := []byte{
			0x02,                // Protocol version
			byte(token >> 8),    // Random token MSB
			byte(token & 0xFF),  // Random token LSB
			0x00,                // PUSH_DATA identifier
		}
		packet = append(packet, gatewayEUI...)
		packet = append(packet, payload...)

		if _, err := g.conn.Write(packet); err != nil {
			log.Printf("Error sending stats: %v", err)
			continue
		}
		log.Printf("Sent gateway statistics (token: %04x)", token)
	}
}

func (g *Gateway) SimulateUplinks() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		g.SendUplinkMessage()
	}
}

func (g *Gateway) SendUplinkMessage() {
	token := g.generateRandomToken()
	gatewayEUI := g.parseGatewayEUI()

	// Generate random LoRaWAN payload (simulated)
	data := make([]byte, 20)
	rand.Read(data)
	encodedData := base64.StdEncoding.EncodeToString(data)

	rxpk := RXPKPayload{
		RXPK: []RXPK{
			{
				Time: time.Now().UTC().Format(time.RFC3339),
				Tmst: uint32(time.Now().Unix()),
				Freq: 868.1,
				Chan: 0,
				RFch: 0,
				Stat: 1,
				Modu: "LORA",
				Datr: "SF7BW125",
				Codr: "4/5",
				Rssi: -57,
				Lsnr: 8.5,
				Size: uint16(len(data)),
				Data: encodedData,
			},
		},
	}

	payload, err := json.Marshal(rxpk)
	if err != nil {
		log.Printf("Error marshaling RXPK: %v", err)
		return
	}

	packet := []byte{
		0x02,                // Protocol version
		byte(token >> 8),    // Random token MSB
		byte(token & 0xFF),  // Random token LSB
		0x00,                // PUSH_DATA identifier
	}
	packet = append(packet, gatewayEUI...)
	packet = append(packet, payload...)

	if _, err := g.conn.Write(packet); err != nil {
		log.Printf("Error sending uplink: %v", err)
		return
	}
	log.Printf("Sent uplink message (token: %04x, RSSI: -57 dBm, SNR: 8.5 dB)", token)
}

func (g *Gateway) ReceiveMessages() {
	buffer := make([]byte, 4096)

	for {
		n, err := g.conn.Read(buffer)
		if err != nil {
			log.Printf("Error reading from server: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if n < 4 {
			continue
		}

		version := buffer[0]
		token := uint16(buffer[1])<<8 | uint16(buffer[2])
		identifier := buffer[3]

		switch identifier {
		case 0x01: // PUSH_ACK
			log.Printf("Received PUSH_ACK (token: %04x)", token)
		case 0x04: // PULL_ACK
			log.Printf("Received PULL_ACK (token: %04x)", token)
		case 0x03: // PULL_RESP
			log.Printf("Received PULL_RESP (token: %04x) - Downlink message", token)
			if n > 4 {
				log.Printf("Downlink payload: %s", string(buffer[4:n]))
			}
		default:
			log.Printf("Received unknown message (version: %d, token: %04x, id: %d)", version, token, identifier)
		}
	}
}

func (g *Gateway) generateRandomToken() uint16 {
	b := make([]byte, 2)
	rand.Read(b)
	return uint16(b[0])<<8 | uint16(b[1])
}

func (g *Gateway) parseGatewayEUI() []byte {
	eui := make([]byte, 8)
	fmt.Sscanf(g.GatewayEUI, "%02x%02x%02x%02x%02x%02x%02x%02x",
		&eui[0], &eui[1], &eui[2], &eui[3], &eui[4], &eui[5], &eui[6], &eui[7])
	return eui
}
