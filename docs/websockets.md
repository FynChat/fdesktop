# WebSockets for FynChat: Hub, Client, Broadcast

The current frontend polls the REST API (`ListMessages`, `SendMessage`). That
works but adds latency and waste. With WebSockets the server pushes new
messages to every client in a channel the moment they arrive.

This doc covers the standard Go pattern: a **Hub** that owns connections,
**Clients** (one per connection), and **broadcast** to channel rooms.

## 1. Flow overview

```
Client A            Server (Hub)            Client B
   |  WS connect /ws?channel_id=...&token=... |
   |------------------->|--register client--->|
   |  WS connect --------------------------->|
   |                    |--register client--->|
POST /channels/{id}/messages
   |------------------->|
   |                    |-- broadcast to channel room -->|
   |                    |                                |  new message event
```

## 2. Hub — the connection registry

The Hub keeps every connected client, grouped by channel, and serializes all
connect/disconnect/broadcast through one goroutine so no mutex is needed on
the map.

```go
type Hub struct {
    rooms      map[string]map[*Client]bool // channelID -> set of clients
    register   chan *Client
    unregister chan *Client
    broadcast  chan Broadcast
}

type Broadcast struct {
    ChannelID string
    Payload   []byte
}

func NewHub() *Hub {
    return &Hub{
        rooms:      make(map[string]map[*Client]bool),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        broadcast:  make(chan Broadcast, 256),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case c := <-h.register:
            if h.rooms[c.ChannelID] == nil {
                h.rooms[c.ChannelID] = make(map[*Client]bool)
            }
            h.rooms[c.ChannelID][c] = true

        case c := <-h.unregister:
            if room, ok := h.rooms[c.ChannelID]; ok {
                if _, ok := room[c]; ok {
                    delete(room, c)
                    close(c.send)
                    if len(room) == 0 {
                        delete(h.rooms, c.ChannelID)
                    }
                }
            }

        case b := <-h.broadcast:
            for c := range h.rooms[b.ChannelID] {
                select {
                case c.send <- b.Payload:
                default: // slow consumer: drop the client
                    close(c.send)
                    delete(h.rooms[b.ChannelID], c)
                }
            }
        }
    }
}
```

Key points:

- One goroutine owns `rooms` — no data races.
- `broadcast` is buffered so a slow slow network never blocks the room write.
- A client that can't keep up (`default` branch) is dropped rather than
  freezing everyone else.

## 3. Client — one per WebSocket connection

Each client has a **read pump** and a **write pump** (gorilla/websocket
requires one writer at a time).

```go
const (
    writeWait  = 10 * time.Second
    pongWait   = 60 * time.Second
    pingPeriod = (pongWait * 9) / 10
)

type Client struct {
    hub       *Hub
    conn      *websocket.Conn
    send      chan []byte
    ChannelID string
    UserID    int
}

func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })
    for {
        _, msg, err := c.conn.ReadMessage()
        if err != nil {
            return // disconnect
        }
        // Optional: handle client->server events (typing indicators, etc.)
        _ = msg
    }
}

func (c *Client) writePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()
    for {
        select {
        case msg, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok { // hub closed the channel
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
                return
            }
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

## 4. Upgrader + auth

WebSocket starts as an HTTP request, so reuse your existing auth:

```go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true }, // tighten in prod
}

func serveWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
    token := r.URL.Query().Get("token") // or Sec-WebSocket-Protocol header
    userID, err := authenticate(token)   // same check as your REST middleware
    if err != nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    channelID := r.URL.Query().Get("channel_id")

    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return // upgrader already wrote the error
    }
    c := &Client{hub: hub, conn: conn, send: make(chan []byte, 256),
        ChannelID: channelID, UserID: userID}
    hub.register <- c
    go c.writePump()
    go c.readPump()
}
```

## 5. Broadcasting on new messages

After `POST /channels/{id}/messages` persists the message, push it to the
room instead of waiting for polling:

```go
func createMessageHandler(hub *Hub) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        msg := ... // validate + save to DB
        payload, err := json.Marshal(msg)
        if err == nil {
            hub.broadcast <- Broadcast{ChannelID: msg.ChannelID, Payload: payload}
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(msg)
    }
}
```

Clients receive the message as a `Message` JSON object — the same struct the
REST endpoint returns, so the Fyne side decodes it identically.

## 6. Fyne side

In `chat.go`, open one connection per open chat and append messages as they
arrive. All UI mutation goes through `fyne.Do`:

```go
conn, _, err := websocket.DefaultDialer.Dial(
    "ws://localhost:8080/ws?channel_id="+channelID+"&token="+client.Token, nil)
go func() {
    for {
        _, data, err := conn.ReadMessage()
        if err != nil { return }
        var m Message
        if json.Unmarshal(data, &m) == nil {
            fyne.Do(func() {
                messagesBox.Add(widget.NewLabel(m.Content))
                messagesBox.Refresh()
                scroll.ScrollToBottom()
            })
        }
    }
}()
```

Close the connection when leaving the chat view (Back button).

## 7. What more you can do / recommendations

1. **Keep REST as the source of truth.** Use WebSockets only for server→client
   push; keep sending messages over HTTP POST. Simpler, easier to debug, and
   Gin/net-http middleware (auth, logging) keeps working.
2. **Message envelope.** Wrap payloads: `{"type": "new_message", "data": {...}}`.
   Lets you add `typing`, `user_joined`, `read_receipt` later without
   breaking clients.
3. **Heartbeat.** The ping/pong + read/write deadlines above detect dead
   connections — don't skip them.
4. **Reconnect with backoff.** On the Fyne side, retry the dial with
   exponential backoff (e.g. 1s, 2s, 4s, cap 30s) and re-`ListMessages` after
   reconnecting to fill any gap.
5. **Rate-limit writes.** If you later accept messages over WS, cap the
   inbound rate per client.
6. **Don't broadcast across server instances.** With one Go process the
   in-memory Hub is fine. When you scale to N servers, swap the Hub's
   broadcast for Redis Pub/Sub (each server subscribes and pushes to its
   local clients) — the Client/pump code stays the same.
7. **TLS in production.** Use `wss://` and proper origin checks.
8. **Auth on the upgrade request.** Validate the token before upgrading, as
   shown above — a plain query param is fine for dev; for stricter setups use
   a short-lived ticket from a REST endpoint.
9. **Order & idempotency.** Messages can arrive out of order on reconnect;
   the `Message.ID` lets the client dedupe and the timestamp lets it sort.
10. **Server library choice.** `github.com/gorilla/websocket` is the de-facto
    standard. Gorilla is now in maintenance mode but stable and widely used;
    `nhooyr.io/websocket` (now `github.com/coder/websocket`) is a good
    actively-maintained alternative with a nicer API.
