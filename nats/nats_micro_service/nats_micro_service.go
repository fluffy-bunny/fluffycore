package nats_micro_service

import (
	"context"
	"fmt"
	"sync"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_nats_micro_service "github.com/fluffy-bunny/fluffycore/contracts/nats_micro_service"
	fluffycore_contracts_nats_micro_service "github.com/fluffy-bunny/fluffycore/contracts/nats_micro_service"
	interceptor "github.com/fluffy-bunny/fluffycore/nats/nats_micro_service/default/interceptor"
	status "github.com/gogo/status"
	nats "github.com/nats-io/nats.go"
	micro "github.com/nats-io/nats.go/micro"
	zerolog "github.com/rs/zerolog"
	codes "google.golang.org/grpc/codes"
	protojson "google.golang.org/protobuf/encoding/protojson"
	proto "google.golang.org/protobuf/proto"
)

func AddCommonNATSServices(builder di.ContainerBuilder) {
	interceptor.AddSingletonNATSMicroInterceptors(builder)
}

type NATSRequestHeaderContainer struct {
	Header map[string][]string
}
type NATSClientOption struct {
	NC      *nats.Conn
	Timeout time.Duration
}

var NATSRequestHeaderContainerKey = &NATSRequestHeaderContainer{}

func WithNATSRequestHeaderContainer(ctx context.Context, headerContainer *NATSRequestHeaderContainer) context.Context {
	return context.WithValue(ctx, NATSRequestHeaderContainerKey, headerContainer)
}
func GetNATSRequestHeaderContainer(ctx context.Context) *NATSRequestHeaderContainer {
	vv := ctx.Value(NATSRequestHeaderContainerKey)
	if vv == nil {
		return &NATSRequestHeaderContainer{}
	}
	return vv.(*NATSRequestHeaderContainer)
}

// HandleNATSRequest is a standalone generic function to handle GRPC to NATS bridge requests
func HandleNATSClientRequest[Req proto.Message, Resp proto.Message](
	ctx context.Context,
	nc *nats.Conn,
	subject string,
	request Req,
	response Resp,
	timeout time.Duration,
) (Resp, error) {

	natsRequestHeaderContainer := GetNATSRequestHeaderContainer(ctx)
	// Pull header from context
	hdr := natsRequestHeaderContainer.Header

	// Marshal the request
	msg, err := protojson.Marshal(request)
	if err != nil {
		return response, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Prepare NATS message
	natsMessage := &nats.Msg{
		Subject: subject,
		Data:    msg,
		Header:  hdr,
	}

	// Send request and wait for response
	natsResponse, err := nc.RequestMsg(natsMessage, timeout)
	if err != nil {
		return response, fmt.Errorf("NATS request failed: %w", err)
	}

	// Unmarshal response
	err = protojson.Unmarshal(natsResponse.Data, response)
	if err != nil {
		return response, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
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

type NATSMicroServicesContainer struct {
	natsMicroSerivices []micro.Service
	nc                 *nats.Conn
	rootContainer      di.Container
	mutex              sync.Mutex
	registered         bool
}

func NewNATSMicroServicesContainer(nc *nats.Conn, rootContainer di.Container) *NATSMicroServicesContainer {
	return &NATSMicroServicesContainer{
		nc:            nc,
		rootContainer: rootContainer,
	}
}
func (s *NATSMicroServicesContainer) Register(ctx context.Context) error {
	s.mutex.Lock()
	defer func() {
		s.mutex.Unlock()
		s.registered = true
	}()
	if s.registered {
		return nil
	}
	log := zerolog.Ctx(ctx).With().Logger()

	natsMicroServiceRegistrations := di.Get[[]fluffycore_contracts_nats_micro_service.INATSMicroServiceRegisration](s.rootContainer)
	for _, reg := range natsMicroServiceRegistrations {
		natsMicroService, err := reg.AddService(s.nc,
			&fluffycore_contracts_nats_micro_service.NATSMicroServiceRegisrationOption{})
		if err != nil {
			log.Error().Err(err).Msg("failed to AddService")
			return err
		}
		s.natsMicroSerivices = append(s.natsMicroSerivices, natsMicroService)
	}
	return nil
}

func (s *NATSMicroServicesContainer) Shutdown(ctx context.Context) error {
	s.mutex.Lock()
	defer func() {
		s.mutex.Unlock()
	}()
	if !s.registered {
		return nil
	}
	log := zerolog.Ctx(ctx).With().Logger()
	err := s.stopNATSMicroServices(ctx, s.natsMicroSerivices)
	if err != nil {
		log.Error().Err(err).Msg("failed to StopNATSMicroServices")
	}
	s.nc.Close()
	return nil
}

func (s *NATSMicroServicesContainer) stopNATSMicroServices(ctx context.Context, ms []micro.Service) error {
	log := zerolog.Ctx(ctx).With().Logger()
	errs := []error{}
	for _, m := range ms {
		err := m.Stop()
		if err != nil {
			log.Error().Err(err).Msg("failed to Shutdown")
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to Stop some services %v", errs)
	}
	return nil
}
