package logging

import (
	"context"
	"io"
	"testing"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_propertybag "github.com/fluffy-bunny/fluffycore/contracts/propertybag"
	middleware_dicontext "github.com/fluffy-bunny/fluffycore/middleware/dicontext"
	fluffycore_services_common "github.com/fluffy-bunny/fluffycore/services/common"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	grpc "google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
)

// dicontext scope + context logging, with one log line from the handler.
func BenchmarkScopeAndLoggingInterceptors(b *testing.B) {
	prev := zlog.Logger
	zlog.Logger = zerolog.New(io.Discard)
	defer func() { zlog.Logger = prev }()

	builder := di.Builder()
	fluffycore_services_common.AddCommonServices(builder)
	root := builder.Build()
	scope := middleware_dicontext.ScopedContextUnaryServerInterceptor(root)
	logging := EnsureContextLoggingUnaryServerInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Svc/M"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		bag := di.Get[fluffycore_contracts_propertybag.IRequestContextLoggingPropertyBag](middleware_dicontext.GetRequestContainer(ctx))
		bag.Set("sub", "user-1")
		bag.Set("aud", []string{"a", "b"})
		zerolog.Ctx(ctx).Info().Msg("hello")
		return nil, nil
	}
	chain := func(ctx context.Context, req interface{}) (interface{}, error) {
		return logging(ctx, req, info, handler)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := scope(ctx, nil, info, chain); err != nil {
			b.Fatal(err)
		}
	}
}
