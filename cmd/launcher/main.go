package main

import (
	"github.com/mrlutik/km2_init/internal/launcher"
)

func main() {
	launcherInterface := launcher.LauncherInterface(&launcher.Linux{})
	var err error
	err = launcherInterface.PrivilageCheck()
	if err != nil {
		panic(err)
	}
	arch, platform := launcherInterface.CheckPlaform()
	// err = launcherInterface.CosignCheck()
	// if err != nil {
	// 	err = launcherInterface.CosignInstall(arch, platform)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }
	// 	err = launcherInterface.WritePubKey()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	err = launcherInterface.ToolsInstall()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	err = launcherInterface.SekaiUtilsInstall()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	err = launcherInterface.SekaiEnvInstall()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	err = launcherInterface.SekaidInstall()
	// 	if err != nil {
	// 		panic(err)
	// 	}
}
