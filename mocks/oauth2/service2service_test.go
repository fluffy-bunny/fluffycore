package oauth2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	testservices "github.com/fluffy-bunny/fluffycore/mocks/testservices"

	"github.com/fluffy-bunny/fluffycore/utils"
)

func TestOAuth2Server(t *testing.T) {

	for _, tc := range []struct {
		name      string
		param     string
		respCode  int
		respBody  string
		bodyFunc  testservices.MockBodyResponseFunc
		expErr    error
		expResult string
	}{
		{
			name:      "valid request",
			respCode:  http.StatusOK,
			expResult: "ok",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := NewOauth2MockServer()
			formData := url.Values{
				"client_id":     {"fluffy-micro"},
				"client_secret": {"secret"},
				"grant_type":    {"client_credentials"},
			}
			resp, err := http.PostForm(mockServer.URL+"/oauth/token", formData)
			if err != nil {
				print(err)
			}
			defer resp.Body.Close()
			var data map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&data)
			if err != nil {
				print(err)
			}
			fmt.Println(utils.PrettyJSON(data))

		})
	}
}
