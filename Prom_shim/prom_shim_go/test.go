package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

func main() {
	// Read YAML file
	yamlFile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Failed to read YAML file: %v", err)
	}
	// Create a map to store the key-value pairs
	data := make(map[string]string)

	// Parse the YAML string into the map
	err = yaml.Unmarshal([]byte(yamlFile), &data)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Iterate over the map and print the key-value pairs
	for key, value := range data {
		fmt.Printf("%s: %s\n", key, value)
	}
}
