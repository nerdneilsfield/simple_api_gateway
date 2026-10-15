package router

import (
	"bufio"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nerdneilsfield/simple_api_gateway/internal/config"
	"github.com/valyala/fasthttp"
)

func TestBuildTargetURL(t *testing.T) {
	cases := []struct {
		name, req, backend string
		route              config.Route
		want               string
	}{
		{"default strips", "/api/v1/users?id=1", "http://h:9000", config.Route{Path: "/api"}, "http://h:9000/v1/users?id=1"},
		{"preserve", "/ws?serverId=abc&role=client&v=2", "http://h:4000", config.Route{Path: "/ws", PreservePath: true}, "http://h:4000/ws?serverId=abc&role=client&v=2"},
		{"preserve subpath", "/ws/x/y?a=1", "http://h:4000/", config.Route{Path: "/ws", PreservePath: true}, "http://h:4000/ws/x/y?a=1"},
	}
	app := fiber.New()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fctx := &fasthttp.RequestCtx{}
			fctx.Request.SetRequestURI(tc.req)
			c := app.AcquireCtx(fctx)
			defer app.ReleaseCtx(c)
			got, err := buildTargetURL(c, tc.backend, tc.route)
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

// 同一路径规则对 WebSocket 生效：后端要求 /ws 与完整查询参数
func TestWebSocketPreservePath(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	got := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		req, err := http.ReadRequest(bufio.NewReader(c))
		if err != nil {
			return
		}
		got <- req.URL.RequestURI()
		c.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))
	}()

	route := config.Route{Path: "/ws", Backends: []string{"http://" + ln.Addr().String()}, PreservePath: true}
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.All(route.Path, CreateNewHandler(route, false))
	gl, _ := net.Listen("tcp", "127.0.0.1:0")
	go app.Listener(gl)
	t.Cleanup(func() { app.Shutdown() })

	c, err := net.Dial("tcp", gl.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	c.Write([]byte("GET /ws?serverId=abc&role=client&v=2 HTTP/1.1\r\nHost: x\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n"))
	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil || resp.StatusCode != 101 {
		t.Fatalf("handshake: %v %v", resp, err)
	}
	if p := <-got; p != "/ws?serverId=abc&role=client&v=2" {
		t.Fatalf("backend got %q", p)
	}
}
