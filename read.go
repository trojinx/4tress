package main

import (
	"encoding/json"
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
	fmt.Println("Mapped all domains to URLs successfully")
}

func ReadConfigJson() {
	configFile, err := os.ReadFile("4tress.config.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = json.Unmarshal(configFile, &Settings)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Successfully read 4tress.config.json")
}
