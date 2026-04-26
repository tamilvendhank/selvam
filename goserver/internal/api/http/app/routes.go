package app

import "net/http"

type Router interface {
	Handle(pattern string, handler http.Handler)
}

func RegisterRoutes(router Router, handler http.Handler) {
	router.Handle("/api/v1/companies", handler)
	router.Handle("/api/v1/companies/", handler)
	router.Handle("/api/v1/reviews", handler)
	router.Handle("/api/v1/reviews/", handler)
	router.Handle("/api/v1/theses", handler)
	router.Handle("/api/v1/theses/", handler)
	router.Handle("/api/v1/capital-allocations", handler)
	router.Handle("/api/v1/capital-allocations/", handler)
	router.Handle("/api/v1/workflow-runs/", handler)
	router.Handle("/api/v1/positions", handler)
	router.Handle("/api/v1/positions/", handler)
}
