package jwt

import (
	"context"
	"testing"
	"time"

	grpc "google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
)

func BenchmarkUnaryInterceptor_ValidToken(b *testing.B) {
	t := &testing.T{}
	idp := newFakeIdP(t)
	key := newSigningKey(t, "k1")
	idp.publish(t, key)
	interceptor := withIdP(t, idp, time.Minute)
	token := signedToken(t, idp.issuer(), key)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Svc/M"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) { return nil, nil }
	if _, err := interceptor(ctx, nil, info, handler); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := interceptor(ctx, nil, info, handler); err != nil {
			b.Fatal(err)
		}
	}
}
