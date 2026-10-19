package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	app "github.com/maxence-charriere/go-app/v10/pkg/app"
	zerolog "github.com/rs/zerolog"
)

// DTO-in structure for the API call.
type CallInput struct {
	Method      string `json:"method"`
	Url         string `json:"url"`
	CallerID    string `json:"caller_id"`
	PageNo      int    `json:"page_no"`
	HideReplies bool   `json:"hide_replies"`

	// CustomHeaders allows adding additional headers to the request
	// Example: map[string]string{"X-Csrf-Token": "token123", "X-Custom-Header": "value"}
	CustomHeaders map[string]string `json:"custom_headers"`

	// Payload body for the API call.
	Data interface{} `json:"data"`
}

// Standardized common response from API.
type Response struct {
	Code      int    `json:"code"`
	Error     error  `json:"-"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`

	// Custom Data field per the app component. Must be referenced via the inner pointer too when the output is required.
	//
	//   Example:
	//   output := &common.Response{&model.User}
	//
	Data interface{} `json:"data"`
}

// RequestInit object. Usable map for export to JSValue via app.ValueOf(x)
var DefaultRequestInit = map[string]interface{}{
	// Serialized JSON string.
	"body": nil,
	// Caching options: default, no-store, reload, no-cache, force-cache, only-if-cached.
	"cache": "default",
	// Credential control options: omit, same-site, include.
	// Use "include" to send cookies with cross-origin requests
	"credentials":    "include",
	"headers":        map[string]interface{}{},
	"keepalive":      false,
	"method":         "GET",
	"mode":           "cors",
	"priority":       "auto",
	"redirect":       "follow",
	"referrer":       "about:client",
	"referrerPolicy": "",
	"signal":         nil,
	"url":            "",
}

