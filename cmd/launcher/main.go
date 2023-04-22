package main

import (
	"github.com/mrlutik/km2_init/internal/launcher"
)

func main() {
	launcherInterface := launcher.LauncherInterface(&launcher.Linux{})
	var err error
	if err = launcherInterface.PrivilageCheck(); err != nil {
		panic(err)
	}
	arch, platform := launcherInterface.CheckPlaform()
	if err = launcherInterface.CosignCheck(); err != nil {
		if err = launcherInterface.CosignInstall(arch, platform); err != nil {
			panic(err)
		}
	}
	if err = launcherInterface.WritePubKey(); err != nil {
		panic(err)
	}
	if err = launcherInterface.ToolsInstall(); err != nil {
		panic(err)
	}
	if err = launcherInterface.SekaiUtilsInstall(); err != nil {
		panic(err)
	}
}
