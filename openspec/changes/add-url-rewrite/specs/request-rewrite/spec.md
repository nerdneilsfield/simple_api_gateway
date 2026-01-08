## ADDED Requirements
### Requirement: Route Rewrite Configuration
The system SHALL allow a route to define an optional path rewrite rule using
`rewrite_from` and `rewrite_to`. The rule is active only when both values are
non-empty.

#### Scenario: Rewrite rule configured
- **WHEN** a route defines rewrite_from "/v1" and rewrite_to "/v4"
- **THEN** the route is eligible for path rewrite

#### Scenario: Rewrite rule disabled
- **WHEN** rewrite_from or rewrite_to is empty
- **THEN** no path rewrite SHALL be applied for that route

### Requirement: Rewrite Application
The system SHALL apply the rewrite rule to the request path portion that follows
the route path prefix before proxying, and SHALL preserve the query string.

#### Scenario: Prefix rewrite applied
- **WHEN** the incoming path is "/xx/v1/models" and the route path is "/xx"
- **AND** rewrite_from is "/v1" and rewrite_to is "/v4"
- **THEN** the upstream request path SHALL be "/v4/models"

#### Scenario: Prefix rewrite not matched
- **WHEN** the request path portion does not start with rewrite_from
- **THEN** the upstream request path SHALL remain unchanged
