# `utils.OAuth2Secrets` — Client Secret Generation, Hashing, and Verification

This document covers the `OAuth2Secrets` helper in [`utils/oauth2_secrets.go`](../utils/oauth2_secrets.go) — how to generate OAuth2 client secrets, how their hashes are stored, and how to verify them on the way back in.

---

## What it solves

When a service issues an OAuth2 client secret it has to:

1. Produce a cryptographically random string suitable for `client_secret_basic` / `client_secret_post`.
2. Store **only a hash** server-side — never the plaintext.
3. Verify an incoming plaintext against the stored hash in **constant time**.
4. Allow the hashing algorithm to evolve over time (rotation, upgrades to argon2id, etc.) **without losing the ability to read existing rows**.

`OAuth2Secrets` does all four. Every hash it emits is prefixed with a PHC-style algorithm tag (`$sha256$…`, `$hmac-sha256$…`) — modeled after argon2's `$argon2id$v=19$…` format — so the stored string is self-describing.

---

## The stored hash format

```
$<algorithm>$<hex-digest>
```

| Algorithm | Prefix | Stored example |
|---|---|---|
| SHA-256 | `$sha256$` | `$sha256$9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08` |
| HMAC-SHA-256 | `$hmac-sha256$` | `$hmac-sha256$c1b2…` |

The prefix is the algorithm discriminator — `DetectHashAlgorithm` reads it and the matching `Verify*` method enforces it.

---

## API surface

```go
type OAuth2Secrets struct {
    HMACKey []byte // optional — required for HashHMAC / VerifyHMAC
}

func NewOAuth2Secrets() *OAuth2Secrets
func NewOAuth2SecretsWithHMACKey(key []byte) *OAuth2Secrets

func (s *OAuth2Secrets) Generate() (*ClientSecret, error)

func (s *OAuth2Secrets) HashSHA256(secret string) (string, error)
func (s *OAuth2Secrets) VerifySHA256(secret, hash string) bool

func (s *OAuth2Secrets) HashHMAC(secret string) (string, error)
func (s *OAuth2Secrets) VerifyHMAC(secret, hash string) bool

func (s *OAuth2Secrets) Verify(secret, hash string) bool             // dispatches by prefix
func (s *OAuth2Secrets) DetectAlgorithm(hash string) HashAlgorithm

// Package-level (stateless)
func DetectHashAlgorithm(hash string) HashAlgorithm
type HashAlgorithm int                                                // SHA256 | HMACSHA256 | Unknown

type ClientSecret struct {
    Secret string `json:"secret"` // plaintext — return to client ONCE
    Hash   string `json:"hash"`   // algorithm-tagged hash — store server-side
}
```

### Validation guarantees

Every `Hash*` call enforces:

- **Length** ≥ 32 characters
- **Shannon entropy** ≥ 3.5 bits/char

This rejects user-supplied secrets that are too short or low-entropy (`"aaaaaaaaa…"`, dictionary words). `Verify*` returns `false` on the same validation failures rather than panicking, so a malformed input never short-circuits to "true".

`VerifySHA256` rejects any hash that does **not** carry the `$sha256$` prefix, and `VerifyHMAC` rejects anything not prefixed with `$hmac-sha256$` — so a SHA-256 hash cannot be silently validated against an HMAC verifier or vice versa.

---

## Typical lifecycles

### Issue a new client secret

```go
s := utils.NewOAuth2SecretsWithHMACKey(serverHMACKey)
cs, err := s.Generate()           // 256 bits of entropy, base64url-encoded
if err != nil { return err }

// Show cs.Secret to the operator ONCE; then forget it.
// Persist cs.Hash on the client_credentials row.
hash, err := s.HashHMAC(cs.Secret) // upgrade the default $sha256$ tag to $hmac-sha256$
if err != nil { return err }
clientRow.SecretHash = hash
```

> `Generate()` always returns a `$sha256$` tagged hash so it stays usable without an HMAC key. When a key is configured, immediately re-hash with `HashHMAC` before persisting — that's the preferred storage format (see "Which algorithm" below).

### Verify on token request

