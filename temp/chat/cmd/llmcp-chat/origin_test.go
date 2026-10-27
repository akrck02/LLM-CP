package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOnlyTheChatsOwnPageMayAct(test *testing.T) {
	reached := false
	handler := fromOwnPage(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))
	cases := []struct {
		name, origin, kind string
		allowed            bool
	}{
		{"its own page", "http://chat.local:8090", "application/json", true},
		{"the phone's app", "", "application/json", true},
		{"another site", "http://elsewhere.example", "application/json", false},
		{"a plain form", "", "text/plain", false},
	}
	for _, each := range cases {
		reached = false
		request := httptest.NewRequest(http.MethodPost, "http://chat.local:8090/archive", strings.NewReader("{}"))
		request.Header.Set("Content-Type", each.kind)
		if each.origin != "" {
			request.Header.Set("Origin", each.origin)
		}
		handler.ServeHTTP(httptest.NewRecorder(), request)
		if reached != each.allowed {
			test.Fatalf("%s: reached the hub %v, want %v", each.name, reached, each.allowed)
		}
	}
}
