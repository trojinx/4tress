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

var RoutingTable = map[string]string{}
var IpTable = map[string]int8{}
var LogFile *os.File
var LogFileCreateError error
var Created bool

func healthCheck(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(200)
	res.Write([]byte("Server is Healthy!"))
	defer req.Body.Close()
}
func rateLimitCheck(res http.ResponseWriter, req *http.Request) {
	isRateLimit := RateLimit(req.RemoteAddr)
	if isRateLimit == true {
		res.WriteHeader(429)
		res.Write([]byte("you have been rate limited" + req.RemoteAddr))
	} else {
		res.WriteHeader(200)
		res.Write([]byte("You have not been rate limited" + req.RemoteAddr))
	}
}

func main() {
	Created, LogFileCreateError = CreateLogger()
	if LogFileCreateError != nil {
		fmt.Println("OOPS! Could not initialize a logger")
	} else if Created {
		fmt.Println("Logger initialized successfully")
	}

	go ReadConfig("4tress.config", &RoutingTable)
	// ReadSettings("4tress.settings")
	http.HandleFunc("/health", healthCheck)
	http.HandleFunc("/", ForwardRequest)
	http.HandleFunc("/rateLimit", rateLimitCheck)

	startError := http.ListenAndServe(":6969", nil)
	if startError != nil {
		fmt.Println("ERROR: could not start the server: ", startError)
	}
}
