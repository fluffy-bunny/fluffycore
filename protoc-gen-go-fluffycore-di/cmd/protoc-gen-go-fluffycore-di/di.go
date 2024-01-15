package main

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

const (
	contextPackage            = protogen.GoImportPath("context")
	errorsPackage             = protogen.GoImportPath("errors")
	grpcPackage               = protogen.GoImportPath("google.golang.org/grpc")
	grpcGatewayRuntimePackage = protogen.GoImportPath("github.com/grpc-ecosystem/grpc-gateway/v2/runtime")
	grpcStatusPackage         = protogen.GoImportPath("google.golang.org/grpc/status")
	grpcCodesPackage          = protogen.GoImportPath("google.golang.org/grpc/codes")
	diPackage                 = protogen.GoImportPath("github.com/fluffy-bunny/fluffy-dozm-di")
	reflectxPackage           = protogen.GoImportPath("github.com/fluffy-bunny/fluffy-dozm-di/reflectx")
	diContextPackage          = protogen.GoImportPath("github.com/fluffy-bunny/fluffycore/middleware/dicontext")
	contractsEndpointPackage  = protogen.GoImportPath("github.com/fluffy-bunny/fluffycore/contracts/endpoint")
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
		filename:    file.GeneratedFilenamePrefix + "_fluffycore_di.pb.go",
	}
	ctx.g = gen.NewGeneratedFile(ctx.filename, file.GoImportPath)
	return ctx
}
func isServiceIgnored(service *protogen.Service) bool {
	// Look for a comment consisting of "di:ignore"
	const ignore = "di:ignore"
	for _, comment := range service.Comments.LeadingDetached {
		if strings.Contains(string(comment), ignore) {
			return true
		}
	}

	return strings.Contains(string(service.Comments.Leading), ignore)
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

	g.P("// Code generated by protoc-gen-go-fluffycore-di. DO NOT EDIT.")
	if *grpcGatewayEnabled {
		g.P("// Code generated grpcGateway")
	}
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
		//serviceGenCtxs = append(serviceGenCtxs, serviceGenCtx)
	}
}
func (s *serviceGenContext) genService() {
	gen := s.gen
	file := s.file
	proto := file.Proto
	g := s.g
	service := s.service

	// IServiceEndpointRegistration
	interfaceGRPCServerName := fmt.Sprintf("%vServer", service.GoName)

	interfaceServerName := fmt.Sprintf("IFluffyCore%s", interfaceGRPCServerName)
	internalServerName := fmt.Sprintf("%vFluffyCoreServer", service.GoName)

	g.P("// ", interfaceServerName, " defines the grpc server")
	g.P("type ", interfaceServerName, " interface {")
	g.P("  	", service.GoName, "Server")
	g.P("}")
	g.P()

	g.P("type UnimplementedFluffyCore", service.GoName, "ServerEndpointRegistration struct {")
	g.P("}")
	g.P()

	g.P("func (UnimplementedFluffyCore", service.GoName, "ServerEndpointRegistration) RegisterHandler(gwmux *", grpcGatewayRuntimePackage.Ident("ServeMux"),
		",conn *", grpcPackage.Ident("ClientConn"), ") {")
	g.P("}")
	g.P()

	// Define the ServiceEndpointRegistration implementation
	//----------------------------------------------------------------------------------------------
	g.P("// ", internalServerName, " defines the grpc server truct")
	g.P("type ", internalServerName, " struct {")
	g.P("  	", "Unimplemented", service.GoName, "Server")
	g.P("  	", "UnimplementedFluffyCore", service.GoName, "ServerEndpointRegistration")
	g.P("}")
	g.P()

	g.P("// Register the server with grpc")
	g.P("func (srv *", internalServerName, ") Register(s *", grpcPackage.Ident("Server"), ") {")
	g.P("   ", "Register", interfaceGRPCServerName, "(s,srv)")
	g.P("}")

	g.P("// Add", service.GoName, "ServerWithExternalRegistration", " adds the fluffycore aware grpc server and external registration service.  Mainly used for grpc-gateway")
	g.P("func Add", service.GoName, "ServerWithExternalRegistration[T ", interfaceServerName, "](cb ", diPackage.Ident("ContainerBuilder"), ", ctor any, register func() ", contractsEndpointPackage.Ident("IEndpointRegistration"), " ) {")
	g.P("   ", diPackage.Ident("AddSingleton"), "[", contractsEndpointPackage.Ident("IEndpointRegistration"), "](cb,register)")
	g.P("   ", diPackage.Ident("AddScoped"), "[", interfaceServerName, "](cb,ctor)")
	g.P("}")

	g.P("// Add", service.GoName, "Server", " adds the fluffycore aware grpc server")
	g.P("func Add", service.GoName, "Server[T ", interfaceServerName, "](cb ", diPackage.Ident("ContainerBuilder"), ", ctor any) {")
	g.P("   Add", service.GoName, "ServerWithExternalRegistration[", interfaceServerName, "](cb,ctor,func() ", contractsEndpointPackage.Ident("IEndpointRegistration"), " {")
	g.P("      return &", internalServerName, "{}")
	g.P("   })")
	g.P("}")

	for _, method := range service.Methods {
		serverType := method.Parent.GoName
		key := "/" + *proto.Package + "." + serverType + "/" + method.GoName
		methodGenCtx := newMethodGenContext(s.uniqueRunID, method, gen, file, g, service)
		s.MethodMapGenCtx[key] = methodGenCtx
	}
	// Client method implementations.
	for _, method := range service.Methods {
		serverType := method.Parent.GoName
		key := "/" + *proto.Package + "." + serverType + "/" + method.GoName
		methodGenCtx := s.MethodMapGenCtx[key]
		methodGenCtx.genServerMethodShim()
	}
}
func (s *methodGenContext) genServerMethodShim() {

	method := s.ProtogenMethod

	if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
		s.generateUnaryServerMethodShim()
	} else {
		s.generateStreamServerMethodShim()
	}

}
func (s *methodGenContext) generateUnaryServerMethodShim() {

	method := s.ProtogenMethod
	g := s.g
	serverType := method.Parent.GoName
	interfaceServerName := fmt.Sprintf("IFluffyCore%vServer", method.Parent.GoName)
	internalServerName := fmt.Sprintf("%vFluffyCoreServer", serverType)

	g.P("// ", s.ProtogenMethod.GoName, "...")
	g.P("func (s *", internalServerName, ") ", s.unaryMethodSignature(), "{")
	g.P("requestContainer := ", diContextPackage.Ident("GetRequestContainer(ctx)"))
	g.P("downstreamService := ", diPackage.Ident("Get"), "[", interfaceServerName, "](requestContainer)")
	g.P("return downstreamService.", method.GoName, "(ctx,request)")
	g.P("}")
	g.P()
}
func (s *methodGenContext) generateStreamServerMethodShim() {
	method := s.ProtogenMethod
	g := s.g
	serverType := method.Parent.GoName
	interfaceServerName := fmt.Sprintf("IFluffyCore%vServer", method.Parent.GoName)
	internalServerName := fmt.Sprintf("%vFluffyCoreServer", serverType)

	sig, argCount := s.streamMethodSignature()

	g.P("// ", s.ProtogenMethod.GoName, "...")
	g.P("func (s *", internalServerName, ") ", sig, "{")
	g.P("ctx := stream.Context()")
	g.P("requestContainer := ", diContextPackage.Ident("GetRequestContainer(ctx)"))
	g.P("downstreamService := ", diPackage.Ident("Get"), "[", interfaceServerName, "](requestContainer)")
	if argCount == 1 {
		// stream only
		g.P("return downstreamService.", method.GoName, "(stream)")
	} else {
		// stream and context
		g.P("return downstreamService.", method.GoName, "(request,stream)")
	}
	g.P("}")
	g.P()

}

