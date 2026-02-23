package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

func (g *Gateway) SendPullData() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		token := g.generateRandomToken()
		gatewayEUI := g.parseGatewayEUI()

		packet := []byte{
			0x02,               // Protocol version
			byte(token >> 8),   // Random token MSB
			byte(token & 0xFF), // Random token LSB
			0x02,               // PULL_DATA identifier
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
			0x02,               // Protocol version
			byte(token >> 8),   // Random token MSB
			byte(token & 0xFF), // Random token LSB
			0x00,               // PUSH_DATA identifier
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
		0x02,               // Protocol version
		byte(token >> 8),   // Random token MSB
		byte(token & 0xFF), // Random token LSB
		0x00,               // PUSH_DATA identifier
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
