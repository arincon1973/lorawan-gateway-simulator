# LoRaWAN Gateway Simulator

A Go application that simulates a LoRaWAN gateway using the **Semtech Packet Forwarder** protocol. It connects to **AWS IoT Core for LoRaWAN** (or any compatible LNS) to send uplinks, receive downlinks, and report gateway statistics—useful for integration testing and development without physical hardware.

## Features

- **Semtech UDP protocol** — PUSH_DATA, PULL_DATA, PUSH_ACK, PULL_ACK, PULL_RESP
- **TLS support** — Connects to AWS LNS over TLS (port 8887); optional plain UDP (e.g. port 1700)
- **Simulated uplinks** — Periodically sends synthetic LoRaWAN frames with random payloads
- **Gateway stats** — Sends periodic gateway status (location, RX/TX counters)
- **Downlink handling** — Listens for PULL_RESP and logs downlink payloads

## Prerequisites

- **Go 1.21+**
- **Certificates** (for TLS): gateway client cert, key, and AWS Root CA in `./certs/`

## Project structure

```
.
├── main.go        # Entry point and gateway configuration
├── types.go       # Gateway, RXPK, Stat, and protocol types
├── connection.go  # TCP/UDP and TLS connection logic
├── protocol.go    # Semtech packet send/receive and helpers
├── go.mod
├── README.md
└── certs/         # Not in git — add your certificates here
    ├── gateway-cert.pem
    ├── gateway-key.pem
    └── AmazonRootCA1.pem
```

## Setup

1. **Clone the repo**

   ```bash
   git clone https://github.com/YOUR_USERNAME/lw-gateway-simulator.git
   cd lw-gateway-simulator
   ```

2. **Add certificates** (required for TLS)

   Create a `certs/` directory and place:

   - `gateway-cert.pem` — Gateway client certificate (from AWS IoT / your LNS)
   - `gateway-key.pem` — Gateway private key
   - `AmazonRootCA1.pem` — [AWS Root CA](https://www.amazontrust.com/repository/AmazonRootCA1.pem)

   The `certs/` folder is listed in `.gitignore`; never commit these files.

3. **Configure the gateway** in `main.go`:

   - `GatewayEUI` — Your gateway’s 16‑hex‑character EUI
   - `ServerAddr` — LNS hostname (e.g. `xxxxxxxx.lns.iot.us-east-1.amazonaws.com`)
   - `ServerPort` — `8887` for TLS, `1700` for plain UDP
   - `UseTLS` — `true` for AWS LNS, `false` for local/testing UDP

## Build and run

```bash
go build -o gateway-simulator .
./gateway-simulator
```

Or run directly:

```bash
go run .
```

You should see logs for connection, PULL_DATA, PUSH_DATA (stats and uplinks), and any PUSH_ACK / PULL_ACK / PULL_RESP from the server.

## Configuration summary

| Setting       | Description |
|---------------|-------------|
| `GatewayEUI`  | 8-byte gateway identifier (16 hex chars) |
| `ServerAddr`  | LNS hostname (e.g. AWS account-specific LNS endpoint) |
| `ServerPort`  | `8887` (TLS) or `1700` (UDP) |
| `UseTLS`      | `true` for production AWS LNS |
| `CertFile`    | Path to client certificate |
| `KeyFile`     | Path to client private key |
| `CAFile`      | Path to CA certificate (e.g. Amazon Root CA) |

## Protocol notes

The simulator speaks the [Semtech Gateway Protocol](https://github.com/Lora-net/packet_forwarder/blob/master/PROTOCOL.TXT): 4-byte header (version, token, identifier) plus optional payload. It sends PULL_DATA every 5 seconds, gateway stats every 30 seconds, and simulated uplinks every 10 seconds.

## License

Use and modify as needed for your projects.
