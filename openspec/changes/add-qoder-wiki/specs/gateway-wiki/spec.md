## ADDED Requirements
### Requirement: Embedded Wiki Route
The gateway SHALL serve the RepoWiki content at the `/wiki` route.

#### Scenario: Serve wiki content
- **WHEN** a request targets `/wiki` or `/wiki/...`
- **THEN** the gateway responds with the wiki HTML/content rendered from `.qoder/repowiki/zh`.

### Requirement: Wiki Route Is Reserved
The gateway SHALL reject configuration routes that use the `/wiki` path.

#### Scenario: Config uses reserved wiki path
- **WHEN** a route is configured with path `/wiki` (or `/wiki/`)
- **THEN** configuration validation fails with an error indicating the path is reserved.

### Requirement: Wiki Asset Paths
The gateway SHALL serve RepoWiki assets required by rendered pages.

#### Scenario: Serve assets under mount prefix
- **WHEN** the browser requests `/wiki/_assets/...`
- **THEN** the gateway serves the RepoWiki assets.

#### Scenario: Redirect base wiki path
- **WHEN** a request targets `/wiki` without a trailing slash
- **THEN** the gateway redirects to `/wiki/` to ensure relative assets resolve.

### Requirement: Wiki Asset Loading via CDN
The wiki renderer SHALL use CDN-based Mermaid and KaTeX assets.

#### Scenario: Mermaid/KaTeX load from CDN
- **WHEN** the wiki page includes Mermaid or KaTeX
- **THEN** the HTML references CDN URLs for these assets.

### Requirement: GitHub Link Rewriting
The wiki renderer SHALL rewrite `file://` links to GitHub blob URLs using a pinned commit ref.

#### Scenario: Links use build commit ref
- **WHEN** the wiki contains a `file://` link
- **THEN** the rendered link points to `https://github.com/nerdneilsfield/simple_api_gateway/blob/<gitCommit>/...`.

### Requirement: Release Packaging Includes Wiki Assets
Docker and GoReleaser artifacts SHALL include `.qoder/repowiki/zh` so the wiki can load at runtime.

#### Scenario: Docker image contains wiki assets
- **WHEN** the gateway runs in a Docker image produced by CI
- **THEN** `.qoder/repowiki/zh` exists and `/wiki` serves content.
