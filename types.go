package main

import (
	"encoding/json"
	"net"
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
