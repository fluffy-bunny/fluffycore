package runtime

import (
	"context"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_nats_micro_service "github.com/fluffy-bunny/fluffycore/nats/nats_micro_service"
	fluffycore_nats_token "github.com/fluffy-bunny/fluffycore/nats/nats_token"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	status "github.com/gogo/status"
	zerolog "github.com/rs/zerolog"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
)

type INATSHandlerGatewayCallback interface {
	SetNATSMicroServicesContainer(container *fluffycore_nats_micro_service.NATSMicroServicesContainer)
}
type StartNATSHandlerGatewayRequest struct {
	Container di.Container
	Conn      *grpc.ClientConn
	Callback  INATSHandlerGatewayCallback
}

func validateStartNATSHandlerGatewayRequest(request *StartNATSHandlerGatewayRequest) error {
	if request == nil {
		return status.Error(codes.InvalidArgument, "request is nil")
	}
	if request.Conn == nil {
		return status.Error(codes.InvalidArgument, "request.Conn is nil")
	}
	if request.Container == nil {
		return status.Error(codes.InvalidArgument, "request.Container is nil")
	}
	if request.Callback == nil {
		return status.Error(codes.InvalidArgument, "request.Callback is nil")
	}
	return nil
}
func StartNATSHandlerGateway(ctx context.Context, request *StartNATSHandlerGatewayRequest) error {
	log := zerolog.Ctx(ctx).With().Logger()
	natsEnabled := fluffycore_utils.BoolEnv("NATS_ENABLED", false)
	if !natsEnabled {
		return nil
	}
	err := validateStartNATSHandlerGatewayRequest(request)
	if err != nil {
		return err
	}

	go func() {
		// pause a bit to let things settle down.
		time.Sleep(1 * time.Second)
		// special case as the hosting service may also be the nats auth service so
		// we will wait a bit before the handlers come on line.

		anyNatsHandler := fluffycore_nats_micro_service.IsAnyNatsHandler(request.Container)
		// no need to do anything if nothing here to be registered
		natsMicroConfig, err := di.TryGet[*fluffycore_nats_micro_service.NATSMicroConfig](request.Container)
		if err != nil {
			log.Error().Err(err).Msg("Could not get *NATSMicroConfig.  You get no nats micros.")
		}
		if err == nil &&
			anyNatsHandler &&
			natsMicroConfig != nil {

			nc, err := fluffycore_nats_token.CreateNatsConnectionWithClientCredentials(
				&fluffycore_nats_token.NATSConnectTokenClientCredentialsRequest{
					NATSUrl:      natsMicroConfig.NATSUrl,
					ClientID:     natsMicroConfig.ClientID,
					ClientSecret: natsMicroConfig.ClientSecret,
				},
			)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to connect to NATS")
			}
			natsMicroServicesContainer := fluffycore_nats_micro_service.NewNATSMicroServicesContainer(
				nc, request.Container,
			)

			err = natsMicroServicesContainer.Register(ctx, request.Conn)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to RegisterNATSMicroServiceHandlers")
			}
			request.Callback.SetNATSMicroServicesContainer(natsMicroServicesContainer)
		}
	}()
	return nil
}
