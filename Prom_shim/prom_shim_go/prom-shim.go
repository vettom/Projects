package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

func main() {
	endpointUrl := getEndpoint("app1")
	fmt.Println(endpointUrl)
}

func getEndpoint(a string) string {
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
	}
	// Iterate over the map and print the key-value pairs
	var Link string
	for key, value := range data {
		if key == a {
			Link = value
		}
	}
	return Link
}
