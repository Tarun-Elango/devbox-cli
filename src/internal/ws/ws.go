package ws

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"
)

const (
	wsGUID         = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	maxPayloadSize = 10 * 1024 * 1024 // 10 MB — guard against oversized frames
)

// Frame opcodes (RFC 6455 §11.8).
const (
	opContinuation byte = 0x0
	opText         byte = 0x1
	opBinary       byte = 0x2
	opClose        byte = 0x8
	opPing         byte = 0x9
	opPong         byte = 0xA
)

// Conn is a minimal WebSocket connection backed by a raw TCP/TLS socket.
type Conn struct {
	conn net.Conn
	r    *bufio.Reader
}

// Dial opens a WebSocket connection to rawURL, injecting token as a Bearer
// Authorization header during the HTTP/1.1 upgrade handshake.
// rawURL may use ws://, wss://, http://, or https:// schemes.
func Dial(rawURL, token string) (*Conn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse ws url: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "http":
		scheme = "ws"
	case "https":
		scheme = "wss"
	case "ws", "wss":
		// already fine
	default:
		return nil, fmt.Errorf("unsupported scheme: %q", u.Scheme)
	}
	useTLS := scheme == "wss"

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if useTLS {
			port = "443"
		} else {
			port = "80"
		}
	}

	netConn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial %s:%s: %w", host, port, err)
	}

	if useTLS {
		tlsConn := tls.Client(netConn, &tls.Config{ServerName: host})
		if err := tlsConn.Handshake(); err != nil {
			netConn.Close()
			return nil, fmt.Errorf("tls handshake: %w", err)
		}
		netConn = tlsConn
	}

	// Random 16-byte key, base64-encoded (RFC 6455 §4.1).
	keyBuf := make([]byte, 16)
	if _, err := rand.Read(keyBuf); err != nil {
		netConn.Close()
		return nil, fmt.Errorf("generate ws key: %w", err)
	}
	key := base64.StdEncoding.EncodeToString(keyBuf)

	reqPath := u.RequestURI()
	if reqPath == "" {
		reqPath = "/"
	}

	// Build and send HTTP upgrade request.
	var sb strings.Builder
	sb.WriteString("GET " + reqPath + " HTTP/1.1\r\n")
	sb.WriteString("Host: " + u.Host + "\r\n")
	sb.WriteString("Upgrade: websocket\r\n")
	sb.WriteString("Connection: Upgrade\r\n")
	sb.WriteString("Sec-WebSocket-Key: " + key + "\r\n")
	sb.WriteString("Sec-WebSocket-Version: 13\r\n")
	if token != "" {
		sb.WriteString("Authorization: Bearer " + token + "\r\n")
	}
	sb.WriteString("\r\n")

	if _, err := io.WriteString(netConn, sb.String()); err != nil {
		netConn.Close()
		return nil, fmt.Errorf("send upgrade request: %w", err)
	}

	reader := bufio.NewReader(netConn)

	// Validate 101 Switching Protocols status line.
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		netConn.Close()
		return nil, fmt.Errorf("read upgrade response: %w", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(statusLine), "HTTP/1.1 101") {
		netConn.Close()
		return nil, fmt.Errorf("unexpected upgrade response: %s", strings.TrimSpace(statusLine))
	}

	// Read response headers and validate Sec-WebSocket-Accept.
	expected := wsAccept(key)
	var accept string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			netConn.Close()
			return nil, fmt.Errorf("read upgrade headers: %w", err)
		}
		if strings.TrimSpace(line) == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "sec-websocket-accept:") {
			accept = strings.TrimSpace(line[len("sec-websocket-accept:"):])
		}
	}
	if accept != expected {
		netConn.Close()
		return nil, fmt.Errorf("Sec-WebSocket-Accept mismatch: got %q want %q", accept, expected)
	}

	return &Conn{conn: netConn, r: reader}, nil
}

