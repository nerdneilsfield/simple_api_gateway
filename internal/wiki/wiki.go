package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	repowiki "github.com/nerdneilsfield/go-embed-qorder-wiki"
	fiberadapter "github.com/nerdneilsfield/go-embed-qorder-wiki/adapters/fiber"
	"go.uber.org/zap"
)

const (
	defaultWikiRoot   = ".qoder/repowiki/zh"
	defaultWikiMount  = "/wiki"
	defaultWikiHome   = "主页.md"
	defaultContentDir = "content"

	mermaidCDN = "https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"
	katexCSS   = "https://cdn.jsdelivr.net/npm/katex@0.16.9/dist/katex.min.css"
	katexJS    = "https://cdn.jsdelivr.net/npm/katex@0.16.9/dist/katex.min.js"
	katexAuto  = "https://cdn.jsdelivr.net/npm/katex@0.16.9/dist/contrib/auto-render.min.js"
)

const repoURL = "https://github.com/nerdneilsfield/simple_api_gateway"

// NewHandler builds a RepoWiki handler mounted at /wiki.
func NewHandler(root string, gitCommit string, logger *zap.Logger) (fiber.Handler, string, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	mount := defaultWikiMount
	if root == "" {
		root = defaultWikiRoot
	}

	resolved := resolveWikiRoot(root)
	if resolved == "" {
		return nil, mount, fmt.Errorf("wiki root not found: %s", root)
	}

	home := resolveWikiHome(resolved, defaultContentDir, logger)
	cfg := repowiki.Config{
		FS:         os.DirFS(resolved),
		Root:       ".",
		ContentDir: defaultContentDir,
		Home:       home,
		Git: repowiki.GitSource{
			RepoURL: repoURL,
			Ref:     gitCommit,
		},
		Assets: repowiki.AssetConfig{
			Mermaid: repowiki.MermaidConfig{
				UseCDN: true,
				CDNURL: mermaidCDN,
			},
			KaTeX: repowiki.KaTeXConfig{
				Enabled: true,
				UseCDN:  true,
				CDN: repowiki.KaTeXCDNConfig{
					CSS:          katexCSS,
					JS:           katexJS,
					AutoRenderJS: katexAuto,
				},
			},
		},
	}

	handler, err := repowiki.New(cfg)
	if err != nil {
		return nil, mount, fmt.Errorf("init wiki handler: %w", err)
	}

	return fiberadapter.Wrap(handler, mount), mount, nil
}

func resolveWikiHome(root string, contentDir string, logger *zap.Logger) string {
	candidates := []string{
		defaultWikiHome,
		"README.md",
		"README_zh.md",
		"README_ZH.md",
		"快速开始.md",
		"Quickstart.md",
	}

	for _, name := range candidates {
		if fileExists(filepath.Join(root, contentDir, name)) {
			if name != defaultWikiHome {
				logger.Info("wiki home fallback", zap.String("home", name))
			}
			return name
		}
	}

	entries, err := os.ReadDir(filepath.Join(root, contentDir))
	if err != nil {
		return defaultWikiHome
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".md") {
			logger.Info("wiki home fallback", zap.String("home", name))
			return name
		}
	}

	return defaultWikiHome
}

func resolveWikiRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}

	if filepath.IsAbs(root) && dirExists(root) {
		return root
	}
	if dirExists(root) {
		return root
	}

	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, root)
		if dirExists(candidate) {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
