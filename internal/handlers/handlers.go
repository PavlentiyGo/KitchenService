package handlers

import (
	core_server "foodDelivery/internal/core/http/server"
)

type Handler interface {
	Routes() []core_server.Route
}

func GetAllRoutes(handlers ...Handler) []core_server.Route {
	var routes []core_server.Route

	for _, handler := range handlers {
		routes = append(routes, handler.Routes()...)
	}
	return routes
}
