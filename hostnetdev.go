package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"github.com/libvirt/libvirt-go"
	"github.com/urfave/cli"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

const (
	pciPathPre = "/sys/bus/pci/devices/0000:"
)

var hostNetdevCmd = cli.Command{
	Name:     "hostnetdev",
	Category: "tools",
	Aliases:  []string{"hn"},
	Usage:    "list hostnetdev or add/remove hostdev to/from vm",
	Action:   listHostDev,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose,v",
			Usage: "Display verbose info",
		},
		cli.BoolFlag{
			Name:  "all,a",
			Usage: "Display all netdev",
		},
		cli.StringFlag{
			Name:  "class,c",
			Usage: "Display netdev of class, such as 200/280",
			Value: "280",
		},
	},
	Subcommands: []cli.Command{
		hostNetdevAdd,
		hostNetdevDel,
	},
}

func getNetdevById(devid string) string {
	netdir := pciPathPre + devid + "/net"
	if _, err := os.Stat(netdir); err != nil {
		return ""
	}
	cmd := exec.Command("ls", netdir)
	ob, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(ob), "\n")
}

func getDevinfoById(devid string, verbose bool) (string, string) {
	cmd := exec.Command("lspci", "-s", devid, "-n")
	ob, err := cmd.Output()
	if err != nil {
		return "", ""
	}
	first := strings.TrimSuffix(string(ob), "\n")

	if verbose {
		cmd = exec.Command("lspci", "-s", devid, "-vv")
		ob, err = cmd.Output()
		if err != nil {
			return "", ""
		}
		other := strings.TrimSuffix(string(ob), "\n")
		return first, strings.Join(strings.Split(string(other), "\n")[1:], "\n")
	}
	return first, ""
}

func getDriverById(devid string) string {
	driverLink := pciPathPre + devid + "/driver"
	link, err := os.Readlink(driverLink)
	if err != nil {
		return ""
	}
	return path.Base(link)
}

func getDevIdsByClass(vendor, device, class string) []string {
	devids := make([]string, 0)
	cmd := exec.Command("lspci", "-d", vendor+":"+device+":"+class, "-n")
	ob, err := cmd.Output()
	if err != nil {
		fmt.Print(err)
		return devids
	}
	o := strings.TrimSuffix(string(ob), "\n")
	for _, devline := range strings.Split(o, "\n") {
		if devline == "" {
			break
		}
		devids = append(devids, strings.Split(devline, " ")[0])
	}
	return devids
}

func listHostDevByClass(vendor, device, class string, verbose bool) {
	devids := getDevIdsByClass(vendor, device, class)
	for _, devid := range devids {
		devline, devinfo := getDevinfoById(devid, verbose)
		if devinfo != "" {
			devinfo = "\n" + devinfo
		}

		netdev := getNetdevById(devid)
		if netdev == "" {
			fmt.Println(devline + devinfo)
			continue
		}

		driver := getDriverById(devid)
		if driver == "" {
			fmt.Println(devline + " " + netdev + devinfo)
			continue
		}

		fmt.Println(devline + " " + netdev + " " + driver + devinfo)
	}
}

func listHostDev(c *cli.Context) {
	verbose := c.Bool("verbose")
	all := c.Bool("all")
	classes := strings.Split(c.String("class"), ",")
	if all {
		classes = []string{
			"280", "200", "201",
		}
	}
	for _, class := range classes {
		listHostDevByClass("", "", class, verbose)
	}
}

type DevAddr struct {
	XMLName       xml.Name `xml:"address"`
	Type          string   `xml:"type,attr"`
	Domain        string   `xml:"domain,attr,omitempty"`
	Bus           string   `xml:"bus,attr,omitempty"`
	Slot          string   `xml:"slot,attr,omitempty"`
	Function      string   `xml:"function,attr,omitempty"`
	multifunction string   `xml:"function,attr,omitempty"`
}

type hostDevConfig struct {
	XMLName    xml.Name `xml:"hostdev"`
	Mode       string   `xml:"mode,attr"`
	Type       string   `xml:"type,attr"`
	Managed    string   `xml:"managed,attr"`
	SrcAddress *DevAddr `xml:"source>address"`
	DstAddress *DevAddr `xml:"address"`
}

var devConfig = &hostDevConfig{
	Mode:    "subsystem",
	Type:    "pci",
	Managed: "yes",
	SrcAddress: &DevAddr{
		Type:   "pci",
		Domain: "0x0000",
	},
	DstAddress: &DevAddr{
		Type:   "pci",
		Domain: "0x0000",
	},
}

