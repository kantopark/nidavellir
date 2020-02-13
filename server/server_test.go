package server_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi"
)

func NewTestRequest(method, target string, body io.Reader, routeParams map[string]string) *http.Request {
	r := httptest.NewRequest(method, target, body)
	r.SetBasicAuth("user", "password")

	if len(routeParams) > 0 {
		routeCtx := chi.NewRouteContext()
		for key, value := range routeParams {
			routeCtx.URLParams.Add(key, value)
		}

		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
	}

	return r
}

func readJson(w *httptest.ResponseRecorder, object interface{}) error {
	return json.NewDecoder(w.Body).Decode(object)
}
