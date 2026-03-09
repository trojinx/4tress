package main

import (
	"fmt"
	"os"
	"strings"
)

func ReadConfig(configPath string, routingTable *map[string]string) {
	configDataBytes, readErr := os.ReadFile(configPath)
	if readErr != nil {
		fmt.Println("ERROR: could not read the config file:", readErr)
		os.Exit(1)
	}

	lines := strings.Split(string(configDataBytes), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			fmt.Println("Skipping invalid line:", line)
			continue
		}
		domain := strings.TrimSpace(parts[0])
		ip := strings.TrimSpace(parts[1])
		(*routingTable)[domain] = ip
	}
	// fmt.Println(routingTable)
}

// func ReadSettings(settingsPath string) {
// 	settings, readErr := os.ReadFile(settingsPath)
// 	if readErr != nil {
// 		fmt.Println("ERROR: could not read the settings file: ", readErr)
// 	}
// 	fmt.Println(string(settings))
// }
