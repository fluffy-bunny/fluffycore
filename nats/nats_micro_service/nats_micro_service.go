package nats_micro_service

import (
	"context"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_nats_micro_service "github.com/fluffy-bunny/fluffycore/contracts/nats_micro_service"
	interceptor "github.com/fluffy-bunny/fluffycore/nats/nats_micro_service/default/interceptor"
	status "github.com/gogo/status"
	micro "github.com/nats-io/nats.go/micro"
	zerolog "github.com/rs/zerolog"
	codes "google.golang.org/grpc/codes"
	protojson "google.golang.org/protobuf/encoding/protojson"
	proto "google.golang.org/protobuf/proto"
)

func AddCommonNATSServices(builder di.ContainerBuilder) {
	interceptor.AddSingletonNATSMicroInterceptors(builder)
}

func HandleRequest[Req, Resp any](
	s contracts_nats_micro_service.INATSMicroService,
	req micro.Request,
	unmarshaler func(*Req) error,
	serviceMethod func(context.Context, *Req) (*Resp, error),
) {
	ctx := context.Background()
	log := zerolog.Ctx(ctx).With().Logger()

	handler := s.Interceptors().WithHandler(func(ctx context.Context, req interface{}) (interface{}, error) {
		var innerRequest Req
		if err := unmarshaler(&innerRequest); err != nil {
			log.Error().Err(err).Msg("unable to parse request")
			err := status.Error(codes.InvalidArgument, "unable to parse request")
			return nil, err
		}
		return serviceMethod(ctx, &innerRequest)
	})

	resp, err := handler(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("error")
		req.Error("500", err.Error(), nil)
		return
	}
	// typecase resp to a protomessage
	respProto, ok := resp.(proto.Message)
	if !ok {
		log.Error().Msg("response is not a proto.Message")
		err := status.Error(codes.Internal, "response is not a proto.Message")
		req.Error("500", err.Error(), nil)
		return
	}
	pbJsonBytes, err := protojson.Marshal(respProto)
	if err != nil {
		log.Error().Err(err).Msg("unable to marshal response")
		err := status.Error(codes.Internal, "unable to marshal response")
		req.Error("500", err.Error(), nil)
		return
	}
	req.Respond(pbJsonBytes)
}
