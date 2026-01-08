## Context
RepoWiki content is generated under `.qoder/repowiki/zh`. We want the gateway to serve it at `/wiki` using the go-embed-qorder-wiki library with CDN assets, and include those files in release artifacts.

## Goals / Non-Goals
- Goals:
  - Serve `/wiki` from RepoWiki content with correct asset paths.
  - Use CDN for Mermaid and KaTeX assets.
  - Rewrite `file://` links to GitHub blob URLs using the build git hash.
  - Package `.qoder/repowiki/zh` in Docker and GoReleaser artifacts.
- Non-Goals:
  - Multi-language wiki switching.
  - Runtime wiki regeneration.

## Decisions
- Use `go-embed-qorder-wiki` with the Fiber adapter mounted at `/wiki`.
- Use `os.DirFS` with a resolved root (`.qoder/repowiki/zh`) for runtime access.
- Redirect `/wiki` to `/wiki/` to ensure relative asset paths resolve correctly.
- Use CDN defaults for Mermaid and KaTeX via the library config.
- Wire `GitSource` to `https://github.com/nerdneilsfield/simple_api_gateway` and the build `gitCommit`.

## Risks / Trade-offs
- The wiki depends on runtime files; missing `.qoder/repowiki/zh` will disable the wiki route.
- CDN outages may affect Mermaid/KaTeX rendering; using CDN reduces asset maintenance.

## Migration Plan
- Add wiki assets to Docker image and GoReleaser archives.
- Deploy with updated images that include `.qoder/repowiki/zh`.

## Open Questions
- Should `/wiki` be configurable or always enabled by default?
