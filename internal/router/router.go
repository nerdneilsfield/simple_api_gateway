package router

import (
	"fmt"
	"net/url"

	"github.com/gofiber/fiber/v2"
	loggerPkg "github.com/nerdneilsfield/shlogin/pkg/logger"
	"github.com/nerdneilsfield/simple_api_gateway/internal/config"
	"go.uber.org/zap"
)

var logger = loggerPkg.GetLogger()

func CreateNewHandler(route config.Route) fiber.Handler {
	targetURL, err := url.Parse(route.Backend)
	if err != nil {
		return func(c *fiber.Ctx) error {
			return c.Status(500).SendString("Error parsing backend URL")
		}
	}

	return func(c *fiber.Ctx) error {
		// 构建目标URL
		trimmedPath := c.Path()[len(route.Path):]
		queryString := string(c.Request().URI().QueryString())
		targetFullURL := targetURL.String() + trimmedPath
		if queryString != "" {
			targetFullURL += "?" + queryString
		}

		// 创建代理请求
		req := fiber.AcquireAgent()
		defer fiber.ReleaseAgent(req)

		// 设置方法和URL
		req.Request().SetRequestURI(targetFullURL)
		req.Request().Header.SetMethod(string(c.Method()))

		// 复制所有头部
		c.Request().Header.VisitAll(func(key, value []byte) {
			req.Request().Header.SetBytesKV(key, value)
		})

		if route.UaClient != "" {
			req.Request().Header.Set("User-Agent", route.UaClient)
		}

		// 添加请求体
		if len(c.Body()) > 0 {
			req.Request().SetBody(c.Body())
		}

		// 发送请求
		if err := req.Parse(); err != nil {
			return c.Status(500).SendString(fmt.Sprintf("Error: %v", err))
		}

		// 发送请求并获取响应
		statusCode, body, errs := req.Bytes()
		if len(errs) > 0 {
			return c.Status(500).SendString(fmt.Sprintf("Error: %v", errs))
		}

		// 设置响应
		c.Status(statusCode)

		// 复制响应头 (这部分可能需要根据实际情况调整)
		// 由于我们不再使用 resp.Header，这部分代码需要修改或移除

		return c.Send(body)
	}
}

func Run(config_ *config.Config) {
	app := fiber.New()

	for _, route := range config_.Routes {
		app.All(route.Path+"/*", CreateNewHandler(route))
	}

	addrString := config_.Host + ":" + fmt.Sprint(config_.Port)
	if err := app.Listen(addrString); err != nil {
		logger.Fatal("failed to run server", zap.Error(err))
	}
}
