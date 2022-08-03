---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "flow_mac_bare_metal_network Resource - terraform-provider-flow"
subcategory: ""
description: |-
  
---

# flow_mac_bare_metal_network (Resource)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `location_id` (Number) unique identifier of the location
- `name` (String) name of the network

### Optional

- `domain_name` (String) domain name of the network
- `domain_name_servers` (List of String) list of domain name servers

### Read-Only

- `allocation_pool` (Attributes) allocation pool (see [below for nested schema](#nestedatt--allocation_pool))
- `cidr` (String) CIDR of the network
- `gateway_ip` (String) gateway IP of the network
- `id` (Number) unique identifier of the network

<a id="nestedatt--allocation_pool"></a>
### Nested Schema for `allocation_pool`

Read-Only:

- `end` (String) end of the allocation pool
- `start` (String) start of the allocation pool

