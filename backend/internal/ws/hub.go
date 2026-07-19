// Package ws is the real-time push layer (§7 of the API contract): it fans
// out server-generated events — price ticks and published news — to every
// connected browser over WebSocket. It is deliberately a thin, transport-only
// in-process hub: producers (internal/ledger, internal/news) never import it,
// they hand it typed events through nil-safe observer hooks wired in main, so
// the trade/news logic stays decoupled from how (or whether) anything is
// listening.
package ws

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// sendBuffer bounds how many undelivered frames a single slow client may queue
// before the hub starts dropping its messages (never blocking the publisher or
// any other client). Ticks are self-describing snapshots, so a client that
// falls behind and misses a few simply catches up on the next one.
const sendBuffer = 64

// writeTimeout bounds a single frame write so one wedged connection can't hold
// its writer goroutine open indefinitely.
const writeTimeout = 10 * time.Second

// Hub tracks the set of live clients subscribed to each topic and broadcasts
// payloads to them. Safe for concurrent use: Publish is called from every
// trade (HTTP handlers and bot goroutines alike) and from the news job, while
// ServeWS adds/removes clients as connections open and close.
type Hub struct {
	// originPatterns is passed straight to websocket.Accept to authorize the
	// browser's Origin header (host[:port], wildcards allowed). Empty means
	// same-origin only.
	originPatterns []string

	mu     sync.RWMutex
	topics map[string]map[*client]struct{}
}

type client struct {
	send chan []byte
}

// NewHub creates an empty hub that will accept WebSocket upgrades whose Origin
// matches one of originPatterns (see websocket.AcceptOptions.OriginPatterns).
func NewHub(originPatterns []string) *Hub {
	return &Hub{
		originPatterns: originPatterns,
		topics:         make(map[string]map[*client]struct{}),
	}
}

// Publish delivers payload to every client currently subscribed to topic. It
// never blocks: a client whose buffer is full simply misses this message.
func (h *Hub) Publish(topic string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.topics[topic] {
		select {
		case c.send <- payload:
		default:
			// Slow client — drop rather than stall the whole broadcast.
		}
	}
}

func (h *Hub) add(topic string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	subs := h.topics[topic]
	if subs == nil {
		subs = make(map[*client]struct{})
		h.topics[topic] = subs
	}
	subs[c] = struct{}{}
}

func (h *Hub) remove(topic string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs := h.topics[topic]; subs != nil {
		delete(subs, c)
		if len(subs) == 0 {
			delete(h.topics, topic)
		}
	}
}

// ServeWS returns an http.HandlerFunc that upgrades the request to a WebSocket
// and streams every payload published to topic until the client disconnects.
// This is a push-only channel — the server reads and discards anything the
// client sends (via CloseRead), which is also how it notices the disconnect.
func (h *Hub) ServeWS(topic string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: h.originPatterns,
		})
		if err != nil {
			// Accept has already written an error response.
			return
		}
		defer conn.CloseNow()

		c := &client{send: make(chan []byte, sendBuffer)}
		h.add(topic, c)
		defer h.remove(topic, c)

		// CloseRead drains (and discards) incoming frames and returns a ctx
		// that is cancelled when the peer closes or the connection breaks.
		ctx := conn.CloseRead(r.Context())

		for {
			select {
			case <-ctx.Done():
				return
			case payload := <-c.send:
				writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
				err := conn.Write(writeCtx, websocket.MessageText, payload)
				cancel()
				if err != nil {
					return
				}
			}
		}
	}
}
