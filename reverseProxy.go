package main

import (
	"fmt"
	"io"
	"net/http"
)

func ForwardRequest(res http.ResponseWriter, req *http.Request) {
	if RateLimit(req.RemoteAddr) && Settings.EnableRateLimiting {
		res.Header().Set("Retry-After", "60")
		res.Header().Set("Connection", "close")
		go Log(req.RemoteAddr + " is rate limited")

		res.WriteHeader(http.StatusTooManyRequests)
		http.ServeFile(res, req, "public/index.html")
		return
	}

	// check rate limit before this
	connection := req.Header.Get("Connection")
	upgrade := req.Header.Get("Upgrade")

	client := Client{
		IpAddr:   req.RemoteAddr,
		Protocol: req.Proto,
		Host:     req.Host,
	}

	reverseProxy := ReverseProxy{
		forwardURL:      RoutingTable[req.Host],
		forwardEndpoint: req.URL.String(),
	}
	reverseProxy.completeURL = reverseProxy.forwardURL + reverseProxy.forwardEndpoint
	//create a new http request instead of reusing the old one, is this causing overhead?????
	forwardReq, reqError := http.NewRequest(req.Method, reverseProxy.completeURL, req.Body)
	if reqError != nil {
		fmt.Println("ERROR: could not create a new HTTP request: ", reqError)
		return
	}
	reverseProxyClient := http.DefaultClient
	forwardReq.Header.Add("Host", client.Host)
	forwardReq.Header.Add("X-Forwarded-For", client.IpAddr)     // original client's IP addr
	forwardReq.Header.Add("X-Forwarded-Proto", client.Protocol) // client's used HTTP protocol & version
	// Adding any other custom headers (if any)
	for key, val := range req.Header {
		for _, v := range val {
			forwardReq.Header.Add(key, v)
		}
	}

	if connection == "websocket" && upgrade == "websocket" {
		// this is websocket, handle separately
		res.Write([]byte("Web socket"))
		defer req.Body.Close()
	}

	if connection == "close" || connection == "" || connection == "keep-alive" {

		originServerRes, sendErr := reverseProxyClient.Do(forwardReq) // forward the req
		if sendErr != nil {
			fmt.Println("ERROR: could not send req: ", sendErr)
			return
		}
		originServerResHeaderMap := originServerRes.Header
		for key, val := range originServerResHeaderMap {
			for _, v := range val {
				res.Header().Add(key, v)
			}
		}
		if originServerRes.Header.Get("Content-Type") == "text/event-stream" {
			// this is sse, handle it:
			buff := make([]byte, 4096)
			for {
				n, err := originServerRes.Body.Read(buff)
				if err != nil {
					fmt.Println("could not read SSE body into buffer")
					return
				}
				if n > 0 {
					_, writeErr := res.Write(buff[0:n])
					if writeErr != nil {
						fmt.Println("ERROR: could not write buffer to response: ", writeErr)
					}
					res.(http.Flusher).Flush()
				}
			}
		}

		res.WriteHeader(originServerRes.StatusCode)
		buff := make([]byte, 4096)
		_, copyErr := io.CopyBuffer(res, originServerRes.Body, buff)

		if copyErr != nil {
			fmt.Println("ERROR: could not copy response body from origin server to a buffer: ", copyErr)
			res.WriteHeader(500)
			res.Write([]byte("Internal server error"))
			return
		}
	}

}
