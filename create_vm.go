package main

import (
	"fmt"
	"github.com/urfave/cli"
	"io"
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
		cli.IntFlag{
			Name:  "netnum",
			Value: 1,
			Usage: "network num, 1:defualt, 2:mgt-net,data-net",
		},
		cli.StringFlag{
			Name:  "macTail",
			Usage: "the mac byte",
		},
		cli.StringFlag{
			Name:  "install,i",
			Usage: "install method: pxe, import, {iso_file}",
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
	if err := os.Mkdir(diskhome+"/disks", 0777); err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
	return diskhome + "/disks"
}

func getCmdPara(c *cli.Context, name string, macTail uint64) []string {
	cmdPara := make(map[string]string)

	netNum := c.Int("netnum")
	diskhome := getDiskHome()
	diskpath := diskhome + "/" + name + ".img"
	disk := fmt.Sprintf("path=%s,size=%s", diskpath, c.String("disk"))
	install := c.String("install")

	mac1 := ""
	mac2 := ""
	if macTail != 0 {
		mac1 = ",mac=52:54:00:51:01:" + strconv.FormatUint(macTail, 16)
		mac2 = ",mac=52:54:00:51:02:" + strconv.FormatUint(macTail, 16)
	}

	cmdPara["--name"] = name
	cmdPara["--memory"] = c.String("memory")
	cmdPara["--disk"] = disk
	cmdPara["--graphics"] = "vnc,listen=0.0.0.0"
	cmdPara["--sound"] = "default"
	cmdPara["--boot"] = "hd,cdrom"
	cmdPara["--vcpus"] = c.String("cpu")
	cmdPara["--noautoconsole"] = ""
	cmdPara["--serial"] = "pty"
	cmdPara["--console"] = "pty,target_type=serial"
	cmdPara["--os-type"] = "linux"
	cmdPara["--os-variant"] = "rhel7"

	if install == "pxe" {
		cmdPara["--pxe"] = ""
	} else if strings.HasSuffix(install, ".iso") {
		cmdPara["--cdrom"] = install
	} else {
		cmdPara["--import"] = ""
		fmt.Printf("copy %s to %s\n", install, diskpath)
		dstFile, err := os.Create(diskpath)
		if err != nil {
			return nil
		}
		srcFile, err := os.Open(install)
		if err != nil {
			return nil
		}
		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return nil
		}
	}

	if netNum == 2 {
		cmdPara["--network1"] = "network=mgt-net,model=virtio" + mac1
		cmdPara["--network"] = "network=data-net,model=virtio" + mac2
	} else if netNum == 1 {
		cmdPara["--network"] = "network=default,model=virtio" + mac1
	} else {
		return nil
	}

	parameters := make([]string, 0)
	for k, v := range cmdPara {
		if k == "--network1" || k == "--network2" {
			k = "--network"
		}
		parameters = append(parameters, k)
		if v != "" {
			parameters = append(parameters, v)
		}
	}
	return parameters
}

func doCreateVm(c *cli.Context, name string, macTail uint64) error {
	cmdPara := getCmdPara(c, name, macTail)
	if cmdPara == nil {
		return fmt.Errorf("invalid parameters")
	}

	fmt.Printf("create vm %s: %s\n", name, cmdPara)
	cmd := exec.Command("virt-install", cmdPara...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	return nil
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
		if err := doCreateVm(c, name, macNum); err != nil {
			log.Fatal(err)
		}
		if macNum != 0 {
			macNum++
			if macNum > 254 {
				log.Fatalf("mac %u out of range", macNum)
			}
		}
	}
	fmt.Println("")
}
