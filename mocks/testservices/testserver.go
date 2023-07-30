package testservices

import (
	"net/http"
	"net/http/httptest"
)

// NewMockServer return a mock HTTP server to test requests
func NewMockServer(rec MockRecorder, procedures ...MockServerProcedure) *httptest.Server {

	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			for _, proc := range procedures {
				requestURI := r.URL.RequestURI()
				if proc.URI == requestURI && proc.HTTPMethod == r.Method {

					headers := w.Header()
					for hkey, hvalue := range proc.Response.Headers {
						headers[hkey] = hvalue
					}

					code := proc.Response.StatusCode
					if proc.RequestValidator != nil {
						code = proc.RequestValidator(r)
					}

					if code == 0 {
						code = http.StatusOK
					}

					if code == http.StatusOK {
						var body []byte
						body = proc.Response.Body
						if proc.Response.BodyFunc != nil {
							body, code = proc.Response.BodyFunc(r)
						}
						w.WriteHeader(code)
						w.Write(body)
					} else {
						w.WriteHeader(code)
					}

					if rec != nil {
						rec.Record(r)
					}

					return
				}
			}

			w.WriteHeader(http.StatusNotFound)
			return
		})
	server := httptest.NewUnstartedServer(handler)
	server.Config.Addr = ":9989"
	server.Start()
	//server := httptest.NewServer(handler)
	return server
}
