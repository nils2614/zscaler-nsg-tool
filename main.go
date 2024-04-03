package main

import (
	"encoding/json"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
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
		Priority    int
		OutputNsg   string
		OutputVars  string
		OutputRules string
	}

	cfgZscaler struct {
		Hub    cfgZscalerHub
		Zpa    cfgZscalerZpa
		Custom cfgZscalerCustom
	}

	cfgZscalerHub struct {
		Enabled bool
		Url     string
	}

	cfgZscalerZpa struct {
		Enabled bool
		Url     string
	}

	cfgZscalerCustom struct {
		Enabled bool
		Ips     []string
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

	// Initialise slices to hold the output text
	var outputNsgText []string   // holds content for file cfg.Main.OutputNsg
	var outputVarsText []string  // holds content for file cfg.Main.OutputVars
	var outputRulesText []string // holds content for file cfg.Main.OutputRules

	// Generate output for cfg.Main.OutputNsg and cfg.Main.OutputVars
	outputNsgText = generateNsgDefinition(cfg.Resources.RgNameTf, cfg.Resources.NsgNameTf, cfg.Resources.NsgNameAz)
	outputVarsText = generateVarDefinition(cfg.Resources.NsgNameTf)

	// Generate output for cfg.Main.OutputRules
	currentPrio := cfg.Main.Priority
	fmt.Println("First priority: " + strconv.Itoa(currentPrio))
	if cfg.Zscaler.Hub.Enabled {
		outputRulesText = append(outputRulesText, appendHubRules(cfg.Zscaler.Hub.Url, &currentPrio)...)
	}
	if cfg.Zscaler.Zpa.Enabled {
		outputRulesText = append(outputRulesText, appendZpaRules(cfg.Zscaler.Zpa.Url, &currentPrio)...)
	}
	if cfg.Zscaler.Custom.Enabled {
		outputRulesText = append(outputRulesText, appendCustomRules(cfg.Zscaler.Custom.Ips, &currentPrio)...)
	}

	// Add head and commas in between the security rules
	addFormatting(outputRulesText, cfg.Resources.NsgNameTf)

	// Write to output files
	writeToFile(outputNsgText, cfg.Main.OutputNsg)
	writeToFile(outputVarsText, cfg.Main.OutputVars)
	writeToFile(outputRulesText, cfg.Main.OutputRules)

	fmt.Println("Terraform code successfully generated!")
}

func appendHubRules(url string, priority *int) []string {
	// Request data and store body in var.body
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("No response from request")
	}

	// Close ReadAll of resp.Body
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing body of data request")
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)

	// Unmarshal JSON from var.body to struct.result
	var result hubApi
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("Can not unmarshal JSON")
	}

	// Create destination string and generate security rules
	fmt.Println("Rules are being generated for Zscaler Hub IPs")
	destinations := makeDestinationList(result.HubPrefixes)
	outputRules := generateSecurityRule("AllowZscaler-Hub", *priority, "Outbound", "Allow", "*", "443", destinations)
	*priority++

	return outputRules
}

func appendZpaRules(url string, priority *int) []string {
	// Request data and store body in var.body
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("No response from request")
	}

	// Close ReadAll of resp.Body
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing body of data request")
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)

	// Unmarshal JSON from var.body to struct.result
	var result zpaApi
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("Can not unmarshal JSON")
	}

	// Create destination string and generate security rules for each IP block
	var outputRules []string
	for i := 0; i < len(result.Content); i++ {
		fmt.Println("Rules are being generated for IP Block " + strconv.Itoa(i+1) + " (" + result.Content[i].DateAdded + ").")
		ruleName := "AllowZscaler" + "-Zpa-" + strconv.Itoa(i+1)

		destinations := makeDestinationList(result.Content[i].IPs)
		outputRules = append(outputRules, generateSecurityRule(ruleName, *priority, "Outbound", "Allow", "*", "443", destinations)...)

		*priority++
	}

	return outputRules
}

