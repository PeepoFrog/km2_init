package helpers

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
)

type LauncherInterface interface {
	PrivilageCheck() error
	CheckPlaform() (architecture string, platform string)
	InstallDocker() error
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

func (l *Linux) PrivilageCheck() error {

	currentUser, err := user.Current()
	if err != nil {
		log.Println("Error getting current user:", err)
		return err
	}
	sudoUser := os.Getenv("SUDO_USER")
	if currentUser.Uid == "0" {
		if sudoUser != "" {
			log.Printf("This application was started with sudo by user %s. Exiting.\n", sudoUser)
			return nil
		} else {
			err = fmt.Errorf("this application should not be run as root. exiting")
			return err
		}
	}
	err = fmt.Errorf("non-root user detected. proceeding with the application")
	return err
}
func (l *Linux) CheckPlaform() (architecture string, platform string) {
	architecture = runtime.GOARCH
	platform = runtime.GOOS
	if strings.Contains(architecture, "arm") {
		architecture = "arm64"
	} else {
		architecture = "amd64"
	}
	log.Printf("ARCH: %s, PLATFORM: %s\n", architecture, platform)
	return architecture, platform
}

func (l *Linux) InstallDocker() error {
	//SETING UP REPOSITORY
	//Update the apt package index and install packages to allow apt to use a repository over HTTPS:
	// sudo apt-get update
	fmt.Println("INSTALING DOCKER")
	var output []byte
	var err error
	if output, err = l.RunBashCommandWithSudo(`apt-get update`); err != nil {
		log.Println(err)

		return err
	}
	log.Println(string(output))

	// sudo apt-get install \
	// ca-certificates \
	// curl \
	// gnupg
	if output, err = l.RunBashCommandWithSudo(`apt-get install ca-certificates  curl  gnupg -y`); err != nil {

		return err
	}
	log.Println(string(output))

	//Add Docker’s official GPG key:
	//sudo install -m 0755 -d /etc/apt/keyrings

	if output, err = l.RunBashCommandWithSudo(`install -m 0755 -d /etc/apt/keyrings`); err != nil {

		return err
	}
	log.Println(string(output))

	// curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

	if output, err = l.RunBashCommand(`curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor --batch --yes -o /etc/apt/keyrings/docker.gpg`); err != nil {

		return err
	}
	log.Println(string(output))

	// sudo chmod a+r /etc/apt/keyrings/docker.gpg

	if output, err = l.RunBashCommandWithSudo(`chmod a+r /etc/apt/keyrings/docker.gpg`); err != nil {
		return err
	}
	log.Println(string(output))
	//echo \
	// "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
	// "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | \
	// sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

	if output, err = l.RunBashCommand(`echo \
	"deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
	"$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | \
	sudo tee /etc/apt/sources.list.d/docker.list > /dev/null`); err != nil {

		return err
	}
	fmt.Println(string(output))
	log.Println(err, 6)

	if output, err = l.RunBashCommandWithSudo(`apt-get update`); err != nil {

		return err
	}
	log.Println(string(output))
	//sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
	log.Println(err, 7)

	if output, err = l.RunBashCommandWithSudo(`apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin -y`); err != nil {

		return err
	}
	log.Println(string(output))
	return nil
}
func (l *Linux) RunBashCommandWithSudo(command string) (output []byte, err error) {
	cmd := exec.Command("sudo", "bash", "-c", command)
	if output, err = cmd.Output(); err != nil {
		log.Println(err, 8, "runbashwithsudo")

		return output, err
	}
	return output, nil
}
func (l *Linux) RunBashCommand(command string) (output []byte, err error) {
	cmd := exec.Command("bash", "-c", command)
	if output, err = cmd.Output(); err != nil {
		return output, err
	}
	return output, nil
}
