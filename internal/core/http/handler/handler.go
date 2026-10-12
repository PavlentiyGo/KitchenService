package core_http_handler

import (
	core_response "foodDelivery/internal/core/http/response"
	"net/http"
)

type HandlerFunc func(w *core_response.Writer, r *http.Request)

func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h(core_response.WrapWriter(w, r.Context()), r)
}
