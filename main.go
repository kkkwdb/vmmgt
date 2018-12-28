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

	app.Flags = []cli.Flag{}
	app.Commands = []cli.Command{
		createCmd,
		deleteCmd,
		listCmd,
		networkCmd,
		sshCmd,
		cpCmd,
		dnatCmd,
	}

	var err error
	virtConn, err = libvirt.NewConnect("qemu:///system")
	if err != nil {
		log.Fatal(err)
	}
	defer virtConn.Close()

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
