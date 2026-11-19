package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func testLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func TestApplicationForwardsRequestAndTrustedHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if got, want := request.URL.Path, "/base/widgets"; got != want {
			t.Errorf("path=%q, want %q", got, want)
		}
		if got, want := request.URL.RawQuery, "page=2"; got != want {
			t.Errorf("query=%q, want %q", got, want)
		}
		if got, want := request.Host, "EXAMPLE.TEST"; got != want {
			t.Errorf("Host=%q, want %q", got, want)
		}
		if got, want := request.Header.Get("X-Forwarded-For"), "203.0.113.7"; got != want {
			t.Errorf("X-Forwarded-For=%q, want %q", got, want)
		}
		if got, want := request.Header.Get("X-Forwarded-Host"), "EXAMPLE.TEST"; got != want {
			t.Errorf("X-Forwarded-Host=%q, want %q", got, want)
		}
		if got, want := request.Header.Get("X-Forwarded-Proto"), "http"; got != want {
			t.Errorf("X-Forwarded-Proto=%q, want %q", got, want)
		}
		if got := request.Header.Get("Connection"); got != "" {
			t.Errorf("hop-by-hop Connection header was forwarded: %q", got)
		}
		_, _ = response.Write([]byte("upstream response"))
	}))
	defer upstream.Close()

	app, err := NewApplication(ReverseProxySettings{RateLimitWindowSeconds: 60}, map[string]string{
		"example.test": upstream.URL + "/base",
	}, testLogger())
	if err != nil {
		t.Fatal(err)
	}
	defer app.transport.(*http.Transport).CloseIdleConnections()

	request := httptest.NewRequest(http.MethodGet, "http://example.test/widgets?page=2", nil)
	request.Host = "EXAMPLE.TEST"
	request.RemoteAddr = "203.0.113.7:4567"
	request.Header.Set("Connection", "keep-alive")
	request.Header.Set("X-Forwarded-For", "untrusted-client-value")
	response := httptest.NewRecorder()

	app.ServeHTTP(response, request)

	if got, want := response.Code, http.StatusOK; got != want {
		t.Fatalf("status=%d, want %d", got, want)
	}
	if got, want := response.Body.String(), "upstream response"; got != want {
		t.Fatalf("body=%q, want %q", got, want)
	}
}

func TestApplicationUsesExactRouteBeforeHostRoute(t *testing.T) {
	defaultUpstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write([]byte("default"))
	}))
	defer defaultUpstream.Close()

	exactUpstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write([]byte(request.URL.Path + "?" + request.URL.RawQuery))
	}))
	defer exactUpstream.Close()

	app, err := NewApplication(ReverseProxySettings{RateLimitWindowSeconds: 60}, map[string]string{
		"example.test":         defaultUpstream.URL,
		"example.test/special": exactUpstream.URL,
	}, testLogger())
	if err != nil {
		t.Fatal(err)
	}
	defer app.transport.(*http.Transport).CloseIdleConnections()

	request := httptest.NewRequest(http.MethodGet, "http://example.test/special?keep=yes", nil)
	request.Host = "example.test"
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)

	if got, want := response.Code, http.StatusOK; got != want {
		t.Fatalf("status=%d, want %d", got, want)
	}
	if got, want := response.Body.String(), "/special?keep=yes"; got != want {
		t.Fatalf("body=%q, want %q", got, want)
	}
}

func TestApplicationReturnsSafeErrors(t *testing.T) {
	app, err := NewApplication(ReverseProxySettings{RateLimitWindowSeconds: 60}, map[string]string{
		"example.test": "http://127.0.0.1:1",
	}, testLogger())
	if err != nil {
		t.Fatal(err)
	}
	defer app.transport.(*http.Transport).CloseIdleConnections()

	missingRoute := httptest.NewRecorder()
	app.ServeHTTP(missingRoute, httptest.NewRequest(http.MethodGet, "http://other.test/path", nil))
	if got, want := missingRoute.Code, http.StatusNotFound; got != want {
		t.Fatalf("missing route status=%d, want %d", got, want)
	}

	failedUpstream := httptest.NewRecorder()
	app.ServeHTTP(failedUpstream, httptest.NewRequest(http.MethodGet, "http://example.test/path", nil))
	if got, want := failedUpstream.Code, http.StatusBadGateway; got != want {
		t.Fatalf("upstream failure status=%d, want %d", got, want)
	}
	if strings.Contains(failedUpstream.Body.String(), "connection refused") {
		t.Fatal("upstream connection details were exposed to the client")
	}
}

func TestApplicationRejectsInvalidRoutes(t *testing.T) {
	_, err := NewApplication(ReverseProxySettings{RateLimitWindowSeconds: 60}, map[string]string{
		"example.test": "127.0.0.1:7000",
	}, testLogger())
	if err == nil {
		t.Fatal("route without an HTTP scheme was accepted")
	}
}

func TestApplicationRateLimitsByIPNotPort(t *testing.T) {
	var upstreamRequests atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		upstreamRequests.Add(1)
		response.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	app, err := NewApplication(ReverseProxySettings{
		EnableRateLimiting:     true,
		RateLimitAfter:         1,
		RateLimitWindowSeconds: 60,
	}, map[string]string{"example.test": upstream.URL}, testLogger())
	if err != nil {
		t.Fatal(err)
	}
	defer app.transport.(*http.Transport).CloseIdleConnections()

	firstRequest := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	firstRequest.RemoteAddr = "203.0.113.7:1000"
	firstResponse := httptest.NewRecorder()
	app.ServeHTTP(firstResponse, firstRequest)

	secondRequest := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	secondRequest.RemoteAddr = "203.0.113.7:2000"
	secondResponse := httptest.NewRecorder()
	app.ServeHTTP(secondResponse, secondRequest)

	if got, want := firstResponse.Code, http.StatusNoContent; got != want {
		t.Fatalf("first status=%d, want %d", got, want)
	}
	if got, want := secondResponse.Code, http.StatusTooManyRequests; got != want {
		t.Fatalf("second status=%d, want %d", got, want)
	}
	if got := secondResponse.Header().Get("Retry-After"); got == "" {
		t.Fatal("rate-limited response is missing Retry-After")
	}
	if got, want := upstreamRequests.Load(), int32(1); got != want {
		t.Fatalf("upstream requests=%d, want %d", got, want)
	}
}
