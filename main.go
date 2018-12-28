package main

import (
	"github.com/libvirt/libvirt-go"
	"github.com/urfave/cli"
	"log"
	"os"
)

var virtConn *libvirt.Connect

func main() {
	app := cli.NewApp()
	app.Name = "vmmgt"
	app.Usage = "Manage virtual machines"
	app.Version = "v0.7"

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
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
	virtConn.Close()
}
