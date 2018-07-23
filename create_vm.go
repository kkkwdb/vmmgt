package main

import (
	"fmt"
	"github.com/urfave/cli"
	"log"
	"os"
	"os/exec"
)

var createCmd = cli.Command{
	Name:    "create",
	Aliases: []string{"c"},
	Usage:   "create a virtual machine",
	Before:  createCheck,
	Action:  createVm,
}

func createCheck(c *cli.Context) error {
	names := c.GlobalStringSlice("name")
	if len(names) == 0 {
		log.Fatal("name is empty")
	}
	new := names[0]

	doms, err := virtConn.ListAllDomains(0)
	if err != nil {
		log.Fatal(err)
	}
	for _, dom := range doms {
		name, err := dom.GetName()
		if err != nil {
			log.Fatal(err)
		}
		if new == name {
			log.Fatal("the name is already used")
		}
	}

	return nil
}

func createVm(c *cli.Context) {
	name := c.GlobalStringSlice("name")[0]
	disk := fmt.Sprintf("path=/opt/libvirt/disks/%s.img,size=100", name)
	cmd := exec.Command("virt-install",
		"--name", name,
		"--memory", "8192",
		"--disk", disk,
		"--graphics", "vnc,listen=0.0.0.0",
		"--sound", "default",
		"--boot", "hd,cdrom",
		"--vcpus", "8",
		"--noautoconsole",
		"--serial", "pty",
		"--console", "pty,target_type=serial",
		"--network", "network=default,model=virtio",
		"--os-type", "linux",
		"--os-variant", "rhel7",
		"--pxe")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
