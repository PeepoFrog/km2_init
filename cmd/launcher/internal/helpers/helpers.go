package helpers

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"
)

type LauncherInterface interface {
	PrivilageCheck() error
	CheckPlaform() (architecture string, platform string)
	// WritePubKey() error
	// CosignCheck() error
	// CosignInstall(architecture string, platform string) error
	// ToolsInstall() error
	// SekaiUtilsInstall() error
	// //test
	// SekaiEnvInstall() error
	// SekaidInstall() error
	// InterxInstall() error
}
type Linux struct {
}

func (*Linux) PrivilageCheck() error {

	currentUser, err := user.Current()
	if err != nil {
		println("Error getting current user:", err)
		return err
	}
	sudoUser := os.Getenv("SUDO_USER")
	if currentUser.Uid == "0" {
		if sudoUser != "" {
			fmt.Printf("This application was started with sudo by user %s. Exiting.\n", sudoUser)
			return nil
		} else {
			err = fmt.Errorf("this application should not be run as root. exiting")
			return err
		}
	}
	err = fmt.Errorf("non-root user detected. proceeding with the application")
	return err
}
func (*Linux) CheckPlaform() (architecture string, platform string) {
	architecture = runtime.GOARCH
	platform = runtime.GOOS
	if strings.Contains(architecture, "arm") {
		architecture = "arm64"
	} else {
		architecture = "amd64"
	}
	fmt.Printf("ARCH: %s, PLATFORM: %s\n", architecture, platform)
	return architecture, platform
}
