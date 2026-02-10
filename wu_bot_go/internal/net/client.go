package net

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"wu_bot_go/internal/game"
)

// MetaInfo holds the server discovery response.
type MetaInfo struct {
	GameServers []struct {
		ID   string `json:"id"`
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"gameServers"`
	LoginServers []struct {
		GameServerID string `json:"gameServerId"`
		BaseURL      string `json:"baseUrl"`
	} `json:"loginServers"`
	LastClientVersion string `json:"lastClientVersion"`
}

// NetClient handles all networking: meta-info fetch, HTTP login, JAR bridge TCP.
type NetClient struct {
	Username string
	Password string
	ServerID string

	// Discovered from meta-info
	GameHost      string
	GamePort      int
	BaseURL       string
	ClientVersion [3]int

	// Auth
	Token string

	// JAR bridge
	jarPath    string
	jar        *JARProcess
	conn       net.Conn
	listener   net.Listener
	sendCh     chan game.OutboundPacket
	packetCh   chan game.InboundPacket
	logFunc    func(string)

	mu         sync.Mutex
	connected  bool
	httpClient *http.Client

	ClientMD5 string
}

// NewNetClient creates a new network client.
func NewNetClient(username, password, serverID, jarPath string, packetCh chan game.InboundPacket, logFunc func(string)) *NetClient {
	return &NetClient{
		Username:   username,
		Password:   password,
		ServerID:   serverID,
		jarPath:    jarPath,
		packetCh:   packetCh,
		sendCh:     make(chan game.OutboundPacket, 256),
		logFunc:    logFunc,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		ClientMD5:  "269980fe6e943c59e8ff10338f719870",
	}
}

// SendCh returns the send channel for sending outbound packets.
func (c *NetClient) SendCh() chan<- game.OutboundPacket {
	return c.sendCh
}

// FetchMetaInfo fetches game server info and login URLs.
func (c *NetClient) FetchMetaInfo() error {
	resp, err := c.httpClient.Get("https://eu.api.waruniverse.space/meta-info")
	if err != nil {
		return fmt.Errorf("fetch meta-info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("meta-info status: %d", resp.StatusCode)
	}

	var meta MetaInfo
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return fmt.Errorf("decode meta-info: %w", err)
	}

	// Find matching servers
	for _, gs := range meta.GameServers {
		if gs.ID == c.ServerID {
			c.GameHost = gs.Host
			c.GamePort = gs.Port
			break
		}
	}
	for _, ls := range meta.LoginServers {
		if ls.GameServerID == c.ServerID {
			c.BaseURL = ls.BaseURL
			break
		}
	}

	if c.GameHost == "" || c.BaseURL == "" {
		return fmt.Errorf("server %s not found in meta-info", c.ServerID)
	}

	// Parse client version
	if meta.LastClientVersion != "" {
		parts := strings.Split(meta.LastClientVersion, ".")
		if len(parts) == 3 {
			fmt.Sscanf(parts[0], "%d", &c.ClientVersion[0])
			fmt.Sscanf(parts[1], "%d", &c.ClientVersion[1])
			fmt.Sscanf(parts[2], "%d", &c.ClientVersion[2])
		}
	}
	if c.ClientVersion[0] == 0 {
		c.ClientVersion = [3]int{1, 233, 0}
	}

	c.log(fmt.Sprintf("Meta-info: server=%s host=%s:%d version=%d.%d.%d",
		c.ServerID, c.GameHost, c.GamePort,
		c.ClientVersion[0], c.ClientVersion[1], c.ClientVersion[2]))

	return nil
}

// Login authenticates via HTTP and returns the combined token.
func (c *NetClient) Login() error {
	url := fmt.Sprintf("%s/auth-api/v3/login/%s/token?password=%s", c.BaseURL, c.Username, c.Password)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		TokenID interface{} `json:"tokenId"`
		Token   interface{} `json:"token"`
	}
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		return fmt.Errorf("decode login response: %w", err)
	}

	c.Token = fmt.Sprintf("%v:%v", result.TokenID, result.Token)
	c.log(fmt.Sprintf("Token: %s", c.Token))
	c.log("Login successful")
	return nil
}

// Start begins the full connection sequence: listen for JAR, spawn JAR, handle I/O.
func (c *NetClient) Start(ctx context.Context) error {
	// Listen on random port for JAR to connect
	var err error
	c.listener, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	port := c.listener.Addr().(*net.TCPAddr).Port
	c.log(fmt.Sprintf("Listening on port %d for JAR", port))

	// Accept JAR connection in goroutine
	acceptCh := make(chan net.Conn, 1)
	go func() {
		conn, err := c.listener.Accept()
		if err != nil {
			c.log(fmt.Sprintf("Accept error: %v", err))
			return
		}
		acceptCh <- conn
	}()

	// Spawn JAR process
	c.jar = NewJARProcess(c.jarPath, "127.0.0.1", port, c.logFunc)
	go func() {
		if err := c.jar.Start(ctx); err != nil && ctx.Err() == nil {
			c.log(fmt.Sprintf("JAR process exited: %v", err))
		}
	}()

	// Wait for JAR to connect or context cancel
	select {
	case conn := <-acceptCh:
		c.conn = conn
		c.connected = true
		c.log("JAR connected")
	case <-ctx.Done():
		c.listener.Close()
		return ctx.Err()
	case <-time.After(30 * time.Second):
		c.listener.Close()
		return fmt.Errorf("JAR connection timeout")
	}

	// Send startClient to JAR
	c.Send(game.BuildStartClientPacket(c.GameHost, c.GamePort))

	// Start read/write loops
	go c.readLoop(ctx)
	go c.writeLoop(ctx)

	return nil
}

// Send queues a packet for sending to the JAR.
func (c *NetClient) Send(pkt game.OutboundPacket) {
	select {
	case c.sendCh <- pkt:
	default:
		c.log("WARN: send channel full, dropping packet")
	}
}

// SendAuth sends the authentication packet through the JAR to the game server.
func (c *NetClient) SendAuth() {
	uid := GenerateUID(c.Username)
	c.log(fmt.Sprintf("Authenticating as %s (uid=%s)", c.Username, uid))

	authData := map[string]interface{}{
		"token": c.Token,
		"clientInfo": map[string]interface{}{
			"uid":             uid,
			"build":           0,
			"version":         []int{c.ClientVersion[0], c.ClientVersion[1], c.ClientVersion[2]},
			"platform":        "Desktop",
			"systemLocale":    "en_US",
			"preferredLocale": "en",
			"clientHash":      c.ClientMD5,
		},
	}

	c.Send(game.BuildApiRequestPacket(1, "auth/token-login", authData))
}

// Close shuts down the network client.
func (c *NetClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connected = false
	if c.conn != nil {
		c.conn.Close()
	}
	if c.listener != nil {
		c.listener.Close()
	}
	if c.jar != nil {
		c.jar.Kill()
	}
}

func (c *NetClient) readLoop(ctx context.Context) {
	scanner := bufio.NewScanner(c.conn)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}

		line := scanner.Text()
		pkt, err := DecodePacket(line)
		if err != nil {
			c.log(fmt.Sprintf("Decode error: %v", err))
			continue
		}
		if pkt == nil {
			continue
		}

		select {
		case c.packetCh <- *pkt:
		case <-ctx.Done():
			return
		}
	}

	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		c.log(fmt.Sprintf("Read error: %v", err))
	}
}

func (c *NetClient) writeLoop(ctx context.Context) {
	for {
		select {
		case pkt := <-c.sendCh:
			data, err := EncodePacket(pkt)
			if err != nil {
				c.log(fmt.Sprintf("Encode error: %v", err))
				continue
			}

			c.mu.Lock()
			if c.conn != nil {
				_, err = c.conn.Write(data)
			}
			c.mu.Unlock()

			if err != nil {
				c.log(fmt.Sprintf("Write error: %v", err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *NetClient) log(msg string) {
	if c.logFunc != nil {
		c.logFunc(msg)
	}
}

// GenerateUID creates a UUID from a username using SHA-256, matching uidForger.js.
func GenerateUID(username string) string {
	hash := sha256.Sum256([]byte(username))
	hex := fmt.Sprintf("%x", hash)
	// Format: first 32 hex chars as UUID: 8-4-4-4-12
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex[0:8], hex[8:12], hex[12:16], hex[16:20], hex[20:32])
}
