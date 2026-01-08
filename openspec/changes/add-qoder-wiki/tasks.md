## 1. Implementation
- [ ] 1.1 Add go-embed-qorder-wiki dependency and a wiki handler package for Fiber
- [ ] 1.2 Mount `/wiki` (and assets/redirect handling) before proxy routes and reserve `/wiki` in config validation
- [ ] 1.3 Configure RepoWiki assets (Mermaid/KaTeX CDN) and GitHub link rewrite with build git commit
- [ ] 1.4 Update README and example configs to mention `/wiki`
- [ ] 1.5 Package `.qoder/repowiki/zh` in Dockerfile and GoReleaser outputs
- [ ] 1.6 Add basic tests for wiki path handling if feasible
