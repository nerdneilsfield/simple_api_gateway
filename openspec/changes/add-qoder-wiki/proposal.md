# Change: Add Qoder RepoWiki at /wiki

## Why
We want built-in documentation served directly from the gateway so users can browse the Qoder RepoWiki at `/wiki` without running a separate docs site.

## What Changes
- Integrate `go-embed-qorder-wiki` and mount the handler at `/wiki`.
- Serve RepoWiki content from `.qoder/repowiki/zh` with a safe root resolver.
- Configure Mermaid/KaTeX assets via CDN and rewrite `file://` links to GitHub using the build git commit.
- Ensure release packaging (Docker image and GoReleaser archives) includes `.qoder/repowiki/zh` assets.

## Impact
- Affected specs: gateway-wiki (new capability).
- Affected code: router initialization, new wiki handler integration, build packaging, docs.