var hostNetdevAdd = cli.Command{
	Name:  "add",
	Usage: "add hostdev to vm",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "regexp,r",
			Usage: "Use regular expression match",
		},
	},
	Before: func(c *cli.Context) error {
		if c.NArg() < 2 {
			return fmt.Errorf("invalid parameters")
		}
		devid := c.Args().First()
		if strings.Index(devid, ":") != 2 || strings.Index(devid, ".") != 5 {
			return fmt.Errorf("invalid parameters")
		}
		return nil
	},
	Action: addHostDev,
}

func getDomAvailPciId(dom *libvirt.Domain) (*DevAddr, error) {
	addr := DevAddr{Type: "pci"}
	max := DevAddr{Type: "pci", Bus: "0x00", Slot: "0x00", Function: "0x"}
	bus := int64(-1)
	slot := int64(-1)
	function := int64(-1)

	domXml, err := dom.GetXMLDesc(0)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	for scanner := bufio.NewScanner(strings.NewReader(domXml)); scanner.Scan(); {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "<address type='pci'") {
			err = xml.Unmarshal([]byte(line), &addr)
			if err != nil {
				fmt.Println(err)
				continue
			}
			bus1, err1 := strconv.ParseInt(addr.Bus[2:], 16, 0)
			slot1, err2 := strconv.ParseInt(addr.Slot[2:], 16, 0)
			function1, err3 := strconv.ParseInt(addr.Function[2:], 16, 0)
			if err1 != nil || err2 != nil || err3 != nil {
				continue
			}
			if bus1 > bus || (bus1 == bus && slot1 > slot) ||
				(bus1 == bus && slot1 == slot && function1 > function) {
				bus = bus1
				slot = slot1
				function = function1
				max = addr
			}
		}
	}
	if bus < 0 {
		return nil, fmt.Errorf("parse vm xml error")
	}
	if slot < 32 {
		slot++
		max.Slot = "0x" + strconv.FormatInt(slot, 16)
	} else {
		bus++
		max.Bus = "0x" + strconv.FormatInt(bus, 16)
	}
	return &max, nil
}

func addHostDev(c *cli.Context) {
	devid := c.Args().First()
	devConfig.SrcAddress.Bus = "0x" + devid[:strings.Index(devid, ":")]
	devConfig.SrcAddress.Slot = "0x" + devid[strings.Index(devid, ":")+1:strings.Index(devid, ".")]
	devConfig.SrcAddress.Function = "0x" + devid[strings.Index(devid, ".")+1:]

	method := 0
	if c.Bool("regexp") {
		method = 1
	}
	vmName := c.Args().Get(1)
	virtMachines := getVms(nil, method)
	for _, vm := range virtMachines {
		matched := matchName(vm.name, []string{vmName}, method)
		if !matched {
			continue
		}

		dom, err := virtConn.LookupDomainByName(vmName)
		if err != nil {
			fmt.Println(err)
			return
		}

		addr, err := getDomAvailPciId(dom)
		devConfig.DstAddress.Bus = addr.Bus
		devConfig.DstAddress.Slot = addr.Slot
		devConfig.DstAddress.Function = addr.Function

		v, err := xml.MarshalIndent(devConfig, "", "  ")
		if err != nil {
			fmt.Println(err)
			return
		}
		err = dom.AttachDevice(string(v))
		if err != nil {
			fmt.Println(err)
			return
		}
		dom.Free()
		break
	}
}

var hostNetdevDel = cli.Command{
	Name:  "del",
	Usage: "del hostdev from vm",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "regexp,r",
			Usage: "Use regular expression match",
		},
	},
	Before: func(c *cli.Context) error {
		if c.NArg() < 2 {
			return fmt.Errorf("invalid parameters")
		}
		devid := c.Args().First()
		if strings.Index(devid, ":") != 2 || strings.Index(devid, ".") != 5 {
			return fmt.Errorf("invalid parameters")
		}
		return nil
	},
	Action: delHostDev,
}

func delHostDev(c *cli.Context) {
	devid := c.Args().First()
	devConfig.SrcAddress.Bus = "0x" + devid[:strings.Index(devid, ":")]
	devConfig.SrcAddress.Slot = "0x" + devid[strings.Index(devid, ":")+1:strings.Index(devid, ".")]
	devConfig.SrcAddress.Function = "0x" + devid[strings.Index(devid, ".")+1:]
	v, err := xml.MarshalIndent(devConfig, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	devConfigXml := string(v)

	method := 0
	if c.Bool("regexp") {
		method = 1
	}

	vmName := c.Args().Get(1)

	virtMachines := getVms(nil, method)
	for _, vm := range virtMachines {
		matched := matchName(vm.name, []string{vmName}, method)
		if !matched {
			continue
		}

		dom, err := virtConn.LookupDomainByName(vmName)
		if err != nil {
			fmt.Println(err)
			return
		}

		err = dom.DetachDevice(devConfigXml)
		if err != nil {
			fmt.Println(err)
			return
		}
		dom.Free()
		break
	}
}
