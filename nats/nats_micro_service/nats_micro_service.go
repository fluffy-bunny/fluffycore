package nats_micro_service

import (
	"context"
	"fmt"
	"sync"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_endpoint "github.com/fluffy-bunny/fluffycore/contracts/endpoint"
	contracts_nats_micro_service "github.com/fluffy-bunny/fluffycore/contracts/nats_micro_service"
	interceptor "github.com/fluffy-bunny/fluffycore/nats/nats_micro_service/default/interceptor"
	nats "github.com/nats-io/nats.go"
	micro "github.com/nats-io/nats.go/micro"
	zerolog "github.com/rs/zerolog"
	grpc "google.golang.org/grpc"
	protojson "google.golang.org/protobuf/encoding/protojson"
	proto "google.golang.org/protobuf/proto"
)

type NATSMicroConfig struct {
	NATSUrl         string `json:"natsUrl"`
	ClientID        string `json:"clientId"`
	ClientSecret    string `json:"clientSecret"`
	TimeoutDuration string `json:"timeoutDuration"`
}

func AddNatsMicroConfig(builder di.ContainerBuilder, config *NATSMicroConfig) {
	di.AddInstance[*NATSMicroConfig](builder, config)
}
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
	req micro.Request,
	unmarshaler func(*Req) error,
	serviceMethod func(context.Context, *Req) (*Resp, error),
) {
	var innerRequest Req
	if err := unmarshaler(&innerRequest); err != nil {
		req.Error("400", err.Error(), nil)
		return
	}

	resp, err := serviceMethod(context.Background(), &innerRequest)
	if err != nil {
		req.Error("500", err.Error(), nil)
		return
	}

	// Type assert resp to proto.Message
	respProto, ok := any(resp).(proto.Message)
	if !ok {
		req.Error("500", "response is not a proto.Message", nil)
		return
	}

	// Marshal proto message to JSON
	respBytes, err := protojson.Marshal(respProto)
	if err != nil {
		req.Error("500", err.Error(), nil)
		return
	}

	req.Respond(respBytes)
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
func IsAnyNatsHandler(rootContainer di.Container) bool {
	natsMicroServiceRegistrations := di.Get[[]contracts_endpoint.INATSEndpointRegistration](rootContainer)
	return len(natsMicroServiceRegistrations) > 0
}
func (s *NATSMicroServicesContainer) Register(ctx context.Context, conn *grpc.ClientConn) error {
	s.mutex.Lock()
	defer func() {
		s.mutex.Unlock()
		s.registered = true
	}()
	if s.registered {
		return nil
	}
	log := zerolog.Ctx(ctx).With().Logger()

	natsMicroServiceRegistrations := di.Get[[]contracts_endpoint.INATSEndpointRegistration](s.rootContainer)
	for _, reg := range natsMicroServiceRegistrations {
		natsMicroService, err := reg.RegisterFluffyCoreNATSHandler(ctx, s.nc, conn,
			&contracts_nats_micro_service.NATSMicroServiceRegisrationOption{})
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
func ConvertToStringMap(h micro.Headers) map[string]string {
	result := make(map[string]string)
	for key, values := range h {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}
