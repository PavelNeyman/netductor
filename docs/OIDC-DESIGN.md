# OIDC / SSO design note (future multi-operator)

**Status:** design only (2026-10-12). Not implemented.

## Goal
Optional OpenID Connect hook so the Web operator UI can accept identity from an external IdP instead of (or in addition to) local session tokens. Aimed at multi-operator setups later.

## Proposed shape
- Config under `/etc/netductor/` or state: `oidc_issuer`, `oidc_client_id`, `oidc_client_secret` (or PKCE), `oidc_redirect_uri`.
- Login flow: Web redirects to IdP → callback exchanges code → maps claims (sub / email / groups) to an internal operator identity.
- Session: still issue a short-lived local `session` token (existing package) bound to the OIDC subject. Existing `requireSession` continues to work.
- Mapping: optional group claim → role (viewer / operator / admin). Default: any successful OIDC login gets operator.
- Fallback: local session issue (current CLI / TG / Web flow) remains available when OIDC is not configured.
- TG stays local-session only for now (no OIDC in the bot).

## Non-goals for now
- Full multi-tenant RBAC.
- Replacing the existing session store.
- Hardware or edge OIDC.

## Open questions
- Which claims to trust for authorization.
- Whether to support both session cookie and Bearer.
- Audit actor field: store OIDC sub instead of "session".

Implementation deferred until multi-operator need is concrete.
