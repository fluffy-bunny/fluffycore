package secretstore

import (
	"sync"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_secretstore "github.com/fluffy-bunny/fluffycore/example/internal/contracts/secretstore"
)

type service struct {
	mu   sync.RWMutex
	data map[string]string
}

func init() {
	var _ fluffycore_contracts_secretstore.ISecretStore = (*service)(nil)
}

// AddSingletonSecretStore registers a process-wide (singleton) ISecretStore.
// Singleton, not scoped: a value set by one request must be visible to a
// later GetSecret call on a different request/connection.
func AddSingletonSecretStore(builder di.ContainerBuilder) {
	di.AddSingleton[fluffycore_contracts_secretstore.ISecretStore](builder, func() fluffycore_contracts_secretstore.ISecretStore {
		return &service{data: make(map[string]string)}
	})
}

func (s *service) Set(orgID, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[compositeKey(orgID, key)] = value
}

func (s *service) Get(orgID, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[compositeKey(orgID, key)]
	return value, ok
}

// compositeKey namespaces by org so two orgs can't read/overwrite each
// other's secrets under the same key.
func compositeKey(orgID, key string) string {
	return orgID + "\x00" + key
}