// wsAccept computes the expected Sec-WebSocket-Accept value for a given key.
func wsAccept(key string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, key+wsGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// ReadMessage reads the next complete text or binary message. Control frames
// (ping/pong/close) are handled transparently. Fragmented messages are
// reassembled before returning.
func (c *Conn) ReadMessage() (string, error) {
	for {
		fin, opcode, payload, err := c.readFrame()
		if err != nil {
			return "", err
		}

		switch opcode {
		case opText, opBinary:
			data := payload
			for !fin {
				f, _, p, e := c.readFrame()
				if e != nil {
					return "", e
				}
				data = append(data, p...)
				fin = f
			}
			return string(data), nil

		case opPing:
			// RFC 6455 §5.5.3: respond with Pong carrying the same payload.
			if err := c.writeFrame(opPong, payload); err != nil {
				return "", fmt.Errorf("send pong: %w", err)
			}

		case opPong:
			// Unsolicited pong — discard.

		case opClose:
			_ = c.writeFrame(opClose, nil) // best-effort echo
			return "", io.EOF

		default:
			return "", fmt.Errorf("unexpected opcode: 0x%X", opcode)
		}
	}
}

// WriteText sends the given string as a single text WebSocket frame.
func (c *Conn) WriteText(text string) error {
	return c.writeFrame(opText, []byte(text))
}

// Close sends a WebSocket close frame and closes the underlying connection.
func (c *Conn) Close() error {
	_ = c.writeFrame(opClose, nil)
	return c.conn.Close()
}

// readFrame reads a single WebSocket frame from the server.
func (c *Conn) readFrame() (fin bool, opcode byte, payload []byte, err error) {
	b0, err := c.r.ReadByte()
	if err != nil {
		return false, 0, nil, fmt.Errorf("frame byte 0: %w", err)
	}
	fin = b0&0x80 != 0
	opcode = b0 & 0x0F

	b1, err := c.r.ReadByte()
	if err != nil {
		return false, 0, nil, fmt.Errorf("frame byte 1: %w", err)
	}
	masked := b1&0x80 != 0
	rawLen := uint64(b1 & 0x7F)

	switch rawLen {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(c.r, ext[:]); err != nil {
			return false, 0, nil, fmt.Errorf("read 16-bit length: %w", err)
		}
		rawLen = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(c.r, ext[:]); err != nil {
			return false, 0, nil, fmt.Errorf("read 64-bit length: %w", err)
		}
		rawLen = binary.BigEndian.Uint64(ext[:])
	}

	if rawLen > maxPayloadSize {
		return false, 0, nil, fmt.Errorf("frame payload too large: %d bytes", rawLen)
	}

	var maskKey [4]byte
	if masked {
		if _, err := io.ReadFull(c.r, maskKey[:]); err != nil {
			return false, 0, nil, fmt.Errorf("read mask key: %w", err)
		}
	}

	payload = make([]byte, rawLen)
	if _, err := io.ReadFull(c.r, payload); err != nil {
		return false, 0, nil, fmt.Errorf("read payload: %w", err)
	}
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}
	return fin, opcode, payload, nil
}

// writeFrame writes a single masked WebSocket frame to the server.
// RFC 6455 §5.3: all client-to-server frames MUST be masked.
func (c *Conn) writeFrame(opcode byte, payload []byte) error {
	var maskKey [4]byte
	if _, err := rand.Read(maskKey[:]); err != nil {
		return fmt.Errorf("generate mask: %w", err)
	}

	n := len(payload)
	hdr := make([]byte, 0, 14)
	hdr = append(hdr, 0x80|opcode) // FIN=1, RSV=0, opcode
	switch {
	case n <= 125:
		hdr = append(hdr, 0x80|byte(n))
	case n <= 65535:
		hdr = append(hdr, 0x80|126, byte(n>>8), byte(n))
	default:
		hdr = append(hdr, 0x80|127,
			byte(n>>56), byte(n>>48), byte(n>>40), byte(n>>32),
			byte(n>>24), byte(n>>16), byte(n>>8), byte(n),
		)
	}
	hdr = append(hdr, maskKey[:]...)

	masked := make([]byte, n)
	for i, b := range payload {
		masked[i] = b ^ maskKey[i%4]
	}

	if _, err := c.conn.Write(append(hdr, masked...)); err != nil {
		return fmt.Errorf("write frame: %w", err)
	}
	return nil
}

// PollUntilDone dials rawURL and reads messages until the server signals
// completion (JSON containing "status":"done") or failure (JSON containing
// "status":"error"), or until timeout elapses. onMessage is called for every
// received message and may be nil.
func PollUntilDone(rawURL, token string, timeout time.Duration, onMessage func(string)) error {
	conn, err := Dial(rawURL, token)
	if err != nil {
		return fmt.Errorf("ws dial: %w", err)
	}
	defer conn.Close()

	if timeout > 0 {
		_ = conn.conn.SetDeadline(time.Now().Add(timeout))
	}

	for {
		msg, err := conn.ReadMessage()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("ws read: %w", err)
		}

		if onMessage != nil {
			onMessage(msg)
		}

		lower := strings.ToLower(msg)
		if strings.Contains(lower, `"status":"done"`) || strings.Contains(lower, `"status": "done"`) {
			return nil
		}
		if strings.Contains(lower, `"status":"error"`) || strings.Contains(lower, `"status": "error"`) {
			return fmt.Errorf("remote error: %s", msg)
		}
	}
}
