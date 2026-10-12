package core_server

import (
	"fmt"
	http_middleware "foodDelivery/internal/core/http/middleware"
	"net/http"
)

type ApiRouterVersion string

const (
	Version1 ApiRouterVersion = "v1"
	Version2 ApiRouterVersion = "v2"
)

type Router struct {
	mux         *http.ServeMux
	version     ApiRouterVersion
	middlewares []http_middleware.Middleware
}

func NewApiVersionRouter(version ApiRouterVersion, middlewares ...http_middleware.Middleware) *Router {
	return &Router{
		mux:         http.NewServeMux(),
		version:     version,
		middlewares: middlewares,
	}
}

func (r *Router) ChainRoutes(routes ...Route) {
	for _, route := range routes {

		pattern := fmt.Sprintf("%s %s", route.Method, route.Path)
		handler := http_middleware.ChainMiddleware(route.Handler, route.Middlewares...)

		r.mux.Handle(pattern, handler)
	}
}
