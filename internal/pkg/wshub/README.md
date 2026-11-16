# go-wshub

[![Go Reference](https://pkg.go.dev/badge/github.com/amarnathdbg101-coder/go/go-wshub.svg)](https://pkg.go.dev/github.com/amarnathdbg101-coder/go/go-wshub)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Production-ready, thread-safe WebSocket connection hub, pub-sub room manager, and keepalive heartbeat engine with zero third-party dependencies.

---

## Features
- **Concurrent Write Safe**: Wraps WebSocket connections in `SafeConn` with write mutex serialization and write deadlines, completely eliminating gorilla WebSocket concurrent write panics.
- **O(1) Pub-Sub Rooms**: Join, leave, and query rooms in O(1) time complexity.
- **Ghost Pruning**: Automatically purges and closes dead connections when client networks drop.
- **Heartbeat & Ping-Pong**: Automated background ping loop keeps reverse proxies (Cloudflare, NGINX) and mobile NATs active.
- **Standardized Real-Time Events**: Built-in JSON `Event` packaging with RFC 3339 timestamps.
- **Zero Third-Party Dependencies**: Pure Go standard library (`sync`, `time`, `context`, `encoding/json`).

---

## Installation

```bash
go get github.com/amarnathdbg101-coder/go/go-wshub@v1.0.0
```

---

## Quickstart

### 1. Initialize Hub

```go
hub := gowshub.New(
    gowshub.WithOnConnect(func(room string, conn *gowshub.SafeConn) {
        log.Printf("Client joined %s", room)
    }),
    gowshub.WithOnDisconnect(func(room string, conn *gowshub.SafeConn) {
        log.Printf("Client left %s", room)
    }),
)
defer hub.Close()
```

### 2. Upgrading HTTP Connection (with Gorilla WebSocket)

```go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

func ServeWS(w http.ResponseWriter, r *http.Request) {
    rawConn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }

    safeConn := gowshub.NewSafeConn(rawConn, gowshub.WithWriteDeadline(5 * time.Second))
    room := "shop:123"

    // Join room
    hub.Join(room, safeConn)
    defer hub.LeaveAll(safeConn)

    // Start background ping keepalive
    stopHeartbeat := gowshub.StartHeartbeat(r.Context(), safeConn, 25*time.Second, 5*time.Second)
    defer stopHeartbeat()

    // Read loop
    for {
        _, _, err := rawConn.ReadMessage()
        if err != nil {
            break
        }
    }
}
```

### 3. Multicasting to Room Subscribers

```go
// Broadcast structured event with automatic timestamp
hub.BroadcastEvent("shop:123", "NEW_ORDER", map[string]any{
    "order_id": "ord_999",
    "total": 450.00,
})

// Or broadcast raw JSON
hub.BroadcastJSON("shop:123", myPayload)
```

---

## License
MIT License.
