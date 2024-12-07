package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	annotations "github.com/fluffy-bunny/fluffycore/nats/api/annotations"
	nats_micro_service "github.com/fluffy-bunny/fluffycore/nats/nats_micro_service"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	protogen "google.golang.org/protobuf/compiler/protogen"
	proto "google.golang.org/protobuf/proto"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
)

const (
	contextPackage                   = protogen.GoImportPath("context")
	reflectPackage                   = protogen.GoImportPath("reflect")
	fmtPackage                       = protogen.GoImportPath("fmt")
	stringsPackage                   = protogen.GoImportPath("strings")
	errorsPackage                    = protogen.GoImportPath("errors")
	timePackage                      = protogen.GoImportPath("time")
	grpcPackage                      = protogen.GoImportPath("google.golang.org/grpc")
	grpcGatewayRuntimePackage        = protogen.GoImportPath("github.com/grpc-ecosystem/grpc-gateway/v2/runtime")
	grpcStatusPackage                = protogen.GoImportPath("google.golang.org/grpc/status")
	grpcCodesPackage                 = protogen.GoImportPath("google.golang.org/grpc/codes")
	diPackage                        = protogen.GoImportPath("github.com/fluffy-bunny/fluffy-dozm-di")
	reflectxPackage                  = protogen.GoImportPath("github.com/fluffy-bunny/fluffy-dozm-di/reflectx")
	diContextPackage                 = protogen.GoImportPath("github.com/fluffy-bunny/fluffycore/middleware/dicontext")
	contractsEndpointPackage         = protogen.GoImportPath("github.com/fluffy-bunny/fluffycore/contracts/endpoint")
	natsGoPackage                    = protogen.GoImportPath("github.com/nats-io/nats.go")
	natsGoMicroPackage               = protogen.GoImportPath("github.com/nats-io/nats.go/micro")
	contractsNatsMicroServicePackage = protogen.GoImportPath("github.com/fluffy-bunny/fluffycore/contracts/nats_micro_service")
	fluffyCoreUtilsPackage           = protogen.GoImportPath("github.com/fluffy-bunny/fluffycore/utils")
	serviceNatsMicroServicePackage   = protogen.GoImportPath("github.com/fluffy-bunny/fluffycore/nats/nats_micro_service")
	protojsonPackage                 = protogen.GoImportPath("google.golang.org/protobuf/encoding/protojson")
	natsClientPackage                = protogen.GoImportPath("github.com/fluffy-bunny/fluffycore/nats/client")
	annotationsPackage               = protogen.GoImportPath("github.com/fluffy-bunny/fluffycore/nats/api/annotations")
)

type genFileContext struct {
	packageName string
	uniqueRunID string
	gen         *protogen.Plugin
	file        *protogen.File
	filename    string
	g           *protogen.GeneratedFile
}

