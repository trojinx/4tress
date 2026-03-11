package main

import (
	"encoding/json"
	"fmt"
	"os"
)

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
