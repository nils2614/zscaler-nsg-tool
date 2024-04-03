# Terraform code generated by zscaler-nsg-tool
variable "restrict_zscaler" {
  type = list(object({
    name                         = string
    priority                     = number
    direction                    = string
    access                       = string
    protocol                     = string
    source_port_range            = string
    destination_port_range       = string
    source_address_prefix        = string
    destination_address_prefixes = list(string)
  }))
  description = "Security rules for the restrict_zscaler NSG"
}