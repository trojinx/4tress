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
type LocalDNS struct {
	Records map[string]string
}

var Dns LocalDNS
var Settings ReverseProxySettings

var IpTable = map[string]int8{}
var LogFile *os.File
var LogFileCreateError error

func healthCheck(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(200)
	res.Write([]byte("Server is Healthy!"))
	defer req.Body.Close()
}

func main() {
	ReadRecords() // reads 4tress.records.json
	go clearMap()
	ReadConfigJson() // reads 4tress.config.json
	CreateLogger()   // creates log.txt file

	http.HandleFunc("/health", healthCheck)
	http.HandleFunc("/", ForwardRequest)

	startError := http.ListenAndServe(":"+Settings.Port, nil)
	if startError != nil {
		fmt.Println("ERROR: could not start the server: ", startError)
	}
}
