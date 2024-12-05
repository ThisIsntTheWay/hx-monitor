package main

import "net/http"

func init() {
	muxRouter.Use(corsMiddleware)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ToDo: Remove All origins
		allowOrigin := "*"
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)

		if r.Method == http.MethodOptions {
			return
		}

		next.ServeHTTP(w, r)
	})
}
