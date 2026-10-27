# Architecture summary — fluffycore (scoped, quick profile)

Source ref d59f530. Scope: middleware/auth/{jwt,mtls,referencetoken}, nats/{nats_micro_service,client,nats_token}, services/common/cache/inmemory.go, runtime/servertls/servertls.go, services/jwtminter/KeyMaterialValidator. Everything else is out of scope. This is a partial pass.

## Product, principals, authority
fluffycore is a Go library/framework (module github.com/fluffy-bunny/fluffycore) for gRPC/Echo services with DI, auth middleware, and a NATS micro-service transport. Consumers are services built on it; operators are those service owners. Lower-trust principals: anonymous callers (no Authorization header becomes `sub=anonymous`, middleware/auth/jwt/middleware.go:366-377), JWT bearer holders, reference-token/PAT holders, mTLS clients, and any NATS client able to publish to a handler subject. Protected resources: the gRPC methods behind the claims gate (middleware/claimsprincipal/claims-principal-middleware.go:124-151, out of scope) and whatever the services do.

## Stack and operation
Go 1.27; gRPC; lestrrat-go/jwx v4 (+ jwkfetch/httprc for JWKS caching); nats.go micro; ttlcache v2; zerolog; fluffy-dozm-di. Deployment is a Docker image plus unauthenticated docker-compose NATS for tests. No comparable-software baseline is source-grounded. Target code cannot be executed here: this is native Windows and the required OS sandbox (no network, read-only target, scratch-only writes, resource limits) cannot be enforced, so all work is source review; anything that needs execution is `needs_validation`.

## Entry surfaces and important paths
1. gRPC `authorization` metadata -> JWT interceptors (unary and stream differ): getTokenFromAuthorizationHeader (:230-251), _validate (:255-291), getRawToken does an unverified parse (:458-482), validators run in registration order and the first `handled=true` wins; a `handled=false` error is swallowed and the next validator is tried. ValidateAccessToken (:180-228) checks issuer, JWKS signature (no visible alg allow-list), 5-minute skew, audience only if configured ("*" disables). Fallbacks: no header -> anonymous; `WithAnonymousOnFailure`; `AddNilValidator` accepts anything.
2. JWKS: per-issuer URL from config, httprc defaults, unknown-kid refresh gated once per 30s per URL, `_issuerConfigs` global map written without a lock.
3. KeyMaterialValidator: sibling IValidator with local keys; no audience check; `ValidateSignature=false` and `ValidateIssuer=false` options; iat must fall in key window.
4. Reference tokens (referencetoken.go): prefix or two-dot detection, resolvers in order; a resolver returns either a JWT (re-validated by the JWT interceptor) or a final claims map trusted directly with no JWT check. Cache key `reftok:` + sha256(raw token); TTL = resolver TTL or 5 min; cached entries skip resolution and JWT validation and can outlive the JWT `exp`. Unary caches after next returns, stream at handoff. `mtls_*` claims are stripped.
5. mTLS: mtls.go trusts VerifiedChains[0][0]; issues mtls_verified/cn/fingerprint/san_uri; never rejects. servertls.go: client-auth modes (default VerifyClientCertIfGiven with a CA), mtime-based hot reload every >=5s, failed reload keeps last good cert/CA, TLS 1.2 minimum, no CRL/OCSP.
6. NATS inbound: HandleRequest (nats_micro_service.go:324-398) copies ALL NATS headers (including Authorization) into gRPC outgoing metadata and calls the service method over a gRPC client connection; it does no authentication itself. Whether that call passes through the gRPC auth chain is generated code outside scope. Subject values are written into request-proto fields via JSON+jsonpath (:59-107); subject authority overrides the body; validateSubjectToken runs only on the client side (errors.go:128), so wildcard-containing subjects are not filtered inbound. No visible message-size limit. Error status/message are sent in headers (errors.go).
7. NATS client: copies outgoing gRPC metadata to NATS headers and overwrites Authorization from a token source; rewrites correlation/span headers; default timeout 5 s. nats_token: connect token is base64 JSON carrying client_id/client_secret/account in clear (encoded, not encrypted); decode only here, validation lives in the NATS auth callout (outside scope).
8. services/common/cache/inmemory.go: absolute-TTL memory cache, singleflight GetOrInsertE, dropExpired before Set; keys caller-controlled, no size bound, shared claim slices.

## Trust boundaries and strongest visible control
Token authenticity: JWKS signature verification plus issuer match. Claim trust: reserved `mtls_*` claim stripping on JWT/reference/cache paths. mTLS: VerifiedChains requirement plus refusal to use the OS root store. Revocation: bounded only by cache TTL; no CRL/OCSP/revocation list. NATS: only the NATS server's own ACLs (not visible) stand between a publisher and a handler.

## Deployment-dependent facts (not source-visible)
Which interceptors are chained and in what order; zero-trust on/off and the per-method claims map; use of WithAnonymousOnFailure/AddNilValidator/ValidateSignature=false; configured issuers, JWKS URLs and audiences; resolver implementations; cache backend (ICache may be shared); NATS server ACLs/accounts/auth callout; TLS-terminating proxies; secrets.

## Starting paths
middleware/auth/jwt/middleware.go, jwt_token.go; middleware/auth/referencetoken/referencetoken.go; middleware/auth/mtls/mtls.go; runtime/servertls/servertls.go; services/jwtminter/KeyMaterialValidator/KeyMaterialValidator.go; nats/nats_micro_service/{nats_micro_service.go,errors.go}; nats/client/client.go; nats/nats_token/{connect.go,NATSConnectToken.go}; services/common/cache/inmemory.go.

## Prior coverage
None: no earlier ledger exists for this repo. No carried confirmations or hunter exclusions.

## Companion selection
Selected: ATTACK-CLASSES.md ordinary blocks (Access control, Cryptography and secrets, Business logic, Resource and file handling, Injection); WEB-PROTOCOL-AND-AUTH.md (JWT verification and claim binding, mTLS peer and application-identity confusion, Certificate lifecycle fallback) because JWT, JWKS and mTLS identity boundaries exist; PROTOCOLS-RPC-AND-MESSAGING.md (Interceptor and method-path inconsistency, Peer identity to application-principal confusion, Topic/routing-key/subscription scope gaps, Dead-letter/retry/diagnostic disclosure) because NATS requests are bridged into gRPC. Excluded: SUPPLY-CHAIN-AND-RELEASE.md (no in-scope release/CI/plugin boundary), MEMORY-SAFETY-AND-BINARY.md, AI-AND-LLM.md, CLIENT-SIDE.md, DESKTOP-MOBILE-AND-LOCAL-IPC.md, CLOUD-AND-DEPLOYMENT.md, DATA-ISOLATION-AND-LIFECYCLE.md, RESOURCE-EXHAUSTION-AND-AVAILABILITY.md (no source-visible boundary in scope requiring them; quick profile).
