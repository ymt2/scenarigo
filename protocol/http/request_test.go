package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/zoncoen/scenarigo/context"
)

type transport struct {
	f func(*http.Request) (*http.Response, error)
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.f(req)
}

func roundTripper(f func(req *http.Request) (*http.Response, error)) http.RoundTripper {
	return &transport{f}
}

func TestRequest_Invoke(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		auth := "Bearer xxxxx"
		m := http.NewServeMux()
		m.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {})
		m.HandleFunc("/echo", func(w http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if req.Header.Get("Authorization") != auth {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			d := json.NewDecoder(req.Body)
			defer req.Body.Close()
			body := map[string]string{}
			if err := d.Decode(&body); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(fmt.Sprintf(`{"message": "%s"}`, body["message"])))
		})
		srv := httptest.NewServer(m)

		tests := map[string]struct {
			vars    interface{}
			request *Request
			result  *result
		}{
			"default": {
				request: &Request{
					URL: srv.URL,
				},
				result: &result{
					status: "200 OK",
				},
			},
			"Post": {
				request: &Request{
					Method: http.MethodPost,
					URL:    srv.URL + "/echo",
					Header: map[string][]string{"Authorization": []string{auth}},
					Body:   map[string]string{"message": "hey"},
				},
				result: &result{
					status: "200 OK",
					body:   map[string]interface{}{"message": "hey"},
				},
			},
			"with vars": {
				vars: map[string]string{
					"url":     srv.URL + "/echo",
					"auth":    auth,
					"message": "hey",
				},
				request: &Request{
					Method: http.MethodPost,
					URL:    "{{vars.url}}",
					Header: map[string][]string{"Authorization": []string{"{{vars.auth}}"}},
					Body:   map[string]string{"message": "{{vars.message}}"},
				},
				result: &result{
					status: "200 OK",
					body:   map[string]interface{}{"message": "hey"},
				},
			},
			"custom client": {
				vars: map[string]interface{}{
					"client": &http.Client{
						Transport: roundTripper(func(req *http.Request) (*http.Response, error) {
							req.Header.Set("Authorization", auth)
							return http.DefaultTransport.RoundTrip(req)
						}),
					},
				},
				request: &Request{
					Client: "{{vars.client}}",
					Method: http.MethodPost,
					URL:    srv.URL + "/echo",
					Body:   map[string]string{"message": "hey"},
				},
				result: &result{
					status: "200 OK",
					body:   map[string]interface{}{"message": "hey"},
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				ctx := context.FromT(t)
				if test.vars != nil {
					ctx = ctx.WithVars(test.vars)
				}

				ctx, res, err := test.request.Invoke(ctx)
				if err != nil {
					t.Fatalf("failed to invoke: %s", err)
				}
				if diff := cmp.Diff(test.result, res,
					cmp.AllowUnexported(
						result{},
					),
				); diff != "" {
					t.Fatalf("differs: (-want +got)\n%s", diff)
				}

				// ensure that ctx.WithRequest and ctx.WithResponse are called
				if diff := cmp.Diff(test.request.Body, ctx.Request()); diff != "" {
					t.Errorf("differs: (-want +got)\n%s", diff)
				}
				if diff := cmp.Diff(test.result.body, ctx.Response()); diff != "" {
					t.Errorf("differs: (-want +got)\n%s", diff)
				}
			})
		}
	})
	t.Run("failure", func(t *testing.T) {
		tests := map[string]struct {
			vars    interface{}
			request *Request
		}{
			"URL is required": {
				request: &Request{},
			},
			"failed to send request": {
				vars: map[string]interface{}{
					"client": &http.Client{
						Transport: roundTripper(func(req *http.Request) (*http.Response, error) {
							return nil, errors.New("error occurred")
						}),
					},
				},
				request: &Request{
					Client: "{{vars.client}}",
					URL:    "http://localhost",
				},
			},
			"failed to execute template": {
				request: &Request{
					URL: "{{vars.url}}",
				},
			},
		}
		for name, test := range tests {
			test := test
			t.Run(name, func(t *testing.T) {
				ctx := context.FromT(t)
				if test.vars != nil {
					ctx = ctx.WithVars(test.vars)
				}
				_, _, err := test.request.Invoke(ctx)
				if err == nil {
					t.Fatal("no error")
				}
			})
		}
	})
}