func newMethodGenContext(uniqueRunId string, protogenMethod *protogen.Method, gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) *methodGenContext {
	ctx := &methodGenContext{
		uniqueRunID:    uniqueRunId,
		MethodInfo:     &MethodInfo{},
		ProtogenMethod: protogenMethod,
		gen:            gen,
		file:           file,
		g:              g,
		service:        service,
	}
	return ctx
}
func newGenFileContext(gen *protogen.Plugin, file *protogen.File) *genFileContext {
	ctx := &genFileContext{
		file:        file,
		gen:         gen,
		uniqueRunID: randomString(32),
		packageName: string(file.GoPackageName),
		filename:    file.GeneratedFilenamePrefix + "_fluffycore_nats.pb.go",
	}
	ctx.g = gen.NewGeneratedFile(ctx.filename, file.GoImportPath)
	return ctx
}
func isServiceIgnored(service *protogen.Service) bool {
	// Look for a comment consisting of "fluffycore:nats:ignore"
	const ignore = "fluffycore:nats:ignore"
	for _, comment := range service.Comments.LeadingDetached {
		if strings.Contains(string(comment), ignore) {
			return true
		}
	}

	return strings.Contains(string(service.Comments.Leading), ignore)
}
func ExtractLastPart(namespace string) string {
	parts := strings.Split(namespace, ":")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func extractNatsNamespaces(input string) []string {
	// Regular expression to match fluffycore:nats:namespace:* pattern
	regex := regexp.MustCompile(`fluffycore:nats:namespace:[^\s\n]+`)

	// Find all matches in the input string
	matches := regex.FindAllString(input, -1)

	return matches
}

func getServiceNamespace(service *protogen.Service) string {
	// Look for a comment consisting of "fluffycore:nats:namespace:roger"
	ss := service.Comments.Leading.String()
	namespaceTags := extractNatsNamespaces(ss)
	for _, match := range namespaceTags {
		ns := ExtractLastPart(string(match))
		return ns
	}

	return ""
}

// generateFile generates a _di.pb.go file containing gRPC service definitions.
func generateFile(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {
	if len(file.Services) == 0 {
		return nil
	}
	ctx := newGenFileContext(gen, file)
	g := ctx.g

	// Default to skip - will unskip if there is a service to generate
	g.Skip()

	g.P("// Code generated by protoc-gen-go-fluffycore-nats. DO NOT EDIT.")

	g.P()
	g.P("package ", file.GoPackageName)
	g.P()
	ctx.generateFileContent()
	return g
}

type MethodInfo struct {
	NewResponseWithErrorFunc string
	NewResponseFunc          string
	ExecuteFunc              string
}
type methodGenContext struct {
	MethodInfo     *MethodInfo
	ProtogenMethod *protogen.Method
	gen            *protogen.Plugin
	file           *protogen.File
	g              *protogen.GeneratedFile
	service        *protogen.Service
	uniqueRunID    string
}
type serviceGenContext struct {
	packageName     string
	MethodMapGenCtx map[string]*methodGenContext
	gen             *protogen.Plugin
	file            *protogen.File
	g               *protogen.GeneratedFile
	service         *protogen.Service
	uniqueRunID     string
}

func newServiceGenContext(packageName string, uniqueRunId string, gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) *serviceGenContext {
	ctx := &serviceGenContext{
		packageName:     packageName,
		uniqueRunID:     uniqueRunId,
		gen:             gen,
		file:            file,
		g:               g,
		service:         service,
		MethodMapGenCtx: make(map[string]*methodGenContext),
	}
	return ctx
}

// generateFileContent generates the DI service definitions, excluding the package statement.
func (s *genFileContext) generateFileContent() {
	gen := s.gen
	file := s.file
	g := s.g

	//var serviceGenCtxs []*serviceGenContext
	// Generate each service
	for _, service := range file.Services {
		// Check if this service is ignored for DI purposes
		if isServiceIgnored(service) {
			continue
		}

		// Now we have something to generate
		g.Unskip()

		serviceGenCtx := newServiceGenContext(s.packageName, s.uniqueRunID, gen, file, g, service)
		serviceGenCtx.genService()
		serviceGenCtx.genClient()
		//serviceGenCtxs = append(serviceGenCtxs, serviceGenCtx)
	}
}
func (s *serviceGenContext) genClient() {
	//	gen := s.gen
	//file := s.file
	//	proto := file.Proto
	g := s.g
	service := s.service

	atLeastOneMethod := false
	for _, method := range service.Methods {
		// only do it if it is not streaming
		if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
			atLeastOneMethod = true
			break
		}
	}
	if !atLeastOneMethod {
		return
	}
	internalClientName := fmt.Sprintf("%vNATSMicroClient", service.GoName)

	/*
			type (
			serviceNATSClientServiceClient struct {
				nc *nats.Conn
			}
		)
	*/
	g.P("type (")
	g.P("	", internalClientName, " struct {")
	g.P("		client *", natsClientPackage.Ident("NATSClient"))
	g.P("	}")
	g.P(")")

	/*
		func NewGreeterNATSMicroClient(  opt ...nats_client.NATSClientOption) (GreeterClient, error) {
			client,err := nats_client.NewNATSClient(opt...)
			if err != nil {
				return nil, err
			}

			pkgPath := reflect.TypeOf((*GreeterServer)(nil)).Elem().PkgPath()
			fullPath := fmt.Sprintf("%s/%s", pkgPath, "Greeter")
			groupName := strings.ReplaceAll(
				fullPath,
				"/",
				".",
			)

			return &GreeterNATSMicroClient{
				client: client,
		 		groupName: groupName,
			}, nil
		}

	*/
	g.P("func New", internalClientName, "(opts ...", natsClientPackage.Ident("NATSClientOption"), ") (", s.service.GoName, "Client, error) {")
	g.P("	client,err := ", natsClientPackage.Ident("NewNATSClient"), "(opts...)")
	g.P("	if err != nil {")
	g.P("		return nil, err")
	g.P("	}")
	g.P(" 	")
	g.P("	return &", internalClientName, "{")
	g.P("		client: client,")
	g.P("	}, nil")
	g.P("}")

	/*
			func (s *serviceNATSClientServiceClient) CreateNATSClient(ctx context.Context, in *cloud_api_business_nats.CreateNATSClientRequest, opts ...grpc.CallOption) (*cloud_api_business_nats.CreateNATSClientResponse, error) {

			response := &cloud_api_business_nats.CreateNATSClientResponse{}

			result, err := HandleNATSRequest(
				ctx,
				s.nc,
				"go.mapped.dev.proto.cloud.api.business.nats.NATSClientService.CreateNATSClient",
				in,
				response,
				2*time.Second,
			)

			return result, err

		}
	*/

	for _, method := range service.Methods {
		serverType := method.Parent.GoName
		key := "/" + *s.file.Proto.Package + "." + serverType + "/" + method.GoName
		methodGenCtx := s.MethodMapGenCtx[key]
		methodGenCtx.generateClientMethodShim()
	}
}
func (s *methodGenContext) grpcClientMethodSignature() string {
	// 	CreateNATSClient(ctx context.Context, in *CreateNATSClientRequest, opts ...grpc.CallOption) (*CreateNATSClientResponse, error)

	g := s.g
	method := s.ProtogenMethod
	var reqArgs []string
	ret := ""
	if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
		reqArgs = append(reqArgs, "ctx "+g.QualifiedGoIdent(contextPackage.Ident("Context")))

		reqArgs = append(reqArgs, "in *"+g.QualifiedGoIdent(method.Input.GoIdent))
		reqArgs = append(reqArgs, "opts ... "+g.QualifiedGoIdent(grpcPackage.Ident("CallOption")))
		ret = "(*" + g.QualifiedGoIdent(method.Output.GoIdent) + ", error)"

	}

	return method.GoName + "(" + strings.Join(reqArgs, ", ") + ") " + ret
}
func (s *methodGenContext) generateClientMethodShim() {
	/*
		func (s *GreeterNATSMicroClient) SayHello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error) {
			response := &HelloReply{}
			result, err := nats_micro_service1.HandleNATSClientRequest(
				ctx,
				s.client,
				fmt.Sprintf("%s.SayHello", s.groupName),
				in,
				response,
			)
			return result, err
		}
	*/
	method := s.ProtogenMethod
	hr := hrvFromMethod(s.file.Proto, method)
	if hr == nil {
		return
	}
	g := s.g
	service := s.service
	internalClientName := fmt.Sprintf("%vNATSMicroClient", service.GoName)
	g.P("// ", s.ProtogenMethod.GoName, "...")
	g.P("func (s *", internalClientName, ") ", s.grpcClientMethodSignature(), "{")
	g.P("	response := &", method.Output.GoIdent.GoName, "{}")
	g.P("	result, err := ", serviceNatsMicroServicePackage.Ident("HandleNATSClientRequest"), "(")
	g.P("		ctx,")
	g.P("		s.client,")
	g.P("		\"", hr.ParameterizedToken, "\",")
	g.P("		in,")
	g.P("		response,")
	g.P("	)")
	g.P("   return result, err")
	g.P("}")
	g.P()

}
func debugPause() {
	fmt.Println("Debug break: Press Enter to continue...")
	// Create a channel to listen for interrupt signals

	time.Sleep(10 * time.Second)
}