```go
s := utils.NewOAuth2SecretsWithHMACKey(serverHMACKey)

if !s.Verify(presented, clientRow.SecretHash) {
    return errInvalidClient
}
```

`Verify` dispatches on the stored prefix, so the same call site keeps working as you migrate rows from `$sha256$` to `$hmac-sha256$` (or, later, to `$argon2id$`).

### Migrate / rehash on next successful auth

```go
if s.Verify(presented, clientRow.SecretHash) {
    if s.DetectAlgorithm(clientRow.SecretHash) != utils.HashAlgorithmHMACSHA256 {
        if newHash, err := s.HashHMAC(presented); err == nil {
            clientRow.SecretHash = newHash
            _ = repo.Save(ctx, clientRow)
        }
    }
}
```

This is the standard "rehash on use" upgrade path — no big-bang migration, no need to ever see the plaintext outside of a real authentication.

---

## Which algorithm should I use?

| Approach | When to pick it | Cost |
|---|---|---|
| **HMAC-SHA-256 + server-side pepper** (recommended default) | Server-generated, high-entropy secrets (which `Generate()` produces). | Must manage the HMAC key out-of-band (KMS, sealed env). DB-only compromise no longer yields an offline attack target. |
| Plain SHA-256 | Sanity / fallback during migration; fine only because the generated entropy is 256 bits. | Anyone with the DB can brute force lower-entropy or user-chosen secrets at billions/sec. |
| Argon2id (not yet implemented here) | If you ever accept user-chosen / low-entropy secrets, or want belt-and-suspenders. | 50–250 ms per verify (tuneable); matters under heavy auth QPS. Adding it = a new prefix (`$argon2id$…`) and a new method pair. |

**Pragmatic default for this library:** use `NewOAuth2SecretsWithHMACKey(serverKey)` everywhere, store the key in your secret manager, and treat `$sha256$` rows as a legacy tier that the rehash-on-use loop will retire on its own.

---

## Security properties

- **Constant-time compare** — both `Verify*` methods use `crypto/subtle.ConstantTimeCompare` so a timing attacker cannot read the hash byte-by-byte.
- **Entropy floor** — short or repetitive secrets are rejected at hash time, so you cannot accidentally store a low-entropy secret even if a caller passes one in.
- **Algorithm pinning at verify time** — the prefix check happens *before* the constant-time compare, so an attacker cannot trick `VerifyHMAC` into accepting a `$sha256$` digest.
- **No plaintext at rest** — `Generate()` returns the plaintext exactly once; the rest of the system only ever sees `cs.Hash`.

### Things this code does **not** do

- Does not derive the HMAC key from a password or rotate it for you — that's your KMS / secrets manager.
- Does not implement argon2id yet (see table above).
- Does not transmit or log the plaintext — but if you write `cs.Secret` to a log, that's on the caller. Treat it like a credential.

---

## Extending with a new algorithm

To add (for example) argon2id later, the existing PHC-tagged design absorbs it cleanly:

1. Add `PrefixArgon2id = "$argon2id$"`.
2. Add `HashAlgorithmArgon2id` to the `HashAlgorithm` enum and its `String()` case.
3. Extend `DetectHashAlgorithm` with the new prefix.
4. Add `HashArgon2id` / `VerifyArgon2id` methods on `OAuth2Secrets`.
5. Add a `case HashAlgorithmArgon2id:` branch in `Verify`.

Existing `$sha256$` and `$hmac-sha256$` rows keep validating through the same `Verify` entry point — only the per-row prefix changes when you rehash.

---

## Test coverage

See [`utils/oauth2_secrets_test.go`](../utils/oauth2_secrets_test.go) for the full suite. Key cases:

- Generated secrets are unique, length-checked, and entropy-checked.
- Round-trip hash/verify for both SHA-256 and HMAC variants.
- Untagged or wrong-prefix hashes are rejected (algorithm pinning).
- Wrong key, wrong secret, tampered hash, low-entropy input — all return `false`, never panic.
- `Verify` auto-dispatch validates both prefixes through one call.
- `DetectHashAlgorithm` returns `Unknown` for empty strings, unprefixed digests, and prefixes from algorithms not implemented here (`$argon2id$…`).
