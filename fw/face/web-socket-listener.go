//go:build !tinygo

package face

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/named-data/ndnd/fw/core"
	defn "github.com/named-data/ndnd/fw/defn"
)

// WebSocketListenerConfig contains WebSocketListener configuration.
type WebSocketListenerConfig struct {
	Bind       string
	Port       uint16
	TLSEnabled bool
	TLSCert    string
	TLSKey     string
}

// WebSocketListener listens for incoming WebSockets connections.
type WebSocketListener struct {
	server   http.Server
	upgrader websocket.Upgrader
	localURI *defn.URI
}

// Constructs a WebSocket URL (ws or wss) using the configuration's bind address, port, and TLS setting.
func (cfg WebSocketListenerConfig) URL() *url.URL {
	addr := net.JoinHostPort(cfg.Bind, strconv.FormatUint(uint64(cfg.Port), 10))
	u := &url.URL{
		Scheme: "ws",
		Host:   addr,
	}
	if cfg.TLSEnabled {
		u.Scheme = "wss"
	}
	return u
}

// Returns a string representation of the WebSocket listener configuration, including its URL and TLS certificate path.
func (cfg WebSocketListenerConfig) String() string {
	return fmt.Sprintf("web-socket-listener (url=%s tls=%s)", cfg.URL(), cfg.TLSCert)
}

// Constructs a WebSocket listener configured with the provided settings, including TLS support if enabled, and initializes the server with the appropriate URI, upgrader, and security parameters.
func NewWebSocketListener(cfg WebSocketListenerConfig) (*WebSocketListener, error) {
	localURI := cfg.URL()
	ret := &WebSocketListener{
		server: http.Server{Addr: localURI.Host},
		upgrader: websocket.Upgrader{
			WriteBufferPool: &sync.Pool{},
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		localURI: defn.MakeWebSocketServerFaceURI(localURI),
	}
	if cfg.TLSEnabled {
		cert, e := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
		if e != nil {
			return nil, fmt.Errorf("tls.LoadX509KeyPair(%s %s): %w", cfg.TLSCert, cfg.TLSKey, e)
		}
		ret.server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		localURI.Scheme = "wss"
	}
	return ret, nil
}

// Returns a string representation of the WebSocketListener, including its local URI.
func (l *WebSocketListener) String() string {
	return "WebSocketListener, " + l.localURI.String()
}

// Starts the WebSocket server using HTTP or HTTPS based on TLS configuration, logging a fatal error if startup fails.
func (l *WebSocketListener) Run() {
	l.server.Handler = http.HandlerFunc(l.handler)

	var err error
	if l.server.TLSConfig == nil {
		err = l.server.ListenAndServe()
	} else {
		err = l.server.ListenAndServeTLS("", "")
	}
	if !errors.Is(err, http.ErrServerClosed) {
		core.Log.Fatal(l, "Unable to start listener", "err", err)
	}
}

// Handles incoming WebSocket connections by upgrading the HTTP request, creating a reliable WebSocket transport for NDN communication, and initializing a link service with fragmentation disabled to manage the network face.
func (l *WebSocketListener) handler(w http.ResponseWriter, r *http.Request) {
	c, e := l.upgrader.Upgrade(w, r, nil)
	if e != nil {
		return
	}

	newTransport := NewWebSocketTransport(l.localURI, c)
	core.Log.Info(l, "Accepting new WebSocket face", "uri", newTransport.RemoteURI())

	options := MakeNDNLPLinkServiceOptions()
	options.IsFragmentationEnabled = false // reliable stream
	MakeNDNLPLinkService(newTransport, options).Run(nil)
}

// Closes the WebSocket listener by initiating a graceful shutdown of the underlying server.
func (l *WebSocketListener) Close() {
	core.Log.Info(l, "Stopping listener")
	l.server.Shutdown(context.TODO())
}
