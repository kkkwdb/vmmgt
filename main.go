package main

import (
	"github.com/libvirt/libvirt-go"
	"github.com/urfave/cli"
	"log"
	"os"
	"os/exec"
)

var virtConn *libvirt.Connect

func getVer() string {
	ver, err := exec.Command("git", "describe", "--tags", "--dirty").Output()
	if err != nil {
		return ""
	}
	return string(ver)
}

func main() {
	app := cli.NewApp()
	app.Name = "vmmgt"
	app.Version = getVer()
	app.Usage = "Manage virtual machines"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "wangdb",
			Email: "wangdb@sugon.com",
		},
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "connect,c",
			Usage: "Connect to hypervisor",
		},
	}

	app.Before = func(c *cli.Context) error {
		var err error
		hv := c.String("connect")
		if hv == "" {
			virtConn, err = libvirt.NewConnect("qemu:///system")
		} else {
			virtConn, err = libvirt.NewConnect("qemu+ssh://" + hv + "/system")
		}
		if err != nil {
			return err
		}
		return nil
	}

	app.Commands = []cli.Command{
		createCmd,
		deleteCmd,
		listCmd,
		networkCmd,
		sshCmd,
		cpCmd,
		dnatCmd,
		hostDevCmd,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
	if virtConn != nil {
		virtConn.Close()
	}
}
