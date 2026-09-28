package secretstore

// ISecretStore is a trivial in-memory secret store for the SetSecret/GetSecret
// example RPCs -- demonstrates a method callable via either a normal JWT or
// mutual TLS (see example/internal/auth's entrypoint config). Not for
// production use: no persistence, no encryption at rest.
//
// Secrets are partitioned by owner -- the authenticated caller, as established by the
// server (JWT subject or verified client certificate), never a request field -- and then
// by orgID and key. orgID alone came from the request, so it isolated nothing: any
// authenticated caller could read or overwrite any org's secrets by naming it.
type ISecretStore interface {
	Set(owner, orgID, key, value string)
	// Get returns the stored value and true, or "" and false if unset.
	Get(owner, orgID, key string) (string, bool)
}