// buildRequestInit returns a fresh RequestInit for input, starting from a copy of
// DefaultRequestInit. The helpers used to write into DefaultRequestInit itself: every call
// mutated shared global state (concurrent calls raced), and fields like callerID stuck
// around for later calls that never set them.
func buildRequestInit(input *CallInput) (map[string]interface{}, error) {
	init := make(map[string]interface{}, len(DefaultRequestInit)+1)
	for k, v := range DefaultRequestInit {
		init[k] = v
	}

	headers := map[string]interface{}{
		"Content-Type": "application/json",
	}
	for key, value := range input.CustomHeaders {
		headers[key] = value
	}
	init["headers"] = headers

	if input.CallerID != "" {
		init["callerID"] = input.CallerID
	}
	if input.Method != "" {
		init["method"] = input.Method
	}
	init["url"] = input.Url

	// Convert body into a string.
	init["body"] = nil
	if input.Data != nil && init["method"] != "GET" && init["method"] != "HEAD" {
		jsonData, err := json.Marshal(input.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		init["body"] = string(jsonData)
	}
	return init, nil
}

// FetchData is a metafunction for input options conversion for the main, lighter Fetch() function.
func FetchData(ctx context.Context, input *CallInput, output *Response) bool {
	log := zerolog.Ctx(ctx).With().
		Str("component", "common.FetchData").
		Interface("input", input).
		Logger()
	log.Debug().Msg("FetchData called")

	init, err := buildRequestInit(input)
	if err != nil {
		log.Error().Err(err).Msg("FetchData: build request")
		output.Error = err
		return false
	}

	// Set the abort controller signal callback.
	/*var aController = app.Window().Get("AbortController").New()
	app.Window().Set("fetchController", aController)
	init["signal"] = aController.Get("signal")*/

	// Call the ligher fetch wrapper.
	out, code, err := Fetch(&init)
	if err != nil {
		log.Error().Err(err).Msg("FetchData: fetch error")
		output.Error = err
		return false
	}
	log.Debug().Str("responseBody", *out).Int("responseCode", code).Msg("FetchData: fetch successful")

	// Read again and associate fields.
	r := strings.NewReader(*out)

	// Unmarshal the raw string to []byte stream JSON to the interface map.
	if err := json.NewDecoder(r).Decode(&output); err != nil {
		log.Error().Err(err).Msg("FetchData: decode error")
		return false
	}
	log.Debug().Interface("output", output).Msg("FetchData: output")

	// Assign the HTTP response code.
	output.Code = code

	return true
}

type WrappedResonseT[T any] struct {
	Response *T  `json:"response"`
	Code     int `json:"code"`
}

// FetchDataT is a generic version of FetchData that directly unmarshals to the specified type T.
// Returns the data, HTTP status code, and any error encountered.
//
// Example usage:
//
//	cfg, code, err := common.FetchDataT[config.Config](&common.CallInput{
//	    Method: "GET",
//	    Url:    "/api/config",
//	})
func FetchDataT[T any](ctx context.Context, input *CallInput) (*T, int, error) {

	log := zerolog.Ctx(ctx).With().
		Str("component", "common.FetchDataT").
		Interface("input", input).
		Logger()

	init, err := buildRequestInit(input)
	if err != nil {
		return nil, 0, err
	}

	// Call the lighter fetch wrapper.
	out, code, err := Fetch(&init)
	if err != nil {
		log.Error().Err(err).Msg("FetchDataT: fetch error")
		return nil, code, err
	}

	log.Debug().Str("responseBody", *out).Int("responseCode", code).Msg("FetchDataT: fetch successful")

	// Unmarshal directly to the generic type T
	var result T
	r := strings.NewReader(*out)
	if err := json.NewDecoder(r).Decode(&result); err != nil {
		log.Error().Err(err).Msg("FetchDataT: decode error")
		return nil, code, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Debug().Interface("result", result).Msg("FetchDataT: decoded result")

	return &result, code, nil
}

func FetchWrappedResponseT[T any](ctx context.Context, input *CallInput) (*WrappedResonseT[T], error) {
	responseData, code, err := FetchDataT[T](ctx, input)
	if err != nil {
		return nil, err
	}
	return &WrappedResonseT[T]{
		Response: responseData,
		Code:     code,
	}, nil
}

// Fetch is a raw implementation of http.Client to omit the `net/*` packages completely. The main purpose is to further optimize the disk and memory space needed by the WASM app client.
func Fetch(input *map[string]interface{}) (*string, int, error) {
	if (*input)["url"] == "" {
		return nil, 0, fmt.Errorf("url not specified for fetch")
	}

	type result struct {
		body string
		code int
		err  error
	}
	done := make(chan result, 1) // exactly one of the callbacks below sends

	// Every app.FuncOf allocates a JS function that lives until Release; they used to be
	// created per call and never released. All four are released once the call settles.
	var status int
	onJSON := app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		done <- result{body: app.Window().Get("JSON").Call("stringify", args[0]).String(), code: status}
		return nil
	})
	onJSONError := app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		// The body wasn't JSON (e.g. an HTML error page or an empty 204). Keep the real
		// status; it used to be replaced with 500.
		done <- result{code: status, err: fmt.Errorf("%s", args[0].Get("message").String())}
		return nil
	})
	onResponse := app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		response := args[0]
		status = response.Get("status").Int()
		// --> fetch(url).then(response => response.json())
		response.Call("json").Call("then", onJSON).Call("catch", onJSONError)
		return nil
	})
	onFetchError := app.FuncOf(func(this app.Value, args []app.Value) interface{} {
		done <- result{err: fmt.Errorf("%s", args[0].Get("message").String())}
		return nil
	})
	defer func() {
		onJSON.Release()
		onJSONError.Release()
		onResponse.Release()
		onFetchError.Release()
	}()

	app.Window().Call("fetch", (*input)["url"], *input).
		Call("then", onResponse).
		Call("catch", onFetchError)

	r := <-done
	return &r.body, r.code, r.err
}
