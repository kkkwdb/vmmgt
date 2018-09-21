package main

import (
	"fmt"
	"github.com/urfave/cli"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var createCmd = cli.Command{
	Name:    "create",
	Aliases: []string{"c"},
	Usage:   "create a virtual machine",
	Before:  createCheck,
	Action:  createVm,
	Flags: []cli.Flag{
		cli.StringSliceFlag{
			Name:  "name,n",
			Usage: "Virtual machine's names '-n vm1,vm2'",
		},
		cli.StringFlag{
			Name:   "names",
			Hidden: true,
		},
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
		cli.StringFlag{
			Name:  "macTail",
			Usage: "the mac byte",
		},
	},
}

func createCheck(c *cli.Context) error {
	names := make([]string, 0)
	oriNames := c.StringSlice("name")

	if len(oriNames) == 0 {
		log.Fatal("name is empty")
	}
	for _, name := range oriNames {
		for _, n := range strings.Split(name, ",") {
			names = append(names, n)
		}
	}

	doms, err := virtConn.ListAllDomains(0)
	if err != nil {
		log.Fatal(err)
	}
	domNames := make(map[string]bool)
	for _, dom := range doms {
		name, err := dom.GetName()
		if err != nil {
			log.Fatal(err)
		}

		domNames[name] = true
	}

	for _, new := range names {
		if domNames[new] {
			log.Fatalf("the name '%s' is already used", new)
		}
	}

	err = c.Set("names", strings.Join(names, " "))
	if err != nil {
		log.Fatal(err)
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

func doCreateVm(c *cli.Context, name string, macTail uint64) {
	diskhome := getDiskHome()
	disk := fmt.Sprintf("path=%s/%s.img,size=%s", diskhome, name, c.String("disk"))

	mac1 := ""
	mac2 := ""
	if macTail != 0 {
		mac1 = ",mac=52:54:00:51:01:" + strconv.FormatUint(macTail, 16)
		mac2 = ",mac=52:54:00:51:02:" + strconv.FormatUint(macTail, 16)
	}

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
		"--network", "network=mgt-net,model=virtio"+mac1,
		"--network", "network=data-net,model=virtio"+mac2,
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
func createVm(c *cli.Context) {
	var macNum uint64 = 0
	var err error
	macTail := c.String("macTail")
	names := c.String("names")

	if macTail != "" {
		macNum, err = strconv.ParseUint(macTail, 16, 8)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, name := range strings.Split(names, " ") {
		doCreateVm(c, name, macNum)
		if macNum != 0 {
			macNum++
			if macNum > 254 {
				log.Fatalf("mac %u out of range", macNum)
			}
		}
	}
}
