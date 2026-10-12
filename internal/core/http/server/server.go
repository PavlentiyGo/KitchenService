package core_server

import (
	core_config "foodDelivery/internal/core/config"
	core_http_handler "foodDelivery/internal/core/http/handler"
	http_middleware "foodDelivery/internal/core/http/middleware"
	"net/http"

	http_swagger "github.com/swaggo/http-swagger/v2"
)

type Route struct {
	Method      string
	Path        string
	Handler     core_http_handler.HandlerFunc
	Middlewares []http_middleware.Middleware
}

func RegisterRouters(routers ...*Router) *http.ServeMux {
	mux := http.NewServeMux()
	for _, router := range routers {
		prefix := "/api/" + string(router.version)

		handler := http_middleware.ChainMiddleware(router.mux, router.middlewares...)

		mux.Handle(
			prefix+"/",
			http.StripPrefix(prefix, handler),
		)

	}
	return mux
}

func NewServer(config *core_config.ServerConfig, routers []*Router, middlewares ...http_middleware.Middleware) *http.Server {

	mux := RegisterRouters(routers...)

	handler := http_middleware.ChainMiddleware(mux, middlewares...)

	mux.Handle("/swagger/", http_swagger.Handler(http_swagger.URL("/swagger/doc.json")))
	server := &http.Server{
		Addr:              config.Addr,
		Handler:           handler,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
	}
	return server
}
