package database

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_probe "github.com/fluffy-bunny/fluffycore/cmd/echo-example/internal/contracts/probe"
	zerolog "github.com/rs/zerolog"
)

type (
	service struct{}
)

func init() {
	var _ contracts_probe.IProbe = (*service)(nil)
}

// AddSingletonIProbe registers the *service as a singleton.
func AddSingletonIProbe(builder di.ContainerBuilder) {
	di.AddSingleton[contracts_probe.IProbe](builder,
		func() (contracts_probe.IProbe, error) {
			return &service{}, nil
		})
}
func (s *service) GetName() string {
	return "database"
}
func (s *service) Probe(ctx context.Context) error {
	log := zerolog.Ctx(ctx).With().Logger()
	log.Debug().Str("probe", "database").Send()
	//return errors.New("DataBase is down")
	return nil
}
