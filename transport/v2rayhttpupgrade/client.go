package v2rayhttpupgrade

import (
	std_bufio "bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/onering"
	"github.com/sagernet/sing-box/common/tls"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

var _ adapter.V2RayClientTransport = (*Client)(nil)

type Client struct {
	dialer        N.Dialer
	serverAddr    M.Socksaddr
	host          string
	path          string
	headers       http.Header
	oneringConfig *onering.Config
}

func NewClient(ctx context.Context, dialer N.Dialer, serverAddr M.Socksaddr, options option.V2RayHTTPUpgradeOptions, tlsConfig tls.Config) (adapter.V2RayClientTransport, error) {
	// Parse OneRing config from TLS ServerName if available
	var oneringCfg *onering.Config
	if tlsConfig != nil {
		serverName := tlsConfig.ServerName()
		if serverName != "" {
			cfg, err := onering.Parse(serverName)
			// REVIEWER FIX #3: Parse errors acknowledged but not logged
			// TODO: Add logging when logger becomes available in NewClient signature
			// Currently NewClient doesn't receive a logger parameter (see transport/v2ray/transport.go:64)
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
		dialer = tls.NewDialer(dialer, tlsConfig)
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

	// Determine host header
	host := options.Host
	if oneringCfg != nil && oneringCfg.Enabled {
		// Use real domain for Host header
		host = oneringCfg.GetHTTPHost()
	} else if host == "" {
		host = serverAddr.AddrString()
	}

	return &Client{
		dialer:        dialer,
		serverAddr:    actualServerAddr,
		host:          host,
		path:          options.Path,
		headers:       options.Headers.Build(),
		oneringConfig: oneringCfg,
	}, nil
}

func (c *Client) DialContext(ctx context.Context) (net.Conn, error) {
	conn, err := c.dialer.DialContext(ctx, N.NetworkTCP, c.serverAddr)
	if err != nil {
		return nil, err
	}

	request := &http.Request{
		Method: http.MethodGet,
		URL: &url.URL{
			Scheme: "http",
			Host:   c.host,
			Path:   c.path,
		},
		Header: c.headers.Clone(),
		Host:   c.host,
	}
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "websocket")

	err = request.Write(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	reader := std_bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, request)
	if err != nil {
		conn.Close()
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusSwitchingProtocols {
		conn.Close()
		return nil, E.New("unexpected status: ", response.Status)
	}

	// Handle buffered data
	if reader.Buffered() > 0 {
		buffer := buf.NewSize(reader.Buffered())
		_, err = buffer.ReadFullFrom(reader, buffer.FreeLen())
		if err != nil {
			conn.Close()
			return nil, err
		}
		conn = bufio.NewCachedConn(conn, buffer)
	}

	return conn, nil
}

func (c *Client) Close() error {
	return nil
}
