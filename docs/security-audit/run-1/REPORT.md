# Security audit report — fluffycore (scoped, quick profile)

## 1. Run summary
- **Profile:** `quick`. **This is a partial pass, not a clean bill of health.**
- **Scope:** middleware/auth/{jwt,mtls,referencetoken}; nats/{nats_micro_service,client,nats_token}; services/common/cache/inmemory.go; runtime/servertls/servertls.go; services/jwtminter/KeyMaterialValidator.
- **Source ref:** d59f530 (tracked tree clean; untracked: .claude/, protoc-gen-go-fluffycore-di/internal/, scripts/profile-plugins.sh).
- **Budget:** 20 agent invocations. **Spent: 11** (4 reconnaissance, 6 hunters, 1 final critic). No verifier was needed because no candidate reached validation.
- **Execution:** source review only. Native Windows cannot enforce the required sandbox (no network, read-only target, scratch-only writes, resource limits), so no target code, build or test was run. Claims about library behaviour (for example jwx v4 rejecting `alg:none`) were not exercised.
- **Prior runs:** none. No carried confirmations or revalidations.
- **Deferred / out of scope:** 7 units deferred and 1 out of scope after the final critic (see section 7).

## 2. Posture summary
No confirmed vulnerabilities and no needs-validation leads in the reviewed scope. The core controls hold up on source review: the JWT signature covers the same bytes the claims are read from, the JWKS URL comes only from server config, reserved `mtls_*` claims are stripped on every path that accepts external claims, cache TTLs are absolute, and a NATS-bridged call re-enters the gRPC auth chain instead of bypassing it. The main residual risk is in things this pass did not reach (section 7), and in deployment choices the source cannot show (audience configuration, zero-trust mode, NATS ACLs).

## 3. Confirmed findings
None.

## 4. Needs validation
None. (No record was retained in `findings.json`; it is an empty array and passes `validate-findings.cjs`.)

## 5. Hardening notes (not findings)
Worth doing, ordered roughly by value. Each is source-grounded but showed no demonstrated boundary violation.
1. **Cap reference-token cache TTL at the JWT's `exp`** (referencetoken.go resolve, ~272-276): a JWT-kind reference token keeps authenticating from cache until the TTL ends even if the JWT expired earlier. Use `min(ttl, exp-now)`.
2. **Require `exp`** in the JWKS and KeyMaterial validators (`jwxt.WithRequiredClaim("exp")`): `jwxt.Validate` accepts a signed token with no `exp`, so one never expires.
3. **Audience is optional**: skipped when `Audiences` is empty or contains `"*"`; KeyMaterialValidator has no audience option at all. Consider fail-closed or a startup warning.
4. **NATS inbound:** allow-list forwarded headers instead of copying all of them into gRPC metadata; apply `validateSubjectToken` to inbound extracted params (a handler can currently receive `*` or `>`); make `Inject…` return an error when the target field is not string/number instead of silently skipping; give `HandleRequest` a bounded context instead of `context.Background()`.
5. **Error disclosure:** JWKS refresh errors reach unauthenticated callers verbatim (disclosing the JWKS URL); reference-token resolver errors and NATS service errors are echoed as `Unknown`/text. Return generic statuses and log detail.
6. **JWT validator wiring:** `ValidateAccessToken` takes the JWKS URL from the global `_issuerConfigs` but audiences from its own config; use `s.config.OAuth2Config.JWKSUrl`. Normalize issuer case on both sides. `_issuerConfigs` is an unlocked global map (safe only if all `AddValidators` calls finish before serving). The unary interceptor lacks the container nil-check the stream one has. `codes.NotFound` doubles as the "no token" sentinel; a custom validator returning it would be treated as anonymous. The claim-validation failure message says "token signature not valid".
7. **mTLS/servertls:** default `verify_if_given` plus an interceptor that never rejects leaves a method open to cert-less clients unless the claims map requires `mtls_verified`; reload failures only log (add a metric); no CRL/OCSP and established connections keep handshake-time chains; CN-only authorization is weak with a multi-purpose CA.
8. **KeyMaterialValidator:** the signing-key window uses signer-chosen `iat`, so it gives rotation hygiene, not protection against a compromised retired key; it returns `handled=false` on failure, silently passing to the next validator.
9. **NATS client / token:** delete case-variant `authorization` keys before setting the token-source header so the override is deterministic; `DecodeNATSConnectTokenClientCredentials` returns a nil pointer with nil error for `token:null`.
10. **Cache:** `services/common/cache/inmemory.go` has no entry-count bound; `loadResolvers` in referencetoken uses a package-level `sync.Once`.

