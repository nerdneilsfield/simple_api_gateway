package router

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	loggerPkg "github.com/nerdneilsfield/shlogin/pkg/logger"
	"github.com/nerdneilsfield/simple_api_gateway/internal/config"
	"go.uber.org/zap"
)

var logger = loggerPkg.GetLogger()

func CreateNewHandler(route config.Route) gin.HandlerFunc {
	targetURL, err := url.Parse(route.Backend)
	logger.Debug("parse backend url", zap.String("backend", route.Backend), zap.String("target", targetURL.String()))
	if err != nil {
		logger.Error("failed to parse backend url", zap.String("backend", route.Backend), zap.Error(err))
		return func(c *gin.Context) {
			c.String(http.StatusInternalServerError, "Error parsing backend URL")
		}
	}

	return func(c *gin.Context) {
		// 设置允许跨域访问的响应头
		logger.Debug("handling request", zap.String("path", c.Request.URL.Path), zap.String("method", c.Request.Method))
		origin := c.Request.Header.Get("Access-Control-Allow-Origin")
		if origin == "" {
			origin = "*"
		}
		c.Header("Access-Control-Allow-Origin", origin)

		// 读取请求体
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error reading request body")
			return
		}
		defer c.Request.Body.Close()

		// 移除路由前缀
		trimmedPath := c.Request.URL.Path[len(route.Path):]

		// 创建一个新的请求，使用trimmedPath
		logger.Debug("Connect to backend url", zap.String("target", targetURL.String()+trimmedPath))
		// c.String(http.StatusOK, "Connect to backend url: "+targetURL.String()+trimmedPath)
		newRequest, err := http.NewRequest(c.Request.Method, targetURL.String()+trimmedPath, io.NopCloser(bytes.NewReader(body)))
		if err != nil {
			logger.Error("failed to create new request", zap.String("trimmedPath", trimmedPath), zap.Error(err))
			c.String(http.StatusInternalServerError, "Error creating new request")
			return
		}

		// 复制原始请求的头
		for key, value := range c.Request.Header {
			newRequest.Header[key] = value
		}

		// 使用 HTTP 客户端发送请求并获取响应
		client := &http.Client{}
		response, err := client.Do(newRequest)
		if err != nil {
			errormsg := fmt.Sprintf("Error sending request to Reverse API %v: %v", route.Backend, err)
			c.String(http.StatusInternalServerError, errormsg)
			return
		}
		defer response.Body.Close()

		for key, values := range response.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		c.Status(response.StatusCode)
		// 复制响应体到响应 writer
		io.Copy(c.Writer, response.Body)
	}
}

func NewRouter(config_ *config.Config, r *gin.Engine) *gin.Engine {
	for _, route := range config_.Routes {
		r.Any(route.Path+"/*any", CreateNewHandler(route))
	}
	return r
}

func Run(config_ *config.Config) {
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	r = NewRouter(config_, r)

	addrString := config_.Host + ":" + strconv.Itoa(config_.Port)
	if err := r.Run(addrString); err != nil {
		logger.Fatal("failed to run server", zap.String("addr", addrString), zap.Error(err))
	}
}
