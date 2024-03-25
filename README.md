# Zscaler NSG Tool
Simple tool to generate a Terraform definition of an Azure nsg to only allow outgoing traffic to the specified Zscaler Networks.

### Variables in config.toml
- main.priority: The starting priority of the security rules
- zscaler.zpa.url: This is the URL of the Zscaler Private Access API
- zscaler.hub.url: This is the URL of the Zscaler Hub API
- resources.rgNameTf: The name of the resource group where the NSG should reside used for reference in Terraform
- resources.nsgNameTf: The name of the NSG used for reference in Terraform
- resources.nsgNameAz: The name of the NSG to be used in Azure