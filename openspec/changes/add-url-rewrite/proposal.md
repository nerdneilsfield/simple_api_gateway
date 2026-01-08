# Change: Add per-route URL rewrite

## Why
Some upstream APIs use different versioned paths than clients expect (e.g., clients call /v1 while upstream exposes /v4). The gateway should provide a simple per-route rewrite to map versioned paths without requiring client changes.

## What Changes
- Add optional per-route rewrite fields to the configuration.
- Apply the rewrite to the path portion that follows the route prefix before proxying.
- Preserve query strings when rewriting paths.

## Impact
- Affected specs: request-rewrite
- Affected code: internal/config, internal/router, example config and docs
