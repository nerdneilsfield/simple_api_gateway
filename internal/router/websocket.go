package router

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nerdneilsfield/simple_api_gateway/internal/config"
	"github.com/nerdneilsfield/simple_api_gateway/internal/loadbalancer"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

// WebSocket 以原始 TCP 隧道转发：握手阶段解析后端 HTTP 响应头，确认 101 后字节原样透传，不解析帧。
// WebSocket is proxied as a raw TCP tunnel: the backend's HTTP handshake response is parsed,
// and only after a 101 are bytes relayed untouched (no frame parsing). TCP<->TCP uses splice(2).

const (
	wsHandshakeTimeout = 10 * time.Second
	wsMaxRejectBody    = 1 << 20
	wsMaxHeadBytes     = 16 << 10 // 握手响应头上限 / cap on backend response head

	// wsReadBufferSize 必须 >= fasthttp 的 ReadBufferSize，保证首次 Read 能排空其内部缓冲。
	// Must be >= fasthttp's ReadBufferSize so the first Read drains its internal buffer.
	wsReadBufferSize = 4096
)

var (
	wsDialer = &net.Dialer{Timeout: wsHandshakeTimeout, KeepAlive: 30 * time.Second}
	// 32KiB：TLS 路径回退使用的拷贝缓冲 / copy buffer for the non-splice fallback
	wsCopyPool = sync.Pool{New: func() any { b := make([]byte, 32<<10); return &b }}
	// 4KiB：首次读取缓冲，空闲长连接只占用这一小块 / small first-read buffer
	wsFirstPool = sync.Pool{New: func() any { b := make([]byte, wsReadBufferSize); return &b }}

	// 逐跳头，不转发给后端 / hop-by-hop headers never forwarded
	wsHopHeaders = []string{"Proxy-Authorization", "Proxy-Connection", "Keep-Alive", "Te", "Trailer", "Transfer-Encoding"}
	// 自定义头不得改变协议边界 / custom headers must not alter protocol framing
	wsProtectedHeaders = map[string]struct{}{
		"Connection": {}, "Upgrade": {}, "Content-Length": {}, "Transfer-Encoding": {},
	}
)

// hasToken reports whether any comma-separated value of the header contains token (exact, case-insensitive).
func hasToken(h *fasthttp.RequestHeader, name, token string) bool {
	found := false
	h.VisitAll(func(k, v []byte) {
		if found || !strings.EqualFold(string(k), name) {
			return
		}
		for _, part := range strings.Split(string(v), ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				found = true
				return
			}
		}
	})
	return found
}

// isWebSocketUpgrade reports whether the request is a body-less GET WebSocket handshake.
// 判断是否为无请求体的 GET WebSocket 握手
func isWebSocketUpgrade(c *fiber.Ctx) bool {
	// 普通请求没有 Upgrade 头，这里一次 Peek 即返回
	if !c.Request().Header.IsGet() || !strings.EqualFold(c.Get(fiber.HeaderUpgrade), "websocket") {
		return false
	}
	return hasToken(&c.Request().Header, fiber.HeaderConnection, "upgrade")
}

// buildWSHeader copies the client handshake, strips hop-by-hop fields and points it at the backend.
// 复制客户端握手头，清理逐跳头，并指向后端
func buildWSHeader(c *fiber.Ctx, u *url.URL, route config.Route) *fasthttp.RequestHeader {
	h := &fasthttp.RequestHeader{}
	c.Request().Header.CopyTo(h)

	// 先收集 Connection 声明的逐跳字段再删除 / collect Connection-listed fields before removing
	var extra []string
	h.VisitAll(func(k, v []byte) {
		if strings.EqualFold(string(k), fiber.HeaderConnection) {
			for _, p := range strings.Split(string(v), ",") {
				if p = strings.TrimSpace(p); p != "" && !strings.EqualFold(p, "upgrade") {
					extra = append(extra, p)
				}
			}
		}
	})
	for _, k := range extra {
		h.Del(k)
	}
	for _, k := range wsHopHeaders {
		h.Del(k)
	}

	h.SetRequestURI(u.RequestURI())
	h.SetHost(u.Host)
	if route.UaClient != "" {
		h.Set(fiber.HeaderUserAgent, route.UaClient)
	}
	for k, v := range route.CustomHeaders {
		if _, bad := wsProtectedHeaders[http.CanonicalHeaderKey(k)]; !bad {
			h.Set(k, v)
		}
	}
	h.Set(fiber.HeaderConnection, "Upgrade")
	h.Set(fiber.HeaderUpgrade, "websocket")
	return h
}

