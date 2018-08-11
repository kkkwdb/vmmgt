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
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "cpu,c",
			Usage: "Cpu number for vm",
			Value: "8",
		},
		cli.StringFlag{
			Name:  "memory,m",
			Usage: "memory for vm",
			Value: "8192",
		},
		cli.StringFlag{
			Name:  "disk,d",
			Usage: "disk capability for vm",
			Value: "100",
		},
	},
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

func getDiskHome() string {
	diskhome := "/home/libvirt"
	f, err := os.Open(diskhome)
	if err != nil {
		diskhome = "/opt/libvirt"
	}
	defer f.Close()
	if err := os.Mkdir(diskhome+"/disks", 0770); err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
	return diskhome + "/disks"
}

func createVm(c *cli.Context) {
	name := c.GlobalStringSlice("name")[0]
	diskhome := getDiskHome()
	disk := fmt.Sprintf("path=%s/%s.img,size=%s", diskhome, name, c.String("disk"))
	cmd := exec.Command("virt-install",
		"--name", name,
		"--memory", c.String("memory"),
		"--disk", disk,
		"--graphics", "vnc,listen=0.0.0.0",
		"--sound", "default",
		"--boot", "hd,cdrom",
		"--vcpus", c.String("cpu"),
		"--noautoconsole",
		"--serial", "pty",
		"--console", "pty,target_type=serial",
		"--network", "network=default,model=virtio",
		"--network", "network=data-net,model=virtio",
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
