package driver

import "github.com/hashicorp/nomad/plugins/shared/hclspec"

// Config contains configuration information for the plugin
type Config struct {
	Emulator string `codec:"emulator"`
}

// TaskConfig contains configuration information for a task that runs within
// this plugin.
type TaskConfig struct {
	Type             string             `codec:"type"`
	OS               OS                 `codec:"os"`
	Disk             []Disk             `codec:"disk"`
	NetworkInterface []NetworkInterface `codec:"network_interface"`
	VNC              *VNC               `codec:"vnc"`
}

type OS struct {
	Arch    string `codec:"arch"`
	Machine string `codec:"machine"`
	Type    string `codec:"type"`
}

type Disk struct {
	Source string `codec:"source"`
	Target string `codec:"target"`
	Device string `codec:"device"`
	Driver Driver `codec:"driver"`
}

type Driver struct {
	Name string `codec:"name"`
	Type string `codec:"type"`
}

type NetworkInterface struct {
	NetworkName string `codec:"network_name"`
	Address     string `codec:"address"`
}

type VNC struct {
	Port      int `codec:"port"`
	Websocket int `codec:"websocket"`
}

var configSpec = hclspec.NewObject(map[string]*hclspec.Spec{
	"emulator": hclspec.NewDefault(
		hclspec.NewAttr("emulator", "string", false),
		hclspec.NewLiteral(`"/usr/bin/qemu-system-x86_64"`),
	),
})

var taskConfigSpec = hclspec.NewObject(map[string]*hclspec.Spec{
	"type": hclspec.NewAttr("type", "string", true),

	"os": hclspec.NewBlock("os", true, hclspec.NewObject(map[string]*hclspec.Spec{
		"arch":    hclspec.NewAttr("arch", "string", true),
		"machine": hclspec.NewAttr("machine", "string", true),
		"type":    hclspec.NewAttr("type", "string", true),
	})),

	"disk": hclspec.NewBlockList("disk", hclspec.NewObject(map[string]*hclspec.Spec{
		"source": hclspec.NewAttr("source", "string", true),
		"driver": hclspec.NewBlock("driver", true, hclspec.NewObject(map[string]*hclspec.Spec{
			"name": hclspec.NewAttr("name", "string", true),
			"type": hclspec.NewAttr("type", "string", true),
		})),
		"target": hclspec.NewAttr("target", "string", true),
		"device": hclspec.NewAttr("device", "string", true),
	})),

	"network_interface": hclspec.NewBlockList("network_interface", hclspec.NewObject(map[string]*hclspec.Spec{
		"network_name": hclspec.NewAttr("network_name", "string", true),
		"address":      hclspec.NewAttr("address", "string", false),
	})),

	"vnc": hclspec.NewBlock("vnc", false, hclspec.NewObject(map[string]*hclspec.Spec{
		"port":      hclspec.NewAttr("port", "number", false),
		"websocket": hclspec.NewAttr("websocket", "number", false),
	})),
})
