package swagger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	grpc_gateway_runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/require"
)

const sampleSwaggerJSON = `{
  "swagger": "2.0",
  "info": {"title": "Test", "version": "1.0"},
  "tags": [
    {"name": "CarService"},
    {"name": "GreeterService"}
  ],
  "paths": {
    "/api/v1/car": {
      "post": {"operationId": "CarService_Create", "tags": ["CarService"]}
    },
    "/api/v1/car/{vin}": {
      "get": {"operationId": "CarService_Get", "tags": ["CarService"]},
      "delete": {"operationId": "CarService_Delete", "tags": ["CarService"]}
    },
    "/v1/greeter/hello": {
      "post": {"operationId": "Greeter_Hello", "tags": ["GreeterService"]}
    }
  }
}`

func TestFilterByTags_EmptyReturnsRaw(t *testing.T) {
	out := FilterByTags([]byte(sampleSwaggerJSON), nil)
	require.Equal(t, sampleSwaggerJSON, string(out))
}

func TestFilterByTags_DropsUnmatchedOperationsAndPaths(t *testing.T) {
	out := FilterByTags([]byte(sampleSwaggerJSON), []string{"CarService"})

	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &doc))

	paths := doc["paths"].(map[string]interface{})
	require.Contains(t, paths, "/api/v1/car")
	require.Contains(t, paths, "/api/v1/car/{vin}")
	require.NotContains(t, paths, "/v1/greeter/hello", "GreeterService path should be dropped")

	tags := doc["tags"].([]interface{})
	require.Len(t, tags, 1)
	require.Equal(t, "CarService", tags[0].(map[string]interface{})["name"])
}

func TestFilterByTags_RemovesPathWhenAllOperationsDropped(t *testing.T) {
	mixed := `{
      "swagger":"2.0",
      "paths": {
        "/x": {
          "get": {"tags":["A"]},
          "post":{"tags":["B"]}
        }
      }
    }`
	out := FilterByTags([]byte(mixed), []string{"A"})
	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &doc))
	x := doc["paths"].(map[string]interface{})["/x"].(map[string]interface{})
	require.Contains(t, x, "get")
	require.NotContains(t, x, "post")
}

func TestFilterByTags_MalformedJSONReturnsRaw(t *testing.T) {
	in := []byte("not json")
	out := FilterByTags(in, []string{"X"})
	require.Equal(t, in, out)
}

func TestRegistrationServer_RoutesSpecOnlyByDefault(t *testing.T) {
	gwmux := grpc_gateway_runtime.NewServeMux()
	srv := &registrationServer{
		specs: []*Spec{{
			Name:        "carservice",
			Title:       "Car Service",
			JSON:        []byte(sampleSwaggerJSON),
			IncludeTags: []string{"CarService"},
			// UIEnabled left false: spec only.
		}},
		options: Options{UIAssetsBaseURL: DefaultUIAssetsBaseURL},
	}
	srv.RegisterFluffyCoreHandler(gwmux, nil)

	// /carservice/swagger.json -> filtered spec
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/carservice/swagger.json", nil)
	gwmux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var doc map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	require.NotContains(t, doc["paths"].(map[string]interface{}), "/v1/greeter/hello")

	// /carservice/swagger -> NOT served when UIEnabled=false
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/carservice/swagger", nil)
	gwmux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)

	// /swagger -> NOT served when no UIs are enabled
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/swagger", nil)
	gwmux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRegistrationServer_RoutesUIWhenEnabled(t *testing.T) {
	gwmux := grpc_gateway_runtime.NewServeMux()
	srv := &registrationServer{
		specs: []*Spec{
			{Name: "carservice", JSON: []byte(sampleSwaggerJSON), IncludeTags: []string{"CarService"}, UIEnabled: true},
			{Name: "greeter", JSON: []byte(sampleSwaggerJSON), IncludeTags: []string{"GreeterService"}, UIEnabled: true},
		},
		options: Options{UIAssetsBaseURL: "https://cdn.example.com/swagger-ui"},
	}
	srv.RegisterFluffyCoreHandler(gwmux, nil)

	rec := httptest.NewRecorder()
	gwmux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/carservice/swagger", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "https://cdn.example.com/swagger-ui/swagger-ui.css")
	require.Contains(t, rec.Body.String(), `"/carservice/swagger.json"`)

	// Index lists both
	rec = httptest.NewRecorder()
	gwmux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "/carservice/swagger")
	require.Contains(t, body, "/greeter/swagger")
}