func getHandlerRule(
	method *protogen.Method,
) *annotations.HandlerRule {
	return proto.GetExtension(method.Desc.Options(), annotations.E_Handler).(*annotations.HandlerRule)
}
func hrvFromMethod(proto *descriptorpb.FileDescriptorProto, method *protogen.Method) *nats_micro_service.NATSMicroHandlerInfo {
	// we may not have anything, but we still have a subject so we will return that in the wildcard token

	hr := getHandlerRule(method)
	if hr == nil {
		return &nats_micro_service.NATSMicroHandlerInfo{
			WildcardToken:      method.GoName,
			ParameterizedToken: method.GoName,
		}
	}
	wildcardToken := hr.WildcardToken
	paramaterizedToken := hr.ParameterizedToken

	if fluffycore_utils.IsNotEmptyOrNil(wildcardToken) {
		wildcardToken = "." + wildcardToken
	}
	if fluffycore_utils.IsNotEmptyOrNil(paramaterizedToken) {
		paramaterizedToken = "." + paramaterizedToken
	}

	return &nats_micro_service.NATSMicroHandlerInfo{
		WildcardToken:      method.GoName + wildcardToken,
		ParameterizedToken: method.GoName + paramaterizedToken,
	}

}
func (s *serviceGenContext) genService() {
	gen := s.gen
	file := s.file
	proto := file.Proto
	g := s.g
	service := s.service
	//debugPause()
	// IServiceEndpointRegistration

	namespace := getServiceNamespace(service)
	groupName := *proto.Package + "." + service.GoName
	if fluffycore_utils.IsNotEmptyOrNil(namespace) {
		namespace = namespace + "."
	}
	groupName = namespace + groupName

	internalRegistrationServerName := fmt.Sprintf("%vFluffyCoreServerNATSMicroRegistration", service.GoName)

	for _, method := range service.Methods {
		serverType := method.Parent.GoName
		key := "/" + *proto.Package + "." + serverType + "/" + method.GoName
		methodGenCtx := newMethodGenContext(s.uniqueRunID, method, gen, file, g, service)
		s.MethodMapGenCtx[key] = methodGenCtx
	}
	atLeastOneMethod := false
	for _, method := range service.Methods {
		// only do it if it is not streaming
		if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
			atLeastOneMethod = true
			break
		}
	}

	if atLeastOneMethod {

		g.P("var method", service.GoName, "HandlerRuleMap = map[string]*", serviceNatsMicroServicePackage.Ident("NATSMicroHandlerInfo"), " {")
		for _, method := range service.Methods {
			hr := hrvFromMethod(proto, method)
			if hr == nil {
				continue
			}
			serverType := method.Parent.GoName
			key := "/" + *proto.Package + "." + serverType + "/" + method.GoName
			hrV := "{  WildcardToken: \"" + hr.WildcardToken + "\",ParameterizedToken: \"" + hr.ParameterizedToken + "\"}"
			g.P("	\"", key, "\":", hrV, ",")
		}
		g.P("}")
		g.P(" ")

		// need to map 	Greeter_SayHello_FullMethodName           = "/helloworld.Greeter/SayHello"
		// to the nats full subject by providing a function.
		g.P("func MethodToSubject_", service.GoName, "(method string) (string,bool){")
		g.P("	ret,ok := method", service.GoName, "HandlerRuleMap[method]")
		g.P("  	if !ok {")
		g.P("  		return \"\",false")
		g.P("  	}")
		g.P("  	return ret.WildcardToken,true")
		g.P("}")
		g.P()

		g.P("func SendNATSRequestUnaryClientInterceptor_", service.GoName, "(natsClient *", natsClientPackage.Ident("NATSClient"), ")", grpcPackage.Ident("UnaryClientInterceptor"), "{")
		g.P("  	return  ", serviceNatsMicroServicePackage.Ident("SendNATSRequestInterceptor"), "(natsClient,MethodToSubject_", service.GoName, ")")
		g.P("}")
		g.P()
	}
	g.P("type ", internalRegistrationServerName, " struct {")
	g.P("}")
	g.P()

	g.P("var stemService", internalRegistrationServerName, " = (*", internalRegistrationServerName, ") (nil)")
	g.P("var _ ", contractsEndpointPackage.Ident("INATSEndpointRegistration"), " = stemService", internalRegistrationServerName)
	g.P()

	g.P("func AddSingleton", internalRegistrationServerName, "(cb ", diPackage.Ident("ContainerBuilder"), ") {")
	g.P("  	", diPackage.Ident("AddSingleton"), "[", contractsEndpointPackage.Ident("INATSEndpointRegistration"), "](cb, ", "stemService", internalRegistrationServerName, ".Ctor)")
	g.P("}")
	g.P()

	g.P("func (s *", internalRegistrationServerName, ") Ctor() (", contractsEndpointPackage.Ident("INATSEndpointRegistration"), ", error) {")
	g.P("  	return &", internalRegistrationServerName, "{")
	g.P("  	}, nil")
	g.P("}")
	g.P()

	g.P("func (s *", internalRegistrationServerName, ") RegisterFluffyCoreNATSHandler(ctx ", contextPackage.Ident("Context"), ",natsConn *", natsGoPackage.Ident("Conn"), ",conn *", grpcPackage.Ident("ClientConn"), ", option *", contractsNatsMicroServicePackage.Ident("NATSMicroServiceRegisrationOption"), ") (", natsGoMicroPackage.Ident("Service"), ", error) {")
	// 	return RegisterGreeterNATSHandler(ctx,natsCon, conn,option)
	g.P("  	return Register", service.GoName, "NATSHandler(ctx, natsConn, conn, option)")
	g.P("}")
	g.P()

	/*
			 func RegisterGreeterNATSHandler(ctx context.Context,natsCon *nats.Conn, conn *grpc.ClientConn,option *nats_micro_service.NATSMicroServiceRegisrationOption) (micro.Service, error) {
			client := NewGreeterClient(conn)
		   	return RegisterGreeterNATSHandlerClient(ctx,natsCon, client,option)
		}
	*/
	g.P("func Register", service.GoName, "NATSHandler(ctx ", contextPackage.Ident("Context"), ",natsCon *", natsGoPackage.Ident("Conn"), ", conn *", grpcPackage.Ident("ClientConn"), ",option *", contractsNatsMicroServicePackage.Ident("NATSMicroServiceRegisrationOption"), ") (", natsGoMicroPackage.Ident("Service"), ", error) {")
	g.P("	client := New", service.GoName, "Client(conn)")
	g.P("  	return Register", service.GoName, "NATSHandlerClient(ctx,natsCon, client,option)")
	g.P("}")
	g.P()
	/*
	   func RegisterGreeterNATSHandlerClient(ctx context.Context, nc *nats.Conn, client GreeterClient,option *nats_micro_service.NATSMicroServiceRegisrationOption) (micro.Service, error) {
	   	defaultConfig := &micro.Config{
	   		Name:        "Greeter",
	   		Version:     "0.0.1",
	   		Description: "The Greeter nats micro service",
	   	}
	   	for _, option := range option.NATSMicroConfigOptions {
	   		option(defaultConfig)
	   	}

	   	pkgPath := reflect.TypeOf((*GreeterServer)(nil)).Elem().PkgPath()
	   	fullPath := fmt.Sprintf("%s/%s", pkgPath, "Greeter")
	   	groupName := strings.ReplaceAll(
	   		fullPath,
	   		"/",
	   		".",
	   	)
	   	if utils.IsNotEmptyOrNil(option.GroupName) {
	   		groupName = option.GroupName
	   	}
	   	svc, err := micro.AddService(nc, *defaultConfig)
	   	if err != nil {
	   		return nil, err
	   	}

	   	m := svc.AddGroup(groupName)
	   //--
	   	m.AddEndpoint("SayHello",
	   		micro.HandlerFunc(func(req micro.Request) {
	   			nats_micro_service1.HandleRequest(
	   				req,
	   				func(r *HelloRequest) error {
	   					return protojson.Unmarshal(req.Data(), r)
	   				},
	   				func(ctx context.Context, request *HelloRequest) (*HelloReply, error) {
	   					return client.SayHello(ctx,request)
	   				},
	   			)
	   		}),
	   		micro.WithEndpointMetadata(map[string]string{
	   			"description":     "SayHello",
	   			"format":          "application/json",
	   			"request_schema":  utils.SchemaFor(&HelloRequest{}),
	   			"response_schema": utils.SchemaFor(&HelloReply{}),
	   		}))
	   		//--
	   		m.AddEndpoint("SayHelloAuth",
	   		micro.HandlerFunc(func(req micro.Request) {
	   			nats_micro_service1.HandleRequest(
	   				req,
	   				func(r *HelloRequest) error {
	   					return protojson.Unmarshal(req.Data(), r)
	   				},
	   				func(ctx context.Context, request *HelloRequest) (*HelloReply, error) {
	   					return client.SayHelloAuth(ctx,request)
	   				},
	   			)
	   		}),
	   		micro.WithEndpointMetadata(map[string]string{
	   			"description":     "SayHelloDownstream",
	   			"format":          "application/json",
	   			"request_schema":  utils.SchemaFor(&HelloRequest{}),
	   			"response_schema": utils.SchemaFor(&HelloReply{}),
	   		}))
	   		m.AddEndpoint("SayHelloDownstream",
	   		micro.HandlerFunc(func(req micro.Request) {
	   			nats_micro_service1.HandleRequest(
	   				req,
	   				func(r *HelloRequest) error {
	   					return protojson.Unmarshal(req.Data(), r)
	   				},
	   				func(ctx context.Context, request *HelloRequest) (*HelloReply, error) {
	   					return client.SayHelloDownstream(ctx,request)
	   				},
	   			)
	   		}),
	   		micro.WithEndpointMetadata(map[string]string{
	   			"description":     "SayHelloDownstream",
	   			"format":          "application/json",
	   			"request_schema":  utils.SchemaFor(&HelloRequest{}),
	   			"response_schema": utils.SchemaFor(&HelloReply{}),
	   		}))
	   	return svc,nil
	   }
	*/
	g.P("func Register", service.GoName, "NATSHandlerClient(ctx ", contextPackage.Ident("Context"), ", nc *", natsGoPackage.Ident("Conn"), ", client ", service.GoName, "Client, option *", contractsNatsMicroServicePackage.Ident("NATSMicroServiceRegisrationOption"), ") (", natsGoMicroPackage.Ident("Service"), ", error) {")
	g.P("  	defaultConfig := &", natsGoMicroPackage.Ident("Config"), "{")
	g.P("  		Name:        \"", service.GoName, "\",")
	g.P("  		Version:     \"0.0.1\",")
	// we need to pull the Descriptin from the proto service comments.
	g.P("  		Description: \"The ", service.GoName, " nats micro service\",")
	g.P("  	}")
	g.P(" ")
	g.P("  	for _, option := range option.NATSMicroConfigOptions {")
	g.P("  		option(defaultConfig)")
	g.P("  	}")
	g.P(" ")
	g.P("  	svc, err := ", natsGoMicroPackage.Ident("AddService"), "(nc, *defaultConfig)")
	g.P("  	if err != nil {")
	g.P("  		return nil, err")
	g.P("  	}")
	g.P(" ")
	// 	   	m := svc.AddGroup(groupName)
	if atLeastOneMethod {
		g.P("  	m := svc.AddGroup(\"", groupName, "\")")

		//mm := make(map[string]bool)
		for _, method := range service.Methods {

			serverType := method.Parent.GoName
			key := "/" + *proto.Package + "." + serverType + "/" + method.GoName
			methodGenCtx := s.MethodMapGenCtx[key]
			// only do it if it is not streaming
			if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
				methodGenCtx.generateNATSMethodGRPCGateway()
			}
		}
	}
	g.P("  	return svc,nil")
	g.P("  	}")

}

