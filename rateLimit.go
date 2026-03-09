package main

import "time"

func clearMap() {
	time.Sleep(5 * time.Second)
	clear(IpTable)
}

func RateLimit(ip string) bool { // return true if IP is rate limited, else return false
	IpTable[ip]++
	if IpTable[ip] > 2 {
		return true
	}
	go clearMap()

	return false
}
