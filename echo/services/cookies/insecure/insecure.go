package secure

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	contracts_cookies "github.com/fluffy-bunny/fluffycore/echo/contracts/cookies"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	status "github.com/gogo/status"
	echo "github.com/labstack/echo/v4"
	codes "google.golang.org/grpc/codes"
)

type (
	service struct {
	}
)

var stemService = (*service)(nil)

func init() {
	var _ contracts_cookies.ICookies = (*service)(nil)
}

func (s *service) Ctor() (contracts_cookies.ICookies, error) {

	return &service{}, nil
}

func AddCookies(b di.ContainerBuilder) {
	di.AddSingleton[contracts_cookies.ICookies](b, stemService.Ctor)
}

func (s *service) validateSetCookieRequest(request *contracts_cookies.SetCookieRequest) error {
	if fluffycore_utils.IsEmptyOrNil(request.Name) {
		return status.Error(codes.InvalidArgument, "Name is required")
	}
	if fluffycore_utils.IsEmptyOrNil(request.Path) {
		return status.Error(codes.InvalidArgument, "Path is required")
	}
	return nil
}
func (s *service) SetCookie(c echo.Context, request *contracts_cookies.SetCookieRequest) (*contracts_cookies.SetCookieResponse, error) {
	err := s.validateSetCookieRequest(request)
	if err != nil {
		return nil, err
	}

	r := c.Request()

	isTLS := r.TLS != nil
	if request.Secure != nil {
		// override
		isTLS = *request.Secure
	}
	cookieData, err := json.Marshal(request.Value)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	// url encode the cookie value

	encoded := base64.URLEncoding.EncodeToString([]byte(cookieData))

	cookie := &http.Cookie{
		Name:     request.Name,
		Value:    encoded,
		Path:     request.Path,
		Secure:   isTLS,
		HttpOnly: request.HttpOnly,
		Expires:  request.Expires,
		MaxAge:   request.MaxAge,
		Domain:   request.Domain,
		SameSite: request.SameSite,
	}
	c.SetCookie(cookie)

	return &contracts_cookies.SetCookieResponse{}, nil
}

func (s *service) GetCookie(c echo.Context, name string) (*contracts_cookies.GetCookieResponse, error) {
	if fluffycore_utils.IsEmptyOrNil(name) {
		return nil, status.Error(codes.InvalidArgument, "Name is required")
	}
	value := make(map[string]interface{})
	cookie, err := c.Cookie(name)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	decoded, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	err = json.Unmarshal(decoded, &value)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &contracts_cookies.GetCookieResponse{
		Value: value,
	}, nil
}
func (s *service) validateDeleteCookieRequest(request *contracts_cookies.DeleteCookieRequest) error {
	if fluffycore_utils.IsEmptyOrNil(request.Name) {
		return status.Error(codes.InvalidArgument, "Name is required")
	}
	if fluffycore_utils.IsEmptyOrNil(request.Path) {
		return status.Error(codes.InvalidArgument, "Path is required")
	}
	if fluffycore_utils.IsEmptyOrNil(request.Domain) {
		return status.Error(codes.InvalidArgument, "Domain is required")
	}
	return nil
}
func (s *service) DeleteCookie(c echo.Context, request *contracts_cookies.DeleteCookieRequest) error {
	err := s.validateDeleteCookieRequest(request)
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:   request.Name,
		Value:  "",
		Path:   request.Path,
		MaxAge: -1,
		Domain: request.Domain,
	}
	c.SetCookie(cookie)
	return nil
}
