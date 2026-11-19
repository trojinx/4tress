package main

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

//go:embed public/tooManyRequests.html
var tooManyRequestsPage []byte

type route struct {
	target *url.URL
	proxy  *httputil.ReverseProxy
}

// Application is the HTTP handler for 4tress.
type Application struct {
	routes             map[string]route
	rateLimitingActive bool
	limiter            *RateLimiter
	logger             *log.Logger
	transport          http.RoundTripper
}

func NewApplication(settings ReverseProxySettings, records map[string]string, logger *log.Logger) (*Application, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("Records must contain at least one route")
	}
	if settings.RateLimitWindowSeconds == 0 {
		settings.RateLimitWindowSeconds = 60
	}
	if settings.RateLimitWindowSeconds < 1 {
		return nil, fmt.Errorf("RateLimitWindowSeconds must be positive")
	}
	if settings.EnableRateLimiting && settings.RateLimitAfter < 1 {
		return nil, fmt.Errorf("RateLimitAfter must be positive when rate limiting is enabled")
	}
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	application := &Application{
		routes:             make(map[string]route, len(records)),
		rateLimitingActive: settings.EnableRateLimiting,
		limiter:            NewRateLimiter(settings.RateLimitAfter, rateLimitWindow(settings)),
		logger:             logger,
		transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}

	for configuredRoute, rawTarget := range records {
		key, err := normalizeRouteKey(configuredRoute)
		if err != nil {
			return nil, err
		}
		target, err := parseTarget(rawTarget)
		if err != nil {
			return nil, fmt.Errorf("route %q: %w", configuredRoute, err)
		}
		if _, exists := application.routes[key]; exists {
			return nil, fmt.Errorf("duplicate route %q", configuredRoute)
		}
		application.routes[key] = route{target: target}
	}

	for key, configuredRoute := range application.routes {
		configuredRoute.proxy = application.newReverseProxy(configuredRoute.target)
		application.routes[key] = configuredRoute
	}

	return application, nil
}

func (application *Application) Close() {
	if transport, ok := application.transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}

func (application *Application) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/health" {
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("Server is Healthy!"))
		return
	}

	if application.rateLimitingActive {
		clientAddress := clientIP(request.RemoteAddr)
		allowed, retryAfter := application.limiter.Allow(clientAddress)
		if !allowed {
			application.logger.Printf("rate limited client=%s host=%s path=%s", clientAddress, request.Host, request.URL.Path)
			writeRateLimited(response, retryAfter)
			return
		}
	}

	key := normalizedHost(request.Host) + request.URL.EscapedPath()
	configuredRoute, found := application.routes[key]
	if !found {
		configuredRoute, found = application.routes[normalizedHost(request.Host)]
	}
	if !found {
		application.logger.Printf("no route for host=%q path=%q", request.Host, request.URL.Path)
		http.NotFound(response, request)
		return
	}

	configuredRoute.proxy.ServeHTTP(response, request)
}

func (application *Application) newReverseProxy(target *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(proxyRequest *httputil.ProxyRequest) {
			proxyRequest.SetURL(target)
			// SetURL sets the upstream URL but clears Host. Preserve the virtual host
			// requested by the client for backends that use host-based routing.
			proxyRequest.Out.Host = proxyRequest.In.Host
			proxyRequest.Out.Header.Set("X-Forwarded-For", clientIP(proxyRequest.In.RemoteAddr))
			proxyRequest.Out.Header.Set("X-Forwarded-Host", proxyRequest.In.Host)
			proxyRequest.Out.Header.Set("X-Forwarded-Proto", requestScheme(proxyRequest.In))
		},
		Transport:     application.transport,
		FlushInterval: -1,
		ErrorHandler: func(response http.ResponseWriter, request *http.Request, err error) {
			application.logger.Printf("upstream request failed host=%q path=%q error=%v", request.Host, request.URL.Path, err)
			http.Error(response, "Bad Gateway", http.StatusBadGateway)
		},
	}
}

func writeRateLimited(response http.ResponseWriter, retryAfter time.Duration) {
	seconds := int(math.Ceil(retryAfter.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Retry-After", fmt.Sprintf("%d", seconds))
	response.WriteHeader(http.StatusTooManyRequests)
	_, _ = response.Write(tooManyRequestsPage)
}

func parseTarget(rawTarget string) (*url.URL, error) {
	target, err := url.Parse(rawTarget)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream URL: %w", err)
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, fmt.Errorf("upstream URL must use http or https")
	}
	if target.Host == "" {
		return nil, fmt.Errorf("upstream URL must include a host")
	}
	if target.User != nil {
		return nil, fmt.Errorf("upstream URL must not include credentials")
	}
	if target.Fragment != "" {
		return nil, fmt.Errorf("upstream URL must not include a fragment")
	}
	return target, nil
}

func normalizeRouteKey(configuredRoute string) (string, error) {
	configuredRoute = strings.TrimSpace(configuredRoute)
	if configuredRoute == "" {
		return "", fmt.Errorf("route key must not be empty")
	}
	if strings.ContainsAny(configuredRoute, " \t\r\n?") {
		return "", fmt.Errorf("invalid route key %q", configuredRoute)
	}

	slash := strings.IndexByte(configuredRoute, '/')
	if slash == -1 {
		return normalizedHost(configuredRoute), nil
	}

	host, path := configuredRoute[:slash], configuredRoute[slash:]
	if host == "" || strings.Contains(host, "://") {
		return "", fmt.Errorf("invalid route key %q", configuredRoute)
	}
	return normalizedHost(host) + path, nil
}

func normalizedHost(host string) string {
	return strings.ToLower(strings.TrimSpace(host))
}

func clientIP(remoteAddress string) string {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		return host
	}
	return remoteAddress
}

func requestScheme(request *http.Request) string {
	if request.TLS != nil {
		return "https"
	}
	return "http"
}
