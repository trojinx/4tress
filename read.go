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
		fmt.Println("ERROR: Coould not read 4tress.config.json: ", err)
		return
	}
	err = json.Unmarshal(configFile, &Settings)
	if err != nil {
		fmt.Println("ERROR: Could not populate Settings struct after reading 4tress.config.json: ", err)
		return
	}
	fmt.Println("Successfully read 4tress.config.json")
}

func ReadRecords() {
	records, err := os.ReadFile("4tress.records.json")
	if err != nil {
		fmt.Println("ERROR: could not read file: ", err)
		return
	}
	err = json.Unmarshal(records, &Dns)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Successfully read 4tress.routes.json")
}
