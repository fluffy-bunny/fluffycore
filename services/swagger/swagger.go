// Package swagger exposes OpenAPI v2 (swagger.json) documents through the
// fluffycore gRPC-Gateway HTTP server.
//
// Each registered Spec is served at:
//
//	GET /{Spec.Name}/swagger.json   — the (optionally tag-filtered) spec
//	GET /{Spec.Name}/swagger        — the Swagger UI page (only if Spec.UIEnabled)
//
// An index page listing every registered Spec is served at GET /swagger when
// any UI is enabled.
//
// Routes are attached to the gateway ServeMux via ServeMux.HandlePath, so they
// only become available when the host application has GRPC_GATEWAY_ENABLED set.
//
// Typical wiring:
//
//	swagger.AddService(builder)
//	swagger.AddSpec(builder, &swagger.Spec{
//	    Name:        "carservice",
//	    Title:       "Car Service",
//	    JSON:        carSwaggerJSON,            // typically //go:embed
//	    IncludeTags: []string{"CarService"},    // cherry-pick from a multi-service spec
//	    UIEnabled:   true,                       // omit for spec-only deployments
//	})
//
// Tag filtering is OpenAPI v2-aware (paths[*][method].tags + top-level tags),
// matching the output of protoc-gen-openapiv2.
package swagger

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"sort"
	"strings"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_contracts_endpoint "github.com/fluffy-bunny/fluffycore/contracts/endpoint"
	grpc_gateway_runtime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	zerolog "github.com/rs/zerolog/log"
	grpc "google.golang.org/grpc"
)

// DefaultUIAssetsBaseURL points at the public swagger-ui-dist CDN. Override
// via Options.UIAssetsBaseURL when egress is restricted or when self-hosting.
const DefaultUIAssetsBaseURL = "https://unpkg.com/swagger-ui-dist@5"

// Spec describes a single OpenAPI v2 document.
type Spec struct {
	// Name is the URL slug. Required. Should be lowercase, no slashes.
	Name string
	// Title is the display name used by the UI and the index page. Falls back to Name.
	Title string
	// JSON is the raw swagger.json bytes (typically loaded via go:embed).
	JSON []byte
	// IncludeTags, when non-empty, restricts the served spec to operations
	// whose `tags` intersect this set. Top-level tags are filtered to match.
	// Empty means: serve the spec as-is.
	IncludeTags []string
	// UIEnabled enables the GET /{Name}/swagger HTML UI route. When false
	// (the default) only /{Name}/swagger.json is served — appropriate for
	// public/customer-facing deployments.
	UIEnabled bool
}

// Options configures the swagger service. All fields are optional.
type Options struct {
	// UIAssetsBaseURL is the base URL the UI HTML loads swagger-ui-dist
	// assets from. Defaults to DefaultUIAssetsBaseURL.
	UIAssetsBaseURL string
}

func (s *Spec) displayTitle() string {
	if s.Title != "" {
		return s.Title
	}
	return s.Name
}

type registrationServer struct {
	specs   []*Spec
	options Options
}

var _ fluffycore_contracts_endpoint.IEndpointRegistration = (*registrationServer)(nil)

func (s *registrationServer) RegisterFluffyCoreGRPCService(_ *grpc.Server) {}

func (s *registrationServer) RegisterFluffyCoreHandler(gwmux *grpc_gateway_runtime.ServeMux, _ *grpc.ClientConn) {
	anyUI := false
	for _, spec := range s.specs {
		jsonPath := "/" + spec.Name + "/swagger.json"
		jsonBytes := filterSwaggerByTags(spec.JSON, spec.IncludeTags)

		if err := gwmux.HandlePath(http.MethodGet, jsonPath, jsonHandler(jsonBytes)); err != nil {
			zerolog.Warn().Err(err).Str("path", jsonPath).Msg("swagger: HandlePath failed")
		}

		if spec.UIEnabled {
			anyUI = true
			uiPath := "/" + spec.Name + "/swagger"
			ui := []byte(renderSwaggerUI(spec.displayTitle(), jsonPath, s.options.UIAssetsBaseURL))
			h := htmlHandler(ui)
			if err := gwmux.HandlePath(http.MethodGet, uiPath, h); err != nil {
				zerolog.Warn().Err(err).Str("path", uiPath).Msg("swagger: HandlePath failed")
			}
			if err := gwmux.HandlePath(http.MethodGet, uiPath+"/", h); err != nil {
				zerolog.Warn().Err(err).Str("path", uiPath+"/").Msg("swagger: HandlePath failed")
			}
		}
	}

	if anyUI {
		idx := htmlHandler([]byte(renderIndex(s.specs)))
		if err := gwmux.HandlePath(http.MethodGet, "/swagger", idx); err != nil {
			zerolog.Warn().Err(err).Msg("swagger: HandlePath failed")
		}
		if err := gwmux.HandlePath(http.MethodGet, "/swagger/", idx); err != nil {
			zerolog.Warn().Err(err).Msg("swagger: HandlePath failed")
		}
	}
}

