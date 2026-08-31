# Mutual TLS, Reference Tokens, and the SetSecret/GetSecret Demo — Overview

This document is a map of three related auth features added to fluffycore, for anyone (human or otherwise) picking this codebase up cold. Each has its own deeper documentation linked below; this page just orients you to what exists and where.

---

## 1. Reference-token middleware

[`middleware/auth/referencetoken`](../middleware/auth/referencetoken) (contract in [`contracts/middleware/auth/referencetoken`](../contracts/middleware/auth/referencetoken)) resolves an opaque external token — e.g. a Personal Access Token — into either:

- a JWT for the existing JWT pipeline to validate as normal, or
- a claims map to trust directly, no JWT involved.

The resolved claims are cached for a TTL so repeat requests with the same reference token skip re-resolution entirely. See that package's own README for the full design and how to plug in a resolver.

## 2. Mutual TLS (mTLS), gated per method

Optional, per-connection TLS with the verified client certificate surfaced as claims, so individual gRPC methods can require it via the same claims-AST gate already used for JWT-derived permissions.

| Piece | Location |
| --- | --- |
| Hot-reloading `*tls.Config` builder | [`runtime/servertls/servertls.go`](../runtime/servertls/servertls.go) |
| Claims-population middleware | [`middleware/auth/mtls/mtls.go`](../middleware/auth/mtls/mtls.go) |
| Self-signed CA/cert generation | [`utils/certgen`](../utils/certgen) |
| `gencert` CLI command | [`cobracore/cmd/gencert`](../cobracore/cmd/gencert), [`cmd/cli/root/gencert`](../cmd/cli/root/gencert) |
| Config fields (`tlsEnabled`, `tlsCertFile`, `tlsKeyFile`, `tlsClientCAFile`, `tlsClientAuth`, `tlsCertsDir`) | [`contracts/config/config.go`](../contracts/config/config.go) |
| Full write-up (architecture, config table, gating a method, testing locally, Vault rotation, caveats) | [`middleware/auth/mtls/README.md`](../middleware/auth/mtls/README.md) |

[`example/server/certs`](../example/server/certs) holds a checked-in, ~100-year-validity dev CA/server/client certificate bundle, so the example app runs with mTLS on by default with nothing to configure — see that directory's own `README.md` for why committing the (throwaway) private keys there is fine.

If you're about to port the mTLS piece into a different gRPC framework, read [`mtls-porting-notes.md`](mtls-porting-notes.md) first — it separates what's copy-portable from what's fluffycore-specific glue, and lists the concrete mistakes made (and fixed) while building this.

## 3. SetSecret / GetSecret demo RPCs

Two example RPCs on `helloworld.Greeter` ([`proto/helloworld/helloworld.proto`](../proto/helloworld/helloworld.proto)), backed by a trivial in-memory store ([`example/internal/services/secretstore`](../example/internal/services/secretstore)), exist solely to demonstrate mTLS end-to-end: both are gated on **`(JWT authenticated) OR (mtls_verified)`**, so they're callable either way. The gate itself lives in [`example/internal/auth/auth.go`](../example/internal/auth/auth.go). `middleware/auth/mtls/README.md`'s TL;DR section is the two-command proof using these RPCs.
