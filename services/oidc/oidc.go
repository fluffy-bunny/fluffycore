package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	fluffycore_contracts_oidc "github.com/fluffy-bunny/fluffycore/contracts/oidc"

	jwxk "github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type (
	service struct {
		Authority         string
		discoveryDocument *DiscoveryDocument
	}
)

func New(ctx context.Context, authority string) (fluffycore_contracts_oidc.IOpenIdConnectConfiguration, error) {
	log := zerolog.Ctx(ctx).With().Logger()
	discoveryDocument, err := NewDiscoveryDocument(&DiscoveryDocumentOptions{
		Authority: authority,
	})
	if err != nil {
		log.Error().Err(err).Msg("error creating discovery document")
		return nil, err
	}
	obj := &service{
		Authority:         authority,
		discoveryDocument: discoveryDocument,
	}
	err = obj.discoveryDocument.Initialize()
	if err != nil {
		log.Error().Err(err).Msg("error initializing discovery document")
		return nil, err
	}
	return obj, nil
}
func (s *service) Refresh(ctx context.Context) error {
	log := zerolog.Ctx(ctx).With().Logger()
	_, err := s.discoveryDocument.Refresh(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error refreshing discovery document")
	}
	return err
}

func (s *service) GetDiscoveryDocument(ctx context.Context) (*fluffycore_contracts_oidc.DiscoveryDocument, error) {
	log := zerolog.Ctx(ctx).With().Logger()
	log.Debug().Msg("GetDiscoveryDocument")
	_, err := s.discoveryDocument.Refresh(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error refreshing discovery document")
		return nil, err
	}
	return &s.discoveryDocument.DiscoveryDocument, nil
}

type OAuth2DiscoveryOptions struct {
	JwksURI string
}
type OAuth2Document struct {
	Options      *OAuth2DiscoveryOptions
	Issuer       string `json:"issuer"`
	JWKSURL      string `json:"jwks_uri"`
	jwksCache    *jwxk.Cache
	jwksCancelAR context.CancelFunc
}
type DiscoveryDocumentOptions struct {
	Authority              string
	OAuth2DiscoveryOptions OAuth2DiscoveryOptions
}
type DiscoveryDocument struct {
	fluffycore_contracts_oidc.DiscoveryDocument
	OAuth2Document *OAuth2Document
	Options        *DiscoveryDocumentOptions
	DiscoveryURL   url.URL
}

func NewOAuth2Document(options *OAuth2DiscoveryOptions) (*OAuth2Document, error) {
	if options == nil {
		log.Fatal().Msg("options cannot be nil")
		panic("options cannot be nil")
	}

	return &OAuth2Document{
		Options: options,
	}, nil
}
func NewDiscoveryDocument(options *DiscoveryDocumentOptions) (*DiscoveryDocument, error) {
	if options == nil {
		log.Fatal().Msg("options cannot be nil")
		panic("options cannot be nil")
	}
	u, err := url.Parse(options.Authority)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, "/.well-known/openid-configuration")

	return &DiscoveryDocument{
		Options:      options,
		DiscoveryURL: *u,
	}, nil
}
func (document *DiscoveryDocument) FetchJwks(ctx context.Context) (jwxk.Set, error) {
	return document.OAuth2Document.FetchJwks(ctx)
}

func (document *OAuth2Document) FetchJwks(ctx context.Context) (jwxk.Set, error) {
	return document.jwksCache.Get(ctx, document.JWKSURL)
}

func (document *DiscoveryDocument) Refresh(ctx context.Context) (jwxk.Set, error) {
	return document.OAuth2Document.Refresh(ctx)
}

func (document *OAuth2Document) Refresh(ctx context.Context) (jwxk.Set, error) {
	return document.jwksCache.Refresh(ctx, document.JWKSURL)
}

func (document *OAuth2Document) Initialize() error {

	var ctx context.Context
	ctx, document.jwksCancelAR = context.WithCancel(context.Background())
	document.jwksCache = jwxk.NewCache(ctx)
	document.jwksCache.Register(document.Options.JwksURI, jwxk.WithMinRefreshInterval(time.Minute*5))

	document.JWKSURL = document.Options.JwksURI

	_, err := document.jwksCache.Refresh(ctx, document.JWKSURL)
	if err != nil {
		log.Error().Err(err).
			Str("uri", document.JWKSURL).
			Msg("Initial fetch of JWKS - will try again in the background and when a request is received")
		return err
	}
	jwkSet, err := document.jwksCache.Get(ctx, document.JWKSURL)
	if err != nil {
		log.Error().Err(err).Str("jwks", document.JWKSURL).Msg("Fetching JWKS at auth time")
		return err
	}
	log.Debug().Int("keys", jwkSet.Len())
	return nil
}

func (document *DiscoveryDocument) Initialize() error {
	err := document.LoadDiscoveryDocument()
	if err != nil {
		return fmt.Errorf("error loading discovery document: %w", err)
	}
	document.Options.OAuth2DiscoveryOptions.JwksURI = document.JwksURI
	document.OAuth2Document, err = NewOAuth2Document(&(document.Options.OAuth2DiscoveryOptions))
	if err != nil {
		return fmt.Errorf("error newOAuth2Document: %w", err)
	}
	err = document.OAuth2Document.Initialize()
	if err != nil {
		return fmt.Errorf("error initializing OAuth2Document: %w", err)
	}
	return nil
}

func (document *DiscoveryDocument) LoadDiscoveryDocument() error {
	resp, err := http.Get(document.DiscoveryURL.String())
	if err != nil {
		return fmt.Errorf("could not fetch discovery url: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	err = json.NewDecoder(resp.Body).Decode(document)
	if err != nil {
		return fmt.Errorf("error decoding discovery document: %w", err)
	}

	return nil
}
