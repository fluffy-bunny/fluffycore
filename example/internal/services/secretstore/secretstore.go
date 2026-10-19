package secretstore

import (
	"fmt"
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

func (s *service) Set(owner, orgID, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[compositeKey(owner, orgID, key)] = value
}

func (s *service) Get(owner, orgID, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[compositeKey(owner, orgID, key)]
	return value, ok
}

// compositeKey namespaces by owner, then org. Parts are length-prefixed so no choice of
// values can make two different (owner, org, key) triples collide.
func compositeKey(owner, orgID, key string) string {
	return fmt.Sprintf("%d:%s|%d:%s|%s", len(owner), owner, len(orgID), orgID, key)
}
