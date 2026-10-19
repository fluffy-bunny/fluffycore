package introspection

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// DefaultTimeout bounds an introspection call when no HTTPClient is supplied.
const DefaultTimeout = 10 * time.Second

// maxResponseBytes caps how much of the introspection response is read.
const maxResponseBytes = 1 << 20

// ValidateToken returns nil only if the introspection endpoint reports the token as active.
//
// Per RFC 7662 an introspection endpoint answers 200 OK for invalid, expired and revoked
// tokens too -- with {"active": false}. This previously checked only the HTTP status, so
// every such token was accepted.
func ValidateToken(options IntrospectionValidationOptions) error {
	_, err := ValidateTokenWithContext(context.Background(), options)
	return err
}

// ValidateTokenWithContext introspects options.Token (RFC 7662) and returns the response
// when the token is active. Any transport failure, non-200 status, unparseable body, or
// inactive token is an error.
func ValidateTokenWithContext(ctx context.Context, options IntrospectionValidationOptions) (*IntrospectionResponse, error) {
	logger := options.Logger
	if logger == nil {
		logger = log.New()
		logger.SetOutput(io.Discard)
	}

	form := url.Values{}
	form.Set("token", options.Token)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, options.IntrospectionURL, strings.NewReader(form.Encode()))
	if err != nil {
		logger.WithError(err).Info("Error creating introspection request")
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	// RFC 6749 §2.3.1: client_secret_basic, which authorization servers MUST support.
	// Credentials are form-urlencoded before being base64-encoded, as that section requires.
	request.SetBasicAuth(url.QueryEscape(options.ClientID), url.QueryEscape(options.ClientSecret))

	client := options.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: DefaultTimeout}
	}
	response, err := client.Do(request)
	if err != nil {
		logger.WithError(err).Info("Error doing introspection request")
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		logger.WithFields(log.Fields{
			"statusCode": response.StatusCode,
			"status":     response.Status,
		}).Info("Invalid introspection response")
		return nil, fmt.Errorf("error validating token: introspection endpoint returned %d", response.StatusCode)
	}

	var result struct {
		IntrospectionResponse
		// A pointer so a response that omits "active" entirely is rejected rather than
		// read as the zero value.
		Active *bool `json:"active"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(&result); err != nil {
		logger.WithError(err).Info("Unparseable introspection response")
		return nil, fmt.Errorf("error validating token: unparseable introspection response: %w", err)
	}
	if result.Active == nil {
		return nil, fmt.Errorf("error validating token: introspection response has no \"active\" member")
	}
	if !*result.Active {
		logger.Info("Token is not active")
		return nil, fmt.Errorf("error validating token: token is not active")
	}
	resp := result.IntrospectionResponse
	resp.Active = true
	return &resp, nil
}
