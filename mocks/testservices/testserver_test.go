package testservices

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/fluffy-bunny/fluffycore/utils"
)

func TestFooClientDoStuff(t *testing.T) {
	path := "/path"

	for _, tc := range []struct {
		name      string
		param     string
		respCode  int
		respBody  string
		bodyFunc  MockBodyResponseFunc
		expErr    error
		expResult string
	}{
		{
			name:     "upstream failure",
			respCode: http.StatusInternalServerError,
			expErr:   fmt.Errorf("upstream error"),
		},
		{
			name:     "valid response to bar",
			param:    "bar",
			respCode: http.StatusOK,
			respBody: `{"result":"ok"}`,
			bodyFunc: func(r *http.Request) ([]byte, int) {
				return []byte(`{"result":"ok"}`), http.StatusOK
			},
			expResult: "ok",
		},
		{
			name:      "valid response to baz",
			param:     "baz",
			respCode:  http.StatusOK,
			respBody:  `{"result":"also ok"}`,
			expResult: "also ok",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockServer := NewMockServer(nil, MockServerProcedure{
				URI:        path,
				HTTPMethod: http.MethodPost,
				Response: MockResponse{
					StatusCode: tc.respCode,
					Body:       []byte(tc.respBody),
					BodyFunc:   tc.bodyFunc,
				},
			})
			formData := url.Values{
				"grant_type": {"refresh_token"},
			}
			resp, err := http.PostForm(mockServer.URL+"/path", formData)
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