Positive patterns: absolute-TTL cache with expiry cleanup before set; per-URL JWKS refresh gate stamped before the fetch; `mtls_*` stripping on JWT, reference-token and cached-claim paths; `VerifiedChains` required for mTLS claims and refusal to use the OS root store; subject-token validation on the NATS client; panic recovery with a generic reply.

## 6. Coverage summary (from the ledger)
| Status | Units |
|---|---|
| covered | 12 |
| deferred | 7 |
| out_of_scope | 1 |
| candidate / blocked | 0 |

Final critic result: ran once (quick profile); it proposed 6 new units and 2 reassignments, all recorded as deferred or out of scope. **No clean-coverage claim is made.**

## 7. What was NOT covered (deferred / out of scope)
Deferred (the critic gave source-backed reasons; none has been investigated):
1. **Claims merge into the principal** (jwt interceptor → `AddClaim` → the claims gate): anonymous/NotFound branches add `sub=anonymous` onto a principal earlier interceptors may have populated; the gate's trust in token-controlled claim types was never read.
2. **Reference-token ↔ JWT interceptor composition:** the capture hook also runs on the `WithAnonymousOnFailure`/NotFound branches, so a `sub=anonymous` principal might be cached for the TTL as the token's claims; direct-claims resolver output has no `sub`/issuer requirement; resolver DI wiring is unreviewed. (Two reference-token units were reopened for this reason.)
3. **NATS connect (Create side):** the client id/secret go in the NATS token base64-encoded with no TLS requirement in the connect options.
4. **servertls config validation:** `TLSClientAuth` set with TLS disabled/certs dir unset may yield a plaintext server with no error; `IsMutualTLSEnabled` may report true for `none`/`request`; reload does not check certificate expiry.
5. **KeyMaterialValidator wiring:** with `ValidateSignature=false` it returns `handled=true` for any parseable token and can accept before the JWKS validator runs; a missing options registration is a nil dereference.
6–7. The two reopened reference-token units (resolution and cache), deferred under the quick profile.

Out of scope: generated NATS handler wiring (`runtime/runtime.go`, the generator template). Hunter h5's "the bridge re-enters the gRPC auth chain" conclusion rests on the checked-in generated file only; the template that emits it was not read.

Deployment facts the source cannot show and that would matter if relied on: configured audiences and `"*"`; whether `AddNilValidator`, `WithAnonymousOnFailure` or `ValidateSignature=false` are used; zero-trust mode and the per-method claims map; interceptor order; NATS server ACLs/accounts; resolver implementations and cache backend; TLS termination and client CA policy.

## 8. Remediation after this run
- **servertls silent plaintext (deferred unit 4, verified by source read afterwards):** `BuildServerTLSConfig` now returns an error when `tlsClientAuth`, `tlsClientCAFile`, `tlsCertFile` or `tlsKeyFile` is set but TLS is not enabled, instead of returning `(nil, nil)` and starting a plaintext server. Regression test: `TestBuildServerTLSConfig_TLSSettingsWithoutEnable_Errors`. Note this makes previously-tolerated stray settings a startup error.
- **Reference-token cache outliving the JWT (hardening note 1):** a JWT-kind resolution is now cached for at most the JWT's remaining lifetime (`capTTLToJWTExpiry`), and not at all if it has already expired. A JWT that does not parse or has no `exp` keeps the configured TTL. Regression tests: `TestCapTTLToJWTExpiry`, `TestUnary_JWTKindCacheDoesNotOutliveTheJWT`.
- Hardening note 2 (require `exp`) and the other notes are not yet addressed.