func jsonHandler(body []byte) grpc_gateway_runtime.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}

func htmlHandler(body []byte) grpc_gateway_runtime.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(body)
	}
}

// AddService registers the swagger endpoint. Call once per app. Use AddSpec
// for each spec you want exposed. Optional Options override defaults.
func AddService(builder di.ContainerBuilder, opts ...Options) {
	options := Options{UIAssetsBaseURL: DefaultUIAssetsBaseURL}
	if len(opts) > 0 {
		o := opts[0]
		if o.UIAssetsBaseURL != "" {
			options.UIAssetsBaseURL = o.UIAssetsBaseURL
		}
	}
	di.AddSingleton[fluffycore_contracts_endpoint.IEndpointRegistration](builder,
		func(specs []*Spec) fluffycore_contracts_endpoint.IEndpointRegistration {
			sorted := append([]*Spec(nil), specs...)
			sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
			return &registrationServer{specs: sorted, options: options}
		})
}

// AddSpec registers a single Spec instance. Multiple calls are allowed.
// Panics if spec is nil or spec.Name is empty.
func AddSpec(builder di.ContainerBuilder, spec *Spec) {
	if spec == nil || spec.Name == "" {
		panic("swagger: Spec.Name is required")
	}
	di.AddInstance[*Spec](builder, spec)
}

// FilterByTags returns an OpenAPI v2 document restricted to operations whose
// `tags` intersect includeTags. Top-level `tags` is filtered to match. If
// includeTags is empty the input is returned unchanged. If raw is not valid
// JSON, raw is returned unchanged. Exposed for tests and ad-hoc tooling.
func FilterByTags(raw []byte, includeTags []string) []byte {
	return filterSwaggerByTags(raw, includeTags)
}

func filterSwaggerByTags(raw []byte, includeTags []string) []byte {
	if len(includeTags) == 0 {
		return raw
	}
	keep := make(map[string]struct{}, len(includeTags))
	for _, t := range includeTags {
		keep[t] = struct{}{}
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return raw
	}

	if pathsRaw, ok := doc["paths"].(map[string]interface{}); ok {
		for path, methodsRaw := range pathsRaw {
			methods, ok := methodsRaw.(map[string]interface{})
			if !ok {
				continue
			}
			for method, opRaw := range methods {
				op, ok := opRaw.(map[string]interface{})
				if !ok {
					continue
				}
				tags, _ := op["tags"].([]interface{})
				matched := false
				for _, t := range tags {
					if ts, ok := t.(string); ok {
						if _, hit := keep[ts]; hit {
							matched = true
							break
						}
					}
				}
				if !matched {
					delete(methods, method)
				}
			}
			if len(methods) == 0 {
				delete(pathsRaw, path)
			}
		}
	}

	if tagsRaw, ok := doc["tags"].([]interface{}); ok {
		filtered := make([]interface{}, 0, len(tagsRaw))
		for _, t := range tagsRaw {
			tm, ok := t.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := tm["name"].(string)
			if _, hit := keep[name]; hit {
				filtered = append(filtered, t)
			}
		}
		doc["tags"] = filtered
	}

	out, err := json.Marshal(doc)
	if err != nil {
		return raw
	}
	return out
}

func renderSwaggerUI(title, specURL, assetsBase string) string {
	specJS, _ := json.Marshal(specURL)
	if assetsBase == "" {
		assetsBase = DefaultUIAssetsBaseURL
	}
	assetsBase = strings.TrimRight(assetsBase, "/")
	return `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <title>` + html.EscapeString(title) + `</title>
    <link rel="stylesheet" href="` + html.EscapeString(assetsBase) + `/swagger-ui.css" />
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="` + html.EscapeString(assetsBase) + `/swagger-ui-bundle.js"></script>
    <script>
      window.onload = () => {
        window.ui = SwaggerUIBundle({
          url: ` + string(specJS) + `,
          dom_id: "#swagger-ui",
        });
      };
    </script>
  </body>
</html>`
}

func renderIndex(specs []*Spec) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html><head><meta charset="UTF-8"><title>API Docs</title>`)
	b.WriteString(`<style>body{font-family:sans-serif;max-width:640px;margin:2rem auto;padding:0 1rem}li{margin:.4rem 0}</style>`)
	b.WriteString(`</head><body><h1>API Docs</h1>`)
	visible := 0
	for _, spec := range specs {
		if spec.UIEnabled {
			visible++
		}
	}
	if visible == 0 {
		b.WriteString(`<p>No swagger UIs registered.</p>`)
	} else {
		b.WriteString(`<ul>`)
		for _, spec := range specs {
			if !spec.UIEnabled {
				continue
			}
			fmt.Fprintf(&b, `<li><a href="/%s/swagger">%s</a> &mdash; <a href="/%s/swagger.json">spec</a></li>`,
				html.EscapeString(spec.Name),
				html.EscapeString(spec.displayTitle()),
				html.EscapeString(spec.Name),
			)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`</body></html>`)
	return b.String()
}
