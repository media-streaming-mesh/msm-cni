package main

import (
	"os"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/media-streaming-mesh/msm-cni/internal/cni"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetOutput(os.Stdout)

	skel.PluginMain(cni.CmdAdd, cni.CmdGet, cni.CmdDel, version.All, "msm-cni")
}
