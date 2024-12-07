package nats_micro_service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_endpoint "github.com/fluffy-bunny/fluffycore/contracts/endpoint"
	contracts_nats_micro_service "github.com/fluffy-bunny/fluffycore/contracts/nats_micro_service"
	nats_client "github.com/fluffy-bunny/fluffycore/nats/client"
	status "github.com/gogo/status"
	jsonpath "github.com/mdaverde/jsonpath"
	nats "github.com/nats-io/nats.go"
	micro "github.com/nats-io/nats.go/micro"
	zerolog "github.com/rs/zerolog"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	metadata "google.golang.org/grpc/metadata"
	protojson "google.golang.org/protobuf/encoding/protojson"
	proto "google.golang.org/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
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
}

type NATSRequestHeaderContainer struct {
	Header map[string][]string
}
type NATSClientOption struct {
	NC      *nats.Conn
	Timeout time.Duration
}

// TokenToValue is assumed to be defined elsewhere in the codebase
// func TokenToValue(token string) interface{}

// ReplaceTokens replaces tokens in the input string using TokenToValue
func ReplaceTokens(paramaterizedToken string, m protoreflect.ProtoMessage) (string, error) {
	// Regular expression to find tokens like ${token}
	tokenRegex := regexp.MustCompile(`\$\{([^}]+)\}`)

	// Find all matches
	matches := tokenRegex.FindAllStringSubmatch(paramaterizedToken, -1)

	// Create a copy of the input string to modify
	replacedString := paramaterizedToken

	pJson, err := protojson.Marshal(m)
	if err != nil {
		return "", err
	}

	var payload interface{}

	err = json.Unmarshal([]byte(pJson), &payload)
	if err != nil {
		return "", err
	}
	// Replace each found token
	for _, match := range matches {
		if len(match) > 1 {
			fullToken := match[0] // The full token like ${orgId}
			tokenName := match[1] // The token name like orgId

			// Use TokenToValue to get the replacement value
			value, err := jsonpath.Get(payload, tokenName)
			if err != nil {
				return "", err
			}
			// Convert the value to string (you might need to adjust this based on actual implementation)
			var stringValue string
			switch v := value.(type) {
			case string:
				stringValue = v
			case int:
				stringValue = strconv.Itoa(v)
			case float64:
				stringValue = strconv.FormatFloat(v, 'f', -1, 64)
			default:
				stringValue = fmt.Sprintf("%v", v)
			}

			// Replace the token with its value
			replacedString = strings.ReplaceAll(replacedString, fullToken, stringValue)
		}
	}

	return replacedString, nil
}

func HandleNATSClientRequest[Req proto.Message, Resp proto.Message](
	ctx context.Context,
	client *nats_client.NATSClient,
	subject string,
	request Req,
	response Resp,
) (Resp, error) {

	// Marshal the request
	msg, err := protojson.Marshal(request)
	if err != nil {
		return response, fmt.Errorf("failed to marshal request: %w", err)
	}

	subject, err = ReplaceTokens(subject, request)
	if err != nil {
		return response, err
	}

	natsResponse, err := client.RequestWithContext(ctx, subject, msg)
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
	ctx := context.Background()
	// propegate all req.Headers to the grpc metadata
	dd := ConvertToStringMap(req.Headers())
	md := metadata.New(dd)
	ctx = metadata.NewOutgoingContext(ctx, md)
	var innerRequest Req
	if err := unmarshaler(&innerRequest); err != nil {
		req.Error("400", err.Error(), nil)
		return
	}

	resp, err := serviceMethod(ctx, &innerRequest)
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

func SendNATSRequestInterceptor(natsClient *nats_client.NATSClient,
	methodToSubject func(string) (string, bool)) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		subject, ok := methodToSubject(method)
		if !ok {
			return status.Error(codes.Internal, "methodToSubject failed")
		}
		// propegate all grpc metadata to the nats headers
		md, ok := metadata.FromOutgoingContext(ctx)
		if ok {
			headers := nats.Header{}
			for k, v := range md {
				headers[k] = v
			}
		}
		// typecase req to a protomessage
		reqProto, ok := req.(protoreflect.ProtoMessage)
		if !ok {
			return status.Error(codes.Internal, "req is not a protoreflect.ProtoMessage")
		}
		// typecase reply to a protomessage
		replyProto, ok := reply.(protoreflect.ProtoMessage)
		if !ok {
			return status.Error(codes.Internal, "reply is not a protoreflect.ProtoMessage")
		}
		// "go.mapped.dev.proto.cloud.api.business.nats.NATSClientService.ListNATSClient"
		// "go.mapped.dev.proto.mapped.cloud.api.business.nats.NATSClientService.ListNATSClient"

		_, err := HandleNATSClientRequest(ctx, natsClient, subject, reqProto, replyProto)

		return err
	}
}