/*
m.AddEndpoint("SayHello",

	micro.HandlerFunc(func(req micro.Request) {
		nats_micro_service1.HandleRequest(
			req,
			func(r *HelloRequest) error {
				return protojson.Unmarshal(req.Data(), r)
			},
			func(ctx context.Context, request *HelloRequest) (*HelloReply, error) {
				return client.SayHello(ctx,request)
			},
		)
	}),
	micro.WithEndpointMetadata(map[string]string{
		"description":     "SayHello",
		"format":          "application/json",
		"request_schema":  utils.SchemaFor(&HelloRequest{}),
		"response_schema": utils.SchemaFor(&HelloReply{}),
	}))
*/
func (s *methodGenContext) generateNATSMethodGRPCGateway() {

	method := s.ProtogenMethod
	hr := hrvFromMethod(s.file.Proto, method)
	if hr == nil {
		return
	}
	g := s.g

	g.P("	m.AddEndpoint(\"", hr.WildcardToken, "\",")
	g.P("		", natsGoMicroPackage.Ident("HandlerFunc"), "(func(req micro.Request) {")
	g.P("			", serviceNatsMicroServicePackage.Ident("HandleRequest"), "(")
	g.P("				req,")
	g.P("				func(r *", method.Input.GoIdent.GoName, ") error {")
	g.P("					return ", protojsonPackage.Ident("Unmarshal"), "(req.Data(), r)")
	g.P("				},")
	g.P("				func(ctx ", contextPackage.Ident("Context"), ", request *", method.Input.GoIdent.GoName, ") (*", method.Output.GoIdent.GoName, ", error) {")
	g.P("					return client.", method.GoName, "(ctx,request)")
	g.P("				},")
	g.P("			)")
	g.P("		}),")
	g.P("		", natsGoMicroPackage.Ident("WithEndpointMetadata"), "(map[string]string{")
	g.P("			\"description\":     \"", method.GoName, "\",")
	g.P("			\"format\":          \"application/json\",")
	g.P("			\"request_schema\": ", fluffyCoreUtilsPackage.Ident("SchemaFor"), "(&", method.Input.GoIdent.GoName, "{}),")
	g.P("			\"response_schema\": ", fluffyCoreUtilsPackage.Ident("SchemaFor"), "(&", method.Output.GoIdent.GoName, "{}),")
	g.P("		}))")
	g.P()
}