func (s *methodGenContext) unaryMethodSignature() string {
	g := s.g
	method := s.ProtogenMethod
	var reqArgs []string
	ret := "error"
	if !method.Desc.IsStreamingClient() && !method.Desc.IsStreamingServer() {
		reqArgs = append(reqArgs, "ctx "+g.QualifiedGoIdent(contextPackage.Ident("Context")))
		ret = "(*" + g.QualifiedGoIdent(method.Output.GoIdent) + ", error)"
	}
	if !method.Desc.IsStreamingClient() {
		reqArgs = append(reqArgs, "request *"+g.QualifiedGoIdent(method.Input.GoIdent))
	}

	return method.GoName + "(" + strings.Join(reqArgs, ", ") + ") " + ret
}
func (s *methodGenContext) streamMethodSignature() (string, int) {
	g := s.g
	method := s.ProtogenMethod
	var reqArgs []string
	ret := "error"

	if !method.Desc.IsStreamingClient() {
		reqArgs = append(reqArgs, "request *"+g.QualifiedGoIdent(method.Input.GoIdent))
	}
	if method.Desc.IsStreamingClient() || method.Desc.IsStreamingServer() {
		reqArgs = append(reqArgs, "stream "+method.Parent.GoName+"_"+method.GoName+"Server")
	}
	return method.GoName + "(" + strings.Join(reqArgs, ", ") + ") " + ret, len(reqArgs)
}
