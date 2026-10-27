package main

import (
	"mime"
	"net/http"
	"net/url"
)

// fromOwnPage lets through only requests that the chat's own page or app could have sent: JSON,
// and from no other site. A page elsewhere could otherwise post a plain form to this server,
// which a browser sends without asking, and act as any session on the hub.
func fromOwnPage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		kind, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if err != nil || kind != "application/json" {
			http.Error(response, "send JSON", http.StatusUnsupportedMediaType)
			return
		}
		if !sameOrigin(request) {
			http.Error(response, "only the chat's own page may do this", http.StatusForbidden)
			return
		}
		next.ServeHTTP(response, request)
	})
}

// sameOrigin tells whether a request comes from a page of this server, or from no page at all
// (the phone's app sends no Origin).
func sameOrigin(request *http.Request) bool {
	origin := request.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	return err == nil && parsed.Host == request.Host
}
