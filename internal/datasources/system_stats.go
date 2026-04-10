package datasources

import (
	"context"
	"fmt"

	"github.com/StevenBuglione/terraform-provider-comfyui/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SystemStatsDataSource{}
var _ datasource.DataSourceWithConfigure = &SystemStatsDataSource{}

type SystemStatsDataSource struct {
	client *client.Client
}

type SystemStatsModel struct {
	OS             types.String  `tfsdk:"os"`
	PythonVersion  types.String  `tfsdk:"python_version"`
	ComfyUIVersion types.String  `tfsdk:"comfyui_version"`
	EmbeddedPython types.Bool    `tfsdk:"embedded_python"`
	Devices        []DeviceModel `tfsdk:"devices"`
}

type DeviceModel struct {
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	Index          types.Int64  `tfsdk:"index"`
	VRAMTotal      types.Int64  `tfsdk:"vram_total"`
	VRAMFree       types.Int64  `tfsdk:"vram_free"`
	TorchVRAMTotal types.Int64  `tfsdk:"torch_vram_total"`
	TorchVRAMFree  types.Int64  `tfsdk:"torch_vram_free"`
}

func NewSystemStatsDataSource() datasource.DataSource {
	return &SystemStatsDataSource{}
}

func (d *SystemStatsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_stats"
}

func (d *SystemStatsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *SystemStatsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves system statistics from the ComfyUI server.",
		Attributes: map[string]schema.Attribute{
			"os": schema.StringAttribute{
				Description: "Operating system of the ComfyUI server.",
				Computed:    true,
			},
			"python_version": schema.StringAttribute{
				Description: "Python version running on the server.",
				Computed:    true,
			},
			"comfyui_version": schema.StringAttribute{
				Description: "ComfyUI version.",
				Computed:    true,
			},
			"embedded_python": schema.BoolAttribute{
				Description: "Whether the server uses an embedded Python installation.",
				Computed:    true,
			},
			"devices": schema.ListNestedAttribute{
				Description: "List of compute devices available on the server.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Device name.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "Device type (e.g., cuda, cpu).",
							Computed:    true,
						},
						"index": schema.Int64Attribute{
							Description: "Device index.",
							Computed:    true,
						},
						"vram_total": schema.Int64Attribute{
							Description: "Total VRAM in bytes.",
							Computed:    true,
						},
						"vram_free": schema.Int64Attribute{
							Description: "Free VRAM in bytes.",
							Computed:    true,
						},
						"torch_vram_total": schema.Int64Attribute{
							Description: "Total Torch VRAM in bytes.",
							Computed:    true,
						},
						"torch_vram_free": schema.Int64Attribute{
							Description: "Free Torch VRAM in bytes.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *SystemStatsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	stats, err := d.client.GetSystemStats()
	if err != nil {
		resp.Diagnostics.AddError("Unable to read system stats", err.Error())
		return
	}

	state := SystemStatsModel{
		OS:             types.StringValue(stats.System.OS),
		PythonVersion:  types.StringValue(stats.System.PythonVersion),
		ComfyUIVersion: types.StringValue(stats.System.ComfyUIVersion),
		EmbeddedPython: types.BoolValue(stats.System.EmbeddedPython),
	}

	for _, dev := range stats.Devices {
		state.Devices = append(state.Devices, DeviceModel{
			Name:           types.StringValue(dev.Name),
			Type:           types.StringValue(dev.Type),
			Index:          types.Int64Value(int64(dev.Index)),
			VRAMTotal:      types.Int64Value(dev.VRAMTotal),
			VRAMFree:       types.Int64Value(dev.VRAMFree),
			TorchVRAMTotal: types.Int64Value(dev.TorchVRAMTotal),
			TorchVRAMFree:  types.Int64Value(dev.TorchVRAMFree),
		})
	}

	// Ensure devices is an empty list rather than null when no devices returned
	if state.Devices == nil {
		state.Devices = []DeviceModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