func appendCustomRules(ips []string, priority *int) []string {
	// Create destination string and generate security rules
	fmt.Println("Rules are being generated for " + strconv.Itoa(len(ips)) + " Custom IPs")
	destinations := makeDestinationList(ips)
	outputRules := generateSecurityRule("AllowZscaler-Custom", *priority, "Outbound", "Allow", "*", "443", destinations)
	*priority++

	return outputRules
}

func makeDestinationList(inputArray []string) string {
	// Generate destination address slice
	var onlyIPv4 []string
	for i := 0; i < len(inputArray); i++ {
		if isIPv4(inputArray[i]) {
			onlyIPv4 = append(onlyIPv4, inputArray[i])
		}
	}

	// Store final result
	var destinations string

	// Create String including all destinations
	for i := 0; i < len(onlyIPv4); i++ {
		if isIPv4(onlyIPv4[i]) {
			//ruleName := "AllowZscaler-Hub" + "-" + strconv.Itoa(i+1)
			//whitelistRules = append(whitelistRules, generateSecurityRule(ruleName, *priority, "Outbound", "Allow", "*", "443", result.HubPrefixes[i])...)
			if 0 == len(onlyIPv4)-1 {
				destinations = "[\"" + onlyIPv4[i] + "\"]"
			} else if i == 0 {
				destinations = destinations + "[\"" + onlyIPv4[i] + "\""
			} else if i == len(onlyIPv4)-1 {
				destinations = destinations + ", \"" + onlyIPv4[i] + "\"]"
			} else {
				destinations = destinations + ", \"" + onlyIPv4[i] + "\""
			}
			//*priority++
		}
	}

	// Return magical string
	return destinations
}

func generateNsgDefinition(rgNameTf string, nsgNameTf string, nsgNameAz string) []string {
	var NsgDefinition []string // empty slice to store Terraform code line by line

	// Terraform NSG definition code
	NsgDefinition = append(NsgDefinition, "# Terraform code generated by zscaler-nsg-tool")
	NsgDefinition = append(NsgDefinition, "resource \"azurerm_network_security_group\" \""+nsgNameTf+"\" {")
	NsgDefinition = append(NsgDefinition, "  name                = \""+nsgNameAz+"\"")
	NsgDefinition = append(NsgDefinition, "  location            = data.azurerm_resource_group."+rgNameTf+".location")
	NsgDefinition = append(NsgDefinition, "  resource_group_name = data.azurerm_resource_group."+rgNameTf+".name")
	NsgDefinition = append(NsgDefinition, "")

	NsgDefinition = append(NsgDefinition, "  dynamic \"security_rule\" {")
	NsgDefinition = append(NsgDefinition, "    for_each = var.restrict_zscaler")
	NsgDefinition = append(NsgDefinition, "    content {")
	NsgDefinition = append(NsgDefinition, "      name                         = security_rule.value.name")
	NsgDefinition = append(NsgDefinition, "      priority                     = security_rule.value.priority")
	NsgDefinition = append(NsgDefinition, "      direction                    = security_rule.value.direction")
	NsgDefinition = append(NsgDefinition, "      access                       = security_rule.value.access")
	NsgDefinition = append(NsgDefinition, "      protocol                     = security_rule.value.protocol")
	NsgDefinition = append(NsgDefinition, "      source_port_range            = security_rule.value.source_port_range")
	NsgDefinition = append(NsgDefinition, "      destination_port_range       = security_rule.value.destination_port_range")
	NsgDefinition = append(NsgDefinition, "      source_address_prefix        = security_rule.value.source_address_prefix")
	NsgDefinition = append(NsgDefinition, "      destination_address_prefixes = security_rule.value.destination_address_prefixes")
	NsgDefinition = append(NsgDefinition, "    }")
	NsgDefinition = append(NsgDefinition, "  }")
	NsgDefinition = append(NsgDefinition, "")

	NsgDefinition = append(NsgDefinition, "  security_rule {")
	NsgDefinition = append(NsgDefinition, "    name                       = \"DenyInternetOutbound\"")
	NsgDefinition = append(NsgDefinition, "    priority                   = 4001")
	NsgDefinition = append(NsgDefinition, "    direction                  = \"Outbound\"")
	NsgDefinition = append(NsgDefinition, "    access                     = \"Deny\"")
	NsgDefinition = append(NsgDefinition, "    protocol                   = \"*\"")
	NsgDefinition = append(NsgDefinition, "    source_port_range          = \"*\"")
	NsgDefinition = append(NsgDefinition, "    destination_port_range     = \"*\"")
	NsgDefinition = append(NsgDefinition, "    source_address_prefix      = \"*\"")
	NsgDefinition = append(NsgDefinition, "    destination_address_prefix = \"Internet\"")
	NsgDefinition = append(NsgDefinition, "    }")

	NsgDefinition = append(NsgDefinition, "")
	NsgDefinition = append(NsgDefinition, "}")

	return NsgDefinition
}

