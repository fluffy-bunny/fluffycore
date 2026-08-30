package secretstore

// ISecretStore is a trivial in-memory secret store for the SetSecret/GetSecret
// example RPCs -- demonstrates a method callable via either a normal JWT or
// mutual TLS (see example/internal/auth's entrypoint config). Not for
// production use: no persistence, no encryption at rest.
type ISecretStore interface {
	Set(orgID, key, value string)
	// Get returns the stored value and true, or "" and false if unset.
	Get(orgID, key string) (string, bool)
}
