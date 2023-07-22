package somedisposable

import (
	"context"

	di "github.com/dozm/di"
	fluffycore_contracts_somedisposable "github.com/fluffy-bunny/fluffycore/example/internal/contracts/somedisposable"
	log "github.com/rs/zerolog/log"
)

type (
	serviceScoped struct{}
)

func init() {
	var _ fluffycore_contracts_somedisposable.IScopedSomeDisposable = &serviceScoped{}
}

func AddScopedSomeDisposable(cb di.ContainerBuilder) {
	di.AddScoped[fluffycore_contracts_somedisposable.IScopedSomeDisposable](cb,
		func() fluffycore_contracts_somedisposable.IScopedSomeDisposable {
			return &serviceScoped{}
		})
}

func (s *serviceScoped) Dispose() {
	log.Info().
		Str("service", "serviceScoped").
		Str("interface", "fluffycore_contracts_somedisposable.IScopedSomeDisposable").
		Msg("Dispose")
}
func (s *serviceScoped) DoSomething(ctx context.Context) (string, error) {
	return "Hello World", nil
}
