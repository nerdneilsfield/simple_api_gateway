package config

import (
	"embed"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	loggerPkg "github.com/nerdneilsfield/shlogin/pkg/logger"
	"github.com/nerdneilsfield/shlogin/pkg/network"
	"go.uber.org/zap"
)

var logger = loggerPkg.GetLogger()

//go:embed example_config.toml
var exampleConfigToml embed.FS

type Config struct {
	Port        int     `toml:"port"`
	Host        string  `toml:"host"`
	LogFilePath string  `toml:"log_file_path"`
	Routes      []Route `toml:"route"`
}

type Route struct {
	Path     string `toml:"path"`
	Backend  string `toml:"backend"`
	UaClient string `toml:"ua_client"`
}

func ParseConfig(path string) (*Config, error) {
	logger.Debug("parsing config", zap.String("path", path))
	var config Config
	_, err := toml.DecodeFile(path, &config)
	if err != nil {
		logger.Error("failed to parse config", zap.Error(err))
		return nil, err
	}

	if config.LogFilePath != "" {
		logger.SetLogFilePath(config.LogFilePath)
		logger.SetSaveToFile(true)
		logger.Reset()
	}

	return &config, nil
}

func ValidateConfig(config *Config) error {

	if config.Port < 0 && config.Port > 65535 {
		logger.Error("Port is not valid", zap.Int("port", config.Port))
		return fmt.Errorf("port is not valid")
	}

	if config.Host == "" {
		logger.Error("Host is not valid", zap.String("host", config.Host))
		return fmt.Errorf("host is not valid")
	}

	if len(config.Routes) == 0 {
		logger.Error("no routes found in config")
		return fmt.Errorf("no routes found in config")
	}

	existingPaths := make(map[string]bool)

	for _, route := range config.Routes {
		if route.Path == "" {
			logger.Error("route path is empty", zap.String("path", route.Path))
			return fmt.Errorf("route path is empty")
		}
		if existingPaths[route.Path] {
			logger.Error("route path is duplicated", zap.String("path", route.Path))
			return fmt.Errorf("route path is duplicated")
		}
		existingPaths[route.Path] = true
		if route.Backend == "" {
			logger.Error("route backend is empty", zap.String("path", route.Path))
			return fmt.Errorf("route backend is empty")
		}

		if _, err := url.ParseRequestURI(route.Backend); err != nil {
			logger.Error("route backend is not a valid URL", zap.String("path", route.Path), zap.String("backend", route.Backend))
			return fmt.Errorf("route backend is not a valid URL")
		}

		if _, err := network.HttpConnect(route.Backend); err != nil {
			logger.Error("failed to connect to route backend", zap.String("path", route.Path), zap.String("backend", route.Backend))
			return fmt.Errorf("failed to connect to route backend")
		}
	}

	return nil
}

// GetExampleConfig returns the example config as a string
func GetExampleConfig() (string, error) {
	exampleConfig, err := exampleConfigToml.ReadFile("example_config.toml")
	if err != nil {
		return "", err
	}
	return string(exampleConfig), nil
}

func GenerateExampleConfigPath(examplePath string) error {
	exampleConfig, err := GetExampleConfig()
	if err != nil {
		logger.Error("Failed to get example config")
		return err
	}

	if examplePath == "" {
		examplePath = "./example.toml"
	}

	ext := filepath.Ext(examplePath)
	if ext != ".toml" {
		logger.Error("example config file must have .toml extension", zap.String("path", examplePath))
		return fmt.Errorf("example config file must have .toml extension")
	}

	dir := filepath.Dir(examplePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		logger.Error("example config directory does not exist", zap.String("path", dir), zap.Error(err))
		return fmt.Errorf("example config directory does not exist")
	}

	return os.WriteFile(examplePath, []byte(exampleConfig), 0644)
}
