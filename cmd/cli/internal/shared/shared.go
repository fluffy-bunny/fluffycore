package shared

import (
	"context"
	"reflect"

	"encoding/json"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	oauth2 "golang.org/x/oauth2"
	cc "golang.org/x/oauth2/clientcredentials"
)

func PrettyJSON(obj interface{}) string {
	jsonBytes, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		panic(err)
	}
	return string(jsonBytes)
}

type (
	OAuth2Config struct {
		ClientID       string
		ClientSecret   string
		TokenEndepoint string
	}
)

var (
	OAuth2      OAuth2Config
	tokenSource oauth2.TokenSource
)

type ChatMessage struct {
	Input string `json:"input"`
}

var EnvFilePath string = ".env"
var StripeLiveApiKey string
var StripeSourceEnvApiKey string
var StripeDestinationEnvApiKey string

func GenericPtr[T any](v T) *T {
	return &v
}
func GetTokenSource(ctx context.Context) oauth2.TokenSource {
	if tokenSource == nil {
		config := &cc.Config{
			ClientID:     OAuth2.ClientID,
			ClientSecret: OAuth2.ClientSecret,
			TokenURL:     OAuth2.TokenEndepoint,
			Scopes:       []string{},
		}
		tokenSource = config.TokenSource(ctx)
	}
	return tokenSource
}

var _ctx context.Context

func SetContext(ctx context.Context) {
	_ctx = ctx
}
func GetContext() context.Context {
	return _ctx
}

var _builder di.ContainerBuilder
var _container di.Container

func SetBuilder(builder di.ContainerBuilder) {
	_builder = builder
}

func GetContainer() di.Container {
	return _container
}

type ConfigureServices func(builder di.ContainerBuilder)

func AddServices(configureServices ...ConfigureServices) {
	for _, configureService := range configureServices {
		configureService(_builder)
	}
}

func BuildContainer() {
	_container = _builder.Build()
}

func NilStringSlice(s []string) []*string {
	if fluffycore_utils.IsEmptyOrNil(s) {
		return nil
	}
	dd := StringSlice(s)
	return dd
}

// StringSlice returns a slice of string pointers given a slice of strings.
func StringSlice(v []string) []*string {
	out := make([]*string, len(v))
	for i := range v {
		out[i] = &v[i]
	}
	return out
}
func NilStrPtr(s interface{}) *string {
	if s == nil {
		return nil
	}

	val := reflect.ValueOf(s)

	// Handle pointer to string-like types
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	// Handle string and string-like types
	if val.Kind() == reflect.String {
		str := val.String()
		if fluffycore_utils.IsEmptyOrNil(str) {
			return nil
		}
		return GenericPtr(str)
	}

	return nil
}
