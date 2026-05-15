# BroChat
A real-time chat application featuring End-to-End Encryption (E2EE) with Post-Quantum resistant algorithms.
## Features
- **Quantum-Resistant Security**: Uses **ML-KEM** for key exchange, making it safe against Harvest now, decrypt later.
- **Authenticated Encryption**: Messages are secured using **ChaCha20-Poly1305**, providing high-speed symmetric encryption, and built-in integrity checking to prevent message tampering.
- **True E2EE**: Encryption and decryption happen strictly on your machine. The server is "blind", it only passes encrypted blobs around without ever seeing your actual text. 
- **Snappy Real-time Messaging**: Built on WebSockets for instant delivery.
---
## Tech Stack
### Client
- **Language**: Go (Golang)
- **UI Framework**: Fyne
### Backend
- **Language**: C#
- **Framework**: ASP .NET Core
---
## Installation & Setup
### 1. Prerequisites
- **Go** (1.25+ recommended)
- **.NET SDK** (latest LTS)

### 2. Server Setup
```bash
cd backend/BroChatServer
dotnet restore
dotnet run
```
The server starts on port 5107.

### 3. Client Setup
```bash
cd client
go mod tidy
go build -o ./build/client ./cmd/client
```

### 4. Start the client
#### Usage
From the client directory:
```bash
./client <server_url> <your_username> <recipient_username>
```
#### Example
Alice wants to talk to Bob.

##### Local Testing:
Terminal 1 (Alice)
```bash
./build/client ws://localhost:5107 Alice Bob
```
Terminal 2 (Bob)
```bash
./build/client ws://localhost:5107 Bob Alice
```
##### Production/Deployed:
Use wss:// (WebSocket Secure) when connecting to a deployed server.  
**Alice's machine:**
```bash
./build/client wss://example.com Alice Bob
```
**Bob's machine:**
```bash
./build/client wss://example.com Bob Alice
```
## Deployment
- **Infrastructure**: AWS EC2
- **Connectivity**: Cloudflare Tunnel (because EC2 has no open inbound ports)
- **Status**: Restricted Access - **[Request Access via Email](mailto:bnkduy1@gmail.com?subject=BroChat%20Access%20Request)**
