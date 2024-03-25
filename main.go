package main

import "fmt"

// Struct to hold information from Zscaler zpa api
type zpaApi struct {
	CloudName string `json:"Cloud Name"`
	Content   []struct {
		IPProtocol string   `json:"IP Protocol"`
		Port       int      `json:"Port"`
		Source     string   `json:"Source"`
		Domains    string   `json:"Domains"`
		IPs        []string `json:"IPs"`
		DateAdded  string   `json:"Date Added"`
	} `json:"content"`
}

type hubApi struct {
	CloudName   string   `json:"cloudName"`
	Type        string   `json:"type"`
	HubPrefixes []string `json:"hubPrefixes"`
}

func main() {
	fmt.Println("Hello World!")
}