// handleWebSocket dials the backend, performs the handshake, and hijacks the client only on a valid 101.
// 拨号后端并完成握手；仅在后端返回有效 101 时才接管客户端连接
func handleWebSocket(c *fiber.Ctx, lb loadbalancer.LoadBalancer, route config.Route) error {
	// 握手要求：无请求体 / require a body-less handshake
	// 用原始请求体判断：c.Body() 会按 Content-Encoding 解压，可被压缩炸弹利用
	// Check the raw body: c.Body() would decompress (zip-bomb) per Content-Encoding
	if len(c.Request().Body()) > 0 || c.Request().Header.ContentLength() > 0 {
		return c.Status(fiber.StatusBadRequest).SendString("WebSocket handshake must not have a body")
	}

	backendURL := lb.NextBackend()
	if backendURL == "" {
		return c.Status(fiber.StatusServiceUnavailable).SendString("No backend servers available")
	}

	target, err := buildTargetURL(c, backendURL, route)
	if err != nil {
		logger.Error("Error parsing backend URL", zap.String("backend", backendURL), zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).SendString("Error parsing backend URL")
	}
	u, err := url.Parse(target)
	if err != nil {
		logger.Error("Error parsing target URL", zap.String("target", target), zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).SendString("Error parsing backend URL")
	}

	useTLS := u.Scheme == "https" || u.Scheme == "wss"
	addr := u.Host
	if u.Port() == "" {
		port := "80"
		if useTLS {
			port = "443"
		}
		addr = net.JoinHostPort(u.Hostname(), port)
	}

	start := time.Now()
	var conn net.Conn
	if useTLS {
		d := tls.Dialer{NetDialer: wsDialer, Config: &tls.Config{ServerName: u.Hostname()}}
		conn, err = d.Dial("tcp", addr)
	} else {
		conn, err = wsDialer.Dial("tcp", addr)
	}
	if err != nil {
		lb.ReportFailure(backendURL)
		logger.Error("WebSocket backend dial failed", zap.String("backend", backendURL), zap.Error(err))
		return c.Status(fiber.StatusBadGateway).SendString("Bad Gateway")
	}

	fail := func(err error) error {
		conn.Close()
		lb.ReportFailure(backendURL)
		logger.Error("WebSocket handshake failed", zap.String("backend", backendURL), zap.Error(err))
		return c.Status(fiber.StatusBadGateway).SendString("Bad Gateway")
	}

	// 握手读写有限时 / bounded handshake I/O
	conn.SetDeadline(time.Now().Add(wsHandshakeTimeout))
	if _, err := buildWSHeader(c, u, route).WriteTo(conn); err != nil {
		return fail(err)
	}
	// 响应头总量限 wsMaxHeadBytes，超限读到 EOF 即失败；解析完再放宽给拒绝响应的 body
	// Response head is capped;超限 surfaces as EOF. Widened afterwards for a rejection body.
	lr := &io.LimitedReader{R: conn, N: wsMaxHeadBytes}
	br := bufio.NewReaderSize(lr, wsReadBufferSize)
	resp, err := readHandshakeResponse(br)
	if err != nil {
		return fail(err)
	}
	lr.N = wsMaxRejectBody
	lb.ReportSuccess(backendURL, time.Since(start))

	if resp.StatusCode != http.StatusSwitchingProtocols || !strings.EqualFold(resp.Header.Get("Upgrade"), "websocket") {
		// 拒绝握手：按普通 HTTP 响应回传并关闭后端，客户端无法借此直连后端
		// Rejected: relay as a plain HTTP response and close the backend; no tunnel is opened
		defer conn.Close()
		status := resp.StatusCode
		if status == http.StatusSwitchingProtocols {
			status = fiber.StatusBadGateway
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, wsMaxRejectBody))
		for k, vs := range resp.Header {
			switch http.CanonicalHeaderKey(k) {
			case "Connection", "Keep-Alive", "Transfer-Encoding", "Content-Length", "Upgrade":
				continue
			}
			for _, v := range vs {
				c.Response().Header.Add(k, v)
			}
		}
		return c.Status(status).Send(body)
	}
	conn.SetDeadline(time.Time{})

	// 101 响应头 + 后端已随之发来的早期字节，先写给客户端
	// The 101 head plus any early backend bytes already buffered go to the client first
	var prelude bytes.Buffer
	prelude.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	resp.Header.Write(&prelude)
	prelude.WriteString("\r\n")
	if n := br.Buffered(); n > 0 {
		early, _ := br.Peek(n)
		prelude.Write(early)
	}

	// fasthttp 在回调前若 flush 失败会跳过回调；守护定时器保证此时后端连接仍会被关闭
	// If fasthttp's pre-callback flush fails the callback never runs; the guard still closes the backend.
	guard := time.AfterFunc(wsHandshakeTimeout, func() { conn.Close() })
	ctx := c.Context()
	ctx.HijackSetNoResponse(true)
	ctx.Hijack(func(client net.Conn) {
		if !guard.Stop() { // 守护已触发，后端已关
			client.Close()
			return
		}
		tunnel(client, conn, prelude.Bytes())
	})
	return nil
}

