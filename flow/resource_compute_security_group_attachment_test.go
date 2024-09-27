package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccComputeSecurityGroupAttachment_Basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeSecurityGroupAttachmentConfigBasic),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("flow_compute_security_group_attachment.foobar", "server_id", "1"),
					resource.TestCheckResourceAttr("flow_compute_security_group_attachment.foobar", "network_interface_id", "1"),
					resource.TestCheckResourceAttr("flow_compute_security_group_attachment.foobar", "security_group_ids", "[]"),
				),
			},
		},
	})
}

const testAccComputeSecurityGroupAttachmentConfigBasic = `
resource "flow_compute_security_group_attachment" "foobar" {
	server_id        		= 1
	network_interface_id 	= 1
	security_group_ids		= []
}
`
