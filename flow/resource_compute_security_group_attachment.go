package flow

import (
	"context"
	"fmt"
	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/common"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ tfsdk.ResourceType = (*computeSecurityGroupAttachmentResourceType)(nil)
var _ tfsdk.Resource = (*computeSecurityGroupAttachmentResource)(nil)

type computeSecurityGroupAttachmentResourceData struct {
	ServerID           types.Int64   `tfsdk:"server_id"`
	NetworkInterfaceID types.Int64   `tfsdk:"network_interface_id"`
	SecurityGroupIDs   []types.Int64 `tfsdk:"security_group_ids"`
}

func (d *computeSecurityGroupAttachmentResourceData) FromEntity(serverID int, nic compute.NetworkInterface) {
	d.ServerID = types.Int64{Value: int64(serverID)}
	d.NetworkInterfaceID = types.Int64{Value: int64(nic.ID)}

	d.SecurityGroupIDs = make([]types.Int64, len(nic.SecurityGroups))
	for i, group := range nic.SecurityGroups {
		d.SecurityGroupIDs[i] = types.Int64{Value: int64(group.ID)}
	}
}

type computeSecurityGroupAttachmentResourceType struct{}

func (c computeSecurityGroupAttachmentResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"server_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "identifier of the instance",
				Required:            true,
			},
			"network_interface_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "identifier of the network interface of the given instance",
				Required:            true,
			},
			"security_group_ids": {
				Type:                types.ListType{ElemType: types.Int64Type},
				MarkdownDescription: "security groups to attach to the instances network interface",
				Required:            true,
			},
		},
	}, nil
}

func (c computeSecurityGroupAttachmentResourceType) NewResource(
	ctx context.Context,
	p tfsdk.Provider,
) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	return computeSecurityGroupAttachmentResource{
		client: prov.client,
	}, nil
}

type computeSecurityGroupAttachmentResource struct {
	client goclient.Client
}

func (c computeSecurityGroupAttachmentResource) Create(
	ctx context.Context,
	request tfsdk.CreateResourceRequest,
	response *tfsdk.CreateResourceResponse,
) {
	var cfg computeSecurityGroupAttachmentResourceData
	response.Diagnostics.Append(request.Config.Get(ctx, &cfg)...)
	if response.Diagnostics.HasError() {
		return
	}

	if len(cfg.SecurityGroupIDs) == 0 {
		err := removeSecurityGroups(ctx, c.client, int(cfg.ServerID.Value), int(cfg.NetworkInterfaceID.Value))
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove security groups: %s", err))
			return
		}
	}

	ids := make([]int, len(cfg.SecurityGroupIDs))
	for idx, id := range cfg.SecurityGroupIDs {
		ids[idx] = int(id.Value)
	}

	service := compute.NewNetworkInterfaceService(c.client, int(cfg.ServerID.Value))
	networkInterface, err := service.UpdateSecurityGroups(
		ctx,
		int(cfg.NetworkInterfaceID.Value),
		compute.NetworkInterfaceSecurityGroupUpdate{SecurityGroupIDs: ids},
	)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update security groups: %s", err))
		return
	}

	var state computeSecurityGroupAttachmentResourceData
	state.FromEntity(int(cfg.ServerID.Value), networkInterface)

	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (c computeSecurityGroupAttachmentResource) Read(
	ctx context.Context,
	request tfsdk.ReadResourceRequest,
	response *tfsdk.ReadResourceResponse,
) {
	var state computeSecurityGroupAttachmentResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	service := compute.NewNetworkInterfaceService(c.client, int(state.ServerID.Value))
	ifaces, err := service.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list network interfaces: %s", err))
	}

	for _, iface := range ifaces.Items {
		if iface.ID == int(state.NetworkInterfaceID.Value) {
			state.FromEntity(int(state.ServerID.Value), iface)
			response.Diagnostics.Append(response.State.Set(ctx, &state)...)
			return
		}
	}

	response.Diagnostics.AddError("Config Error", "specified network interface does not exist on server")
}

func (c computeSecurityGroupAttachmentResource) Update(
	ctx context.Context,
	request tfsdk.UpdateResourceRequest,
	response *tfsdk.UpdateResourceResponse,
) {
	var state computeSecurityGroupAttachmentResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	var cfg computeSecurityGroupAttachmentResourceData
	response.Diagnostics.Append(request.Config.Get(ctx, &cfg)...)
	if response.Diagnostics.HasError() {
		return
	}

	// remove security group from current server by applying the default security group at that location
	if state.ServerID != cfg.ServerID {
		err := removeSecurityGroups(ctx, c.client, int(state.ServerID.Value), int(state.NetworkInterfaceID.Value))
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove security groups: %s", err))
			return
		}
	}
	// drop all security groups and attach the default group of the location
	if len(cfg.SecurityGroupIDs) == 0 {
		err := removeSecurityGroups(ctx, c.client, int(cfg.ServerID.Value), int(cfg.NetworkInterfaceID.Value))
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove security groups: %s", err))
			return
		}
	}

	ids := make([]int, len(cfg.SecurityGroupIDs))
	for idx, id := range cfg.SecurityGroupIDs {
		ids[idx] = int(id.Value)
	}

	serviceConfig := compute.NewNetworkInterfaceService(c.client, int(cfg.ServerID.Value))
	networkInterfaceCurrent, err := serviceConfig.UpdateSecurityGroups(
		ctx,
		int(cfg.NetworkInterfaceID.Value),
		compute.NetworkInterfaceSecurityGroupUpdate{SecurityGroupIDs: ids},
	)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update security groups: %s", err))
		return
	}

	state.FromEntity(int(cfg.ServerID.Value), networkInterfaceCurrent)
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (c computeSecurityGroupAttachmentResource) Delete(
	ctx context.Context,
	request tfsdk.DeleteResourceRequest,
	response *tfsdk.DeleteResourceResponse,
) {
	var state computeSecurityGroupAttachmentResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := removeSecurityGroups(ctx, c.client, int(state.ServerID.Value), int(state.NetworkInterfaceID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove security groups: %s", err))
		return
	}
}

func removeSecurityGroups(
	ctx context.Context,
	client goclient.Client,
	serverID, networkInterfaceID int,
) error {
	server, err := compute.NewServerService(client).Get(ctx, serverID)
	if err != nil {
		return err
	}

	group, err := findDefaultSecurityGroup(ctx, client, server.Location)
	if err != nil {
		return err
	}

	serviceState := compute.NewNetworkInterfaceService(client, serverID)
	_, err = serviceState.UpdateSecurityGroups(
		ctx,
		networkInterfaceID,
		compute.NetworkInterfaceSecurityGroupUpdate{SecurityGroupIDs: []int{group.ID}},
	)

	return err
}

func findDefaultSecurityGroup(
	ctx context.Context,
	client goclient.Client,
	location common.Location,
) (compute.SecurityGroup, error) {
	securityGroupService := compute.NewSecurityGroupService(client)

	groups, err := securityGroupService.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		return compute.SecurityGroup{}, err
	}

	for _, item := range groups.Items {
		if item.Default && item.Location.ID == location.ID {
			return item, nil
		}
	}

	return compute.SecurityGroup{}, fmt.Errorf("default security group not found")
}
