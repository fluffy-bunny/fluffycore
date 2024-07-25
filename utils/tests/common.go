package tests

import (
	"context"
	"net"
	"os"
	"testing"

	gomock "github.com/golang/mock/gomock"
	grpc "google.golang.org/grpc"
	insecure "google.golang.org/grpc/credentials/insecure"
	bufconn "google.golang.org/grpc/test/bufconn"
)

const BufSize = 1024 * 1024

func CreateConnection(ctx context.Context, lis *bufconn.Listener, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	contextDialerOpt := grpc.WithContextDialer(func(c context.Context, s string) (net.Conn, error) {
		return lis.Dial()
	})
	finalOpts := []grpc.DialOption{
		contextDialerOpt,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	//opts = append([]DialOption{withDefaultScheme("passthrough")}, opts...)

	finalOpts = append(finalOpts, opts...)
	// NOTE: grpc.NewClient doesn't seem to work with bufnet.  grpc.DialContext does.
	conn, err := grpc.NewClient("passthrough://bufnet", finalOpts...)
	//conn, err := grpc.DialContext(ctx, "bufnet", finalOpts...)

	if err != nil {
		return nil, err
	}
	return conn, nil
}

func CreateForEach(setUp func(*testing.T), tearDown func()) func(*testing.T, func(*gomock.Controller)) {
	return func(t *testing.T, testFunc func(ctrl *gomock.Controller)) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		setUp(t)
		testFunc(ctrl)
		tearDown()
	}
}

var RunTest = CreateForEach(setUp, tearDown)

func setUp(t *testing.T) {
	// SETUP METHOD WHICH IS REQUIRED TO RUN FOR EACH TEST METHOD
	// your code here
	//t.Setenv("APPLICATION_ENVIRONMENT", "Test")
}

func tearDown() {
	// TEAR DOWN METHOD WHICH IS REQUIRED TO RUN FOR EACH TEST METHOD
	// your code here
}
func SkipCI(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
}
func SetupEnvironment(t *testing.T, envData map[string]string) {
	for key, value := range envData {
		t.Setenv(key, value)
	}
	t.Setenv("UNDER_TEST", "true")
}
