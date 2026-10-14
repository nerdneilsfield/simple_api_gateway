package router

import (
	"bufio"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nerdneilsfield/simple_api_gateway/internal/config"
	"github.com/valyala/fasthttp"
)

var leak = make(chan string, 1)

// 后端：完成 101 握手后原样回显字节（不解析帧，网关本身也不解析）
func startEchoBackend(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	backendHost := ln.Addr().String()
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				br := bufio.NewReader(c)
				req, err := http.ReadRequest(br)
				if err == nil && req.URL.Path == "/reject" {
					c.Write([]byte("HTTP/1.1 401 Unauthorized\r\nContent-Length: 2\r\n\r\nno"))
					// keep-alive：若网关开了隧道，下一条请求会到这里
					if r2, err := http.ReadRequest(br); err == nil {
						leak <- r2.URL.Path
					}
					return
				}
				if err != nil || req.URL.Path != "/ws" || req.Header.Get("X-Custom") != "v" ||
					req.Host != backendHost || req.Header.Get("Proxy-Authorization") != "" || req.Header.Get("X-Hop") != "" ||
					req.Header.Get("Connection") != "Upgrade" {
					c.Write([]byte("HTTP/1.1 400 Bad Request\r\nContent-Length: 0\r\n\r\n"))
					return
				}
				c.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))
				buf := make([]byte, 1024)
				for {
					n, err := br.Read(buf)
					if n > 0 {
						c.Write(buf[:n])
					}
					if err != nil {
						return
					}
				}
			}()
		}
	}()
	return "http://" + ln.Addr().String()
}

func TestWebSocketTunnel(t *testing.T) {
	backend := startEchoBackend(t)
	// LB 按 route.Path 全局缓存，路径需唯一以免复用旧后端
	prefix := "/chat" + strings.ReplaceAll(backend[len("http://127.0.0.1:"):], ":", "")
	route := config.Route{Path: prefix, Backends: []string{backend}, CustomHeaders: map[string]string{"X-Custom": "v"}}

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.All(route.Path+"/*", CreateNewHandler(route, false))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go app.Listener(ln)
	t.Cleanup(func() { app.Shutdown() })

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))

	// 握手与首批数据一并发送，验证 fasthttp 已缓冲的字节不丢
	c.Write([]byte("GET " + prefix + "/ws HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: keep-alive\r\nConnection: Upgrade, X-Hop\r\nX-Hop: s\r\nProxy-Authorization: x\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\nhello"))

	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, nil)
	if err != nil || resp.StatusCode != 101 {
		t.Fatalf("handshake: %v %v", resp, err)
	}
	got := make([]byte, 5)
	if _, err := readFull(br, got); err != nil || string(got) != "hello" {
		t.Fatalf("early echo: %q %v", got, err)
	}

	c.Write([]byte("world"))
	if _, err := readFull(br, got); err != nil || string(got) != "world" {
		t.Fatalf("echo: %q %v", got, err)
	}
}

func TestIsWebSocketUpgradeIgnoresPlainRequests(t *testing.T) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)
	if isWebSocketUpgrade(ctx) {
		t.Fatal("plain request must not be upgrade")
	}
}

func readFull(br *bufio.Reader, p []byte) (int, error) {
	n := 0
	for n < len(p) {
		m, err := br.Read(p[n:])
		n += m
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func TestWebSocketRejectedHandshakeOpensNoTunnel(t *testing.T) {
	backend := startEchoBackend(t)
	prefix := "/rej" + strings.ReplaceAll(backend[len("http://127.0.0.1:"):], ":", "")
	route := config.Route{Path: prefix, Backends: []string{backend}}
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.All(route.Path+"/*", CreateNewHandler(route, false))
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	go app.Listener(ln)
	t.Cleanup(func() { app.Shutdown() })

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	// 被拒绝的握手后紧跟第二条请求，不得到达后端
	c.Write([]byte("GET " + prefix + "/reject HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n" +
		"GET /smuggled HTTP/1.1\r\nHost: x\r\n\r\n"))
	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil || resp.StatusCode != 401 {
		t.Fatalf("want 401, got %v %v", resp, err)
	}
	select {
	case p := <-leak:
		t.Fatalf("request leaked to backend: %s", p)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestIsWebSocketUpgradeRejectsSubstring(t *testing.T) {
	app := fiber.New()
	rc := &fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod("GET")
	rc.Request.Header.Set("Upgrade", "websocket")
	rc.Request.Header.Set("Connection", "notupgrade")
	ctx := app.AcquireCtx(rc)
	defer app.ReleaseCtx(ctx)
	if isWebSocketUpgrade(ctx) {
		t.Fatal("Connection: notupgrade must not match")
	}
}

func TestWebSocketRejectsBodyAndHugeHead(t *testing.T) {
	// 带（压缩）请求体的升级请求 → 400，不拨号
	// 后端响应头超限 → 502
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				http.ReadRequest(bufio.NewReader(c))
				c.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nX-Big: " + strings.Repeat("a", 64<<10) + "\r\n\r\n"))
			}()
		}
	}()
	backend := "http://" + ln.Addr().String()
	prefix := "/big" + strings.ReplaceAll(backend[len("http://127.0.0.1:"):], ":", "")
	route := config.Route{Path: prefix, Backends: []string{backend}}
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.All(route.Path+"/*", CreateNewHandler(route, false))
	gl, _ := net.Listen("tcp", "127.0.0.1:0")
	go app.Listener(gl)
	t.Cleanup(func() { app.Shutdown() })

	do := func(extra string) int {
		c, err := net.Dial("tcp", gl.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		c.SetDeadline(time.Now().Add(5 * time.Second))
		c.Write([]byte("GET " + prefix + "/ws HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" + extra + "\r\n"))
		resp, err := http.ReadResponse(bufio.NewReader(c), nil)
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode
	}
	if got := do(""); got != 502 {
		t.Fatalf("huge head: want 502, got %d", got)
	}
	// gzip(empty) 声明了 Content-Length，原始体非空
	if got := do("Content-Encoding: gzip\r\nContent-Length: 20\r\n\r\n\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x03\x00\x00\x00\x00\x00\x00\x00\x00\x00"); got != 400 {
		t.Fatalf("body: want 400, got %d", got)
	}
}
