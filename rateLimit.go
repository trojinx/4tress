package main

import "time"

func clearMap() {
	time.Sleep(5 * time.Second)
	clear(IpTable)
}

func RateLimit(ip string) bool { // return true if IP is rate limited, else return false
	IpTable[ip]++
	if IpTable[ip] > int8(Settings.RateLimitAfter) {
		return true
	}
	go clearMap()

	return false
}
