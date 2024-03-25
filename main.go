package main

import (
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"os"
)

// Struct to hold information from Zscaler API and config file
type (
	zpaApi struct {
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

	hubApi struct {
		CloudName   string   `json:"cloudName"`
		Type        string   `json:"type"`
		HubPrefixes []string `json:"hubPrefixes"`
	}

	myConfig struct {
		Main      cfgMain
		Zscaler   cfgZscaler
		Resources cfgResources
	}

	cfgMain struct {
		Priority int
		Output   string
	}

	cfgZscaler struct {
		Zpa    cfgZscalerZpa
		Hub    cfgZscalerHub
		Custom cfgZscalerCustom
	}

	cfgZscalerZpa struct {
		Enabled bool
		Url     string
	}

	cfgZscalerHub struct {
		Enabled bool
		Url     string
	}

	cfgZscalerCustom struct {
		Enabled bool
		Url     string
	}

	cfgResources struct {
		RgNameTf  string
		NsgNameTf string
		NsgNameAz string
	}
)

func main() {
	fmt.Println("Starting..")

	var cfg myConfig
	configFile, err := os.ReadFile("config.toml")
	if err != nil {
		fmt.Println("Error loading config file:", err.Error())
	} else {
		err = toml.Unmarshal(configFile, &cfg)
		if err != nil {
			fmt.Println("Error loading config file:", err.Error())
		}
	}

	// Test
	fmt.Println(cfg.Zscaler.Custom.Enabled)
}