// readHandshakeResponse parses the backend response head; the handshake deadline bounds the time spent.
// 解析后端响应头，耗时由握手 deadline 限制
func readHandshakeResponse(br *bufio.Reader) (*http.Response, error) {
	// ReadResponse 对 101 不读 body；对其它状态码 Body 绑定 br，由调用方限量读取
	return http.ReadResponse(br, nil)
}

// tunnel relays bytes both ways until either side closes.
// 双向转发字节，任一方关闭即结束
func tunnel(client, backend net.Conn, prelude []byte) {
	raw := client
	if u, ok := client.(interface{ UnsafeConn() net.Conn }); ok {
		raw = u.UnsafeConn()
	}
	// hijackConn.Close 在默认配置下是空操作，必须关底层连接
	// hijackConn.Close is a no-op by default, so close the underlying conn
	closeBoth := func() {
		raw.Close()
		backend.Close()
	}

	raw.SetWriteDeadline(time.Now().Add(wsHandshakeTimeout))
	if _, err := raw.Write(prelude); err != nil {
		closeBoth()
		return
	}
	raw.SetWriteDeadline(time.Time{})

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer closeBoth()
		relay(raw, backend)
	}()

	// 首次读走 hijackConn，取走 fasthttp 已缓冲的字节（缓冲 >= ReadBufferSize，一次排空）；
	// 之后缓冲必为空，可直接用底层连接（可 splice）。
	// First read goes through hijackConn to drain fasthttp's buffer in one call; afterwards the
	// raw (spliceable) conn is used.
	bp := wsFirstPool.Get().(*[]byte)
	n, err := client.Read(*bp)
	if n > 0 {
		if _, werr := backend.Write((*bp)[:n]); werr != nil {
			err = werr
		}
	}
	wsFirstPool.Put(bp)
	if err == nil {
		relay(backend, raw)
	}
	closeBoth()
	<-done
}

type (
	onlyWriter struct{ io.Writer }
	onlyReader struct{ io.Reader }
)

// relay copies src to dst. TCP->TCP goes through the kernel (splice); otherwise a pooled buffer is used.
// 复制 src 到 dst。TCP→TCP 走内核 splice，否则使用池化缓冲区
func relay(dst, src net.Conn) {
	if _, ok := dst.(*net.TCPConn); ok {
		if _, ok := src.(*net.TCPConn); ok {
			io.Copy(dst, src)
			return
		}
	}
	bp := wsCopyPool.Get().(*[]byte)
	io.CopyBuffer(onlyWriter{dst}, onlyReader{src}, *bp)
	wsCopyPool.Put(bp)
}