func generateVarDefinition(nsgNameTf string) []string {
	var VarDefinition []string // empty slice to store Terraform code line by line

	// Terraform variable definition code
	VarDefinition = append(VarDefinition, "# Terraform code generated by zscaler-nsg-tool")
	VarDefinition = append(VarDefinition, "variable \""+nsgNameTf+"\" {")
	VarDefinition = append(VarDefinition, "  type = list(object({")

	VarDefinition = append(VarDefinition, "    name                         = string")
	VarDefinition = append(VarDefinition, "    priority                     = number")
	VarDefinition = append(VarDefinition, "    direction                    = string")
	VarDefinition = append(VarDefinition, "    access                       = string")
	VarDefinition = append(VarDefinition, "    protocol                     = string")
	VarDefinition = append(VarDefinition, "    source_port_range            = string")
	VarDefinition = append(VarDefinition, "    destination_port_range       = string")
	VarDefinition = append(VarDefinition, "    source_address_prefix        = string")
	VarDefinition = append(VarDefinition, "    destination_address_prefixes = list(string)")
	VarDefinition = append(VarDefinition, "  }))")
	VarDefinition = append(VarDefinition, "  description = \"Security rules for the "+nsgNameTf+" NSG\"")
	VarDefinition = append(VarDefinition, "}")

	return VarDefinition
}

func generateSecurityRule(name string, priority int, direction string, access string, protocol string, port string, ips string) []string {
	var securityRule []string
	securityRule = append(securityRule, "{")
	securityRule = append(securityRule, "  name                         = \""+name+"\"")
	securityRule = append(securityRule, "  priority                     = "+strconv.Itoa(priority))
	securityRule = append(securityRule, "  direction                    = \""+direction+"\"")
	securityRule = append(securityRule, "  access                       = \""+access+"\"")
	securityRule = append(securityRule, "  protocol                     = \""+protocol+"\"")
	securityRule = append(securityRule, "  source_port_range            = \"*\"")
	securityRule = append(securityRule, "  destination_port_range       = \""+port+"\"")
	securityRule = append(securityRule, "  source_address_prefix        = \"*\"")
	securityRule = append(securityRule, "  destination_address_prefixes = "+ips+" #tfsec:ignore:azure-network-no-public-egress")
	securityRule = append(securityRule, "}")
	return securityRule
}

func writeToFile(lines []string, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating file on disk")
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println("Error closing file")
		}
	}(file)

	for _, line := range lines {
		_, err := file.WriteString(line + "\n")
		if err != nil {
			fmt.Println("Error writing to file on disk")
		}
	}
}

func isIPv4(ip string) bool {
	// Very naive function, but enough considering we know what the API returns
	if strings.Count(ip, ".") == 3 {
		return true
	}
	return false
}

func addFormatting(outputRules []string, nsgNameTf string) {
	// Add commas
	for i := 10; i < len(outputRules)-11; i = i + 11 {
		outputRules[i] = outputRules[i] + ","
	}

	// Add head and brackets
	outputRules[0] = nsgNameTf + " = [" + outputRules[0]
	outputRules[len(outputRules)-1] = outputRules[len(outputRules)-1] + "]"
}
