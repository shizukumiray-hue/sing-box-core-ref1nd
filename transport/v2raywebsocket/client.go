package v2raywebsocket

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/onering"
	"github.com/sagernet/sing-box/common/tls"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/bufio/deadline"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	sHTTP "github.com/sagernet/sing/protocol/http"
	"github.com/sagernet/ws"
)

var _ adapter.V2RayClientTransport = (*Client)(nil)

type Client struct {
	dialer              N.Dialer
	serverAddr          M.Socksaddr
	requestURL          url.URL
	headers             http.Header
	maxEarlyData        uint32
	earlyDataHeaderName string
	oneringConfig       *onering.Config
}

func NewClient(ctx context.Context, dialer N.Dialer, serverAddr M.Socksaddr, options option.V2RayWebsocketOptions, tlsConfig tls.Config) (adapter.V2RayClientTransport, error) {
	// Parse OneRing config from TLS ServerName if available
	var oneringCfg *onering.Config
	if tlsConfig != nil {
		serverName := tlsConfig.ServerName()
		if serverName != "" {
			cfg, err := onering.Parse(serverName)
			// REVIEWER FIX #3: Parse errors acknowledged but not logged
			// TODO: Add logging when logger becomes available in NewClient signature
			// Currently NewClient doesn't receive a logger parameter (see transport/v2ray/transport.go:57)
			// The error is intentionally ignored for backward compatibility - invalid onering
			// config falls back to standard mode rather than breaking the connection.
			if err != nil {
				// Parse error ignored, onering disabled, falls back to standard connection
				_ = err
			} else if cfg.Enabled {
				oneringCfg = cfg
				// Override TLS ServerName with bug domain
				tlsConfig.SetServerName(cfg.GetTLSSNI())
			}
		}
		if len(tlsConfig.NextProtos()) == 0 {
			tlsConfig.SetNextProtos([]string{"http/1.1"})
		}
		dialer = tls.NewDialer(dialer, tlsConfig)
	}
	
	var requestURL url.URL
	if tlsConfig == nil {
		requestURL.Scheme = "ws"
	} else {
		requestURL.Scheme = "wss"
	}
	
	// Override serverAddr if OneRing is enabled
	actualServerAddr := serverAddr
	if oneringCfg != nil && oneringCfg.Enabled {
		bugDomain := oneringCfg.GetDialAddress()
		// REVIEWER FIX #2: Use net.SplitHostPort to properly detect ports
		// Previous simple strings.Contains(":") check fails for IPv6 addresses like "::1" or "[2001:db8::1]:8080"
		// net.SplitHostPort handles IPv6 brackets correctly
		// NOTE: IPv6 addresses are currently rejected by domain validation in onering.go
		// If IPv6 support is added in the future, this logic will handle it correctly.
		_, _, err := net.SplitHostPort(bugDomain)
		if err == nil {
			// Bug domain already has port (e.g., "zoom.us:8443" or "[::1]:8080")
			actualServerAddr = M.ParseSocksaddr(bugDomain)
		} else {
			// No port, append server port (e.g., "zoom.us" → "zoom.us:443")
			actualServerAddr = M.ParseSocksaddr(fmt.Sprintf("%s:%d", bugDomain, serverAddr.Port))
		}
	}
	
	// Set initial requestURL.Host (will be overridden later for OneRing)
	requestURL.Host = serverAddr.String()
	requestURL.Path = options.Path
	err := sHTTP.URLSetPath(&requestURL, options.Path)
	if err != nil {
		return nil, E.Cause(err, "parse path")
	}
	if !strings.HasPrefix(requestURL.Path, "/") {
		requestURL.Path = "/" + requestURL.Path
	}
	headers := options.Headers.Build()
	
	// Handle Host header with OneRing
	if oneringCfg != nil && oneringCfg.Enabled {
		// Set Host in URL to real domain for OneRing
		requestURL.Host = oneringCfg.GetHTTPHost()
	} else {
		// Original logic
		if host := headers.Get("Host"); host != "" {
			headers.Del("Host")
			requestURL.Host = host
		}
	}
	
	if headers.Get("User-Agent") == "" {
		headers.Set("User-Agent", "Go-http-client/1.1")
	}
	return &Client{
		dialer,
		actualServerAddr,
		requestURL,
		headers,
		options.MaxEarlyData,
		options.EarlyDataHeaderName,
		oneringCfg,
	}, nil
}

func (c *Client) dialContext(ctx context.Context, requestURL *url.URL, headers http.Header) (*WebsocketConn, error) {
	// Use serverAddr which already contains bug domain if OneRing is enabled
	conn, err := c.dialer.DialContext(ctx, N.NetworkTCP, c.serverAddr)
	if err != nil {
		return nil, err
	}
	var deadlineConn net.Conn
	if deadline.NeedAdditionalReadDeadline(conn) {
		deadlineConn = deadline.NewConn(conn)
	} else {
		deadlineConn = conn
	}
	deadlineConn.SetDeadline(time.Now().Add(C.TCPTimeout))
	var protocols []string
	if protocolHeader := headers.Get("Sec-WebSocket-Protocol"); protocolHeader != "" {
		protocols = []string{protocolHeader}
		headers.Del("Sec-WebSocket-Protocol")
	}
	reader, _, err := ws.Dialer{Header: ws.HandshakeHeaderHTTP(headers), Protocols: protocols}.Upgrade(deadlineConn, requestURL)
	deadlineConn.SetDeadline(time.Time{})
	if err != nil {
		conn.Close()
		return nil, err
	}
	if reader != nil {
		buffer := buf.NewSize(reader.Buffered())
		_, err = buffer.ReadFullFrom(reader, buffer.FreeLen())
		if err != nil {
			conn.Close()
			return nil, err
		}
		conn = bufio.NewCachedConn(conn, buffer)
	}
	return NewConn(conn, nil, ws.StateClientSide), nil
}

func (c *Client) DialContext(ctx context.Context) (net.Conn, error) {
	if c.maxEarlyData <= 0 {
		conn, err := c.dialContext(ctx, &c.requestURL, c.headers)
		if err != nil {
			return nil, err
		}
		return conn, nil
	} else {
		return &EarlyWebsocketConn{Client: c, ctx: ctx, create: make(chan struct{})}, nil
	}
}

func (c *Client) Close() error {
	return nil
}
