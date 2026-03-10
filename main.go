package main

import (
	"fmt"
	"net/http"
	"os"
)

type ReverseProxy struct {
	forwardURL      string
	forwardEndpoint string
	completeURL     string
}

type Client struct { //NOTE: only those fields are used that are required to set as headers by reverse proxy
	IpAddr   string
	Protocol string // http or https
	Host     string // this is the requested domain
}
type ReverseProxySettings struct {
	Port               string
	EnableRateLimiting bool
	EnableWebFireWall  bool
	RateLimitAfter     int
}

var Settings ReverseProxySettings

var RoutingTable = map[string]string{}
var IpTable = map[string]int8{}
var LogFile *os.File
var LogFileCreateError error

func healthCheck(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(200)
	res.Write([]byte("Server is Healthy!"))
	defer req.Body.Close()
}

func main() {
	ReadConfigJson()
	CreateLogger()
	ReadConfig("4tress.config", &RoutingTable)

	http.HandleFunc("/health", healthCheck)
	http.HandleFunc("/", ForwardRequest)

	startError := http.ListenAndServe(":"+Settings.Port, nil)
	if startError != nil {
		fmt.Println("ERROR: could not start the server: ", startError)
	}
}
