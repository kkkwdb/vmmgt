package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"github.com/libvirt/libvirt-go"
	"github.com/urfave/cli"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

const (
	pciPathPre = "/sys/bus/pci/devices/0000:"
)

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

var hostDevCmd = cli.Command{
	Name:     "hostdev",
	Category: "tools",
	Aliases:  []string{"hn"},
	Usage:    "list hostdev or attach/detach hostdev to/from vm",
	Subcommands: []cli.Command{
		hostDevAdd,
		hostDevDel,
		hostDevList,
	},
}

var hostDevList = cli.Command{
	Name:        "list",
	Aliases:     []string{"l"},
	Usage:       "list hostdev",
	ArgsUsage:   "{devid[,devid]...|vendor device|vmNamePattern[,vmNamePattern]...}",
	Description: "devid or {vendor device} is for --host option",
	Action:      listHostDev,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose,v",
			Usage: "Display verbose info",
		},
		cli.BoolFlag{
			Name:  "host,t",
			Usage: "Display host hostdev",
		},
		cli.StringFlag{
			Name:  "class,c",
			Usage: "Display hostdev of class, such as 200/280",
		},
		cli.BoolFlag{
			Name:  "regexp,r",
			Usage: "Use regular expression match",
		},
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
	first := string(ob)[:23]

	if verbose {
		cmd = exec.Command("lspci", "-s", devid, "-vv")
		ob, err = cmd.Output()
		if err != nil {
			return "", ""
		}
		other := strings.Split(strings.TrimSuffix(string(ob), "\n"), "\n")
		head := other[0][8:] + "\n"
		info := strings.Join(other[1:], "\n")
		return first, head + info
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
	devids := []string(nil)
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

func getIpOfNetdev(name string) (string, string) {
	inf, err := net.InterfaceByName(name)
	if err != nil {
		return "", ""
	}
	addrs, err := inf.Addrs()
	if err != nil {
		return "", ""
	}
	if len(addrs) < 1 {
		return "", ""
	}
	ipnet, ok := addrs[0].(*net.IPNet)
	if !ok {
		return "", ""
	}
	s := "down"
	if (inf.Flags & net.FlagUp) == net.FlagUp {
		s = "up"
	}
	return ipnet.IP.String(), s
}
func listHostDevById(devid string, verbose bool) string {
	devline, devinfo := getDevinfoById(devid, verbose)
	if devline == "" {
		return ""
	}

	if devinfo != "" {
		devinfo = "\n" + devinfo
	}

	driver := getDriverById(devid)
	if driver == "" {
		return fmt.Sprintf("%-24s%s\n", devline, devinfo)
	}

	netdev := getNetdevById(devid)
	if netdev == "" {
		return fmt.Sprintf("%-24s%-12s%s\n", devline, driver, devinfo)
	}
	ip, s := getIpOfNetdev(netdev)
	if ip == "" {
		return fmt.Sprintf("%-24s%-12s%-8s%s\n", devline, driver, netdev, devinfo)
	}
	return fmt.Sprintf("%-24s%-12s%-8s/%-8s/%-6s%s\n", devline, driver, netdev, ip, s, devinfo)
}

func listHostDevByClass(vendor, device, class string, verbose bool) string {
	devids := getDevIdsByClass(vendor, device, class)
	o := ""
	for _, devid := range devids {
		o += listHostDevById(devid, verbose)
	}
	return o
}

func getHostDevConfig(vm virtMachine) []*hostDevConfig {
	devCofnigs := []*hostDevConfig(nil)
	dom, err := virtConn.LookupDomainByName(vm.name)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	domXml, err := dom.GetXMLDesc(0)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	config := ""
	for scanner := bufio.NewScanner(strings.NewReader(domXml)); scanner.Scan(); {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "<hostdev mode='subsystem'") {
			if config != "" {
				fmt.Println("error: parse vm config")
				dom.Free()
				return nil
			}
			config = line + "\n"
		} else {
			if config != "" {
				config += line + "\n"
			}
			if line == "</hostdev>" {
				v := new(hostDevConfig)
				err := xml.Unmarshal([]byte(config), v)
				if err != nil {
					fmt.Println(err)
					dom.Free()
					return nil
				}
				devCofnigs = append(devCofnigs, v)
				config = ""
			}
		}
	}

	dom.Free()
	return devCofnigs
}

func listHostDev(c *cli.Context) {
	verbose := c.Bool("verbose")
	host := c.Bool("host")
	if host {
		i := 0
		devid := c.Args().Get(i)
		if len(devid) == 7 && devid[2] == ':' && devid[5] == '.' {
			fmt.Print(listHostDevById(devid, verbose))
			i++
			for devid = c.Args().Get(i); devid != ""; {
				if len(devid) == 7 && devid[2] == ':' && devid[5] == '.' {
					fmt.Print(listHostDevById(devid, verbose))
				}
				i++
				devid = c.Args().Get(i)
			}
			return
		}
		classes := []string{
			"280", "200", "201",
		}
		if c.String("class") != "" {
			classes = strings.Split(c.String("class"), ",")
		}
		vendor := c.Args().Get(0)
		device := c.Args().Get(1)
		for _, class := range classes {
			o := listHostDevByClass(vendor, device, class, verbose)
			if o != "" {
				fmt.Print(o)
			}
		}
		return
	}

	method := 0
	if c.Bool("regexp") {
		method = 1
	}

	vms := []virtMachine(nil)
	if c.NArg() == 0 {
		vms = getVms(nil, method)
	}
	for i := 0; i < c.NArg(); i++ {
		vms = append(vms, getVms([]string{c.Args().Get(i)}, method)...)
	}

	for _, vm := range vms {
		hostdevConfigs := getHostDevConfig(vm)
		vmName := vm.name
		for _, c := range hostdevConfigs {
			o := listHostDevById(c.SrcAddress.Bus[2:]+":"+c.SrcAddress.Slot[2:]+"."+
				c.SrcAddress.Function[2:], verbose)
			if o != "" {
				if vmName != "" {
					fmt.Println("vm " + vmName + ":")
					vmName = ""
				}
				fmt.Print(o)
			}
		}
	}
}

var hostDevAdd = cli.Command{
	Name:      "attach",
	Usage:     "attach hostdev to vm",
	ArgsUsage: "{device id} {vmNamePattern}",
	Aliases:   []string{"a"},
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "regexp,r",
			Usage: "Use regular expression match{match first}",
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
	Action: attachHostDev,
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

func attachHostDev(c *cli.Context) {
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

var hostDevDel = cli.Command{
	Name:      "detach",
	Usage:     "detach hostdev from vm",
	ArgsUsage: "{device id[,device id]...}",
	Aliases:   []string{"d"},
	Before: func(c *cli.Context) error {
		if c.NArg() < 1 {
			return fmt.Errorf("invalid parameters")
		}
		for i := 0; i < c.NArg(); i++ {
			devid := c.Args().Get(i)
			if strings.Index(devid, ":") != 2 || strings.Index(devid, ".") != 5 {
				return fmt.Errorf("invalid parameters")
			}
		}
		return nil
	},
	Action: detachHostDev,
}

func detachHostDev(c *cli.Context) {
	for i := 0; i < c.NArg(); i++ {
		devid := c.Args().Get(i)
		devConfig.SrcAddress.Bus = "0x" + devid[:strings.Index(devid, ":")]
		devConfig.SrcAddress.Slot = "0x" + devid[strings.Index(devid, ":")+1:strings.Index(devid, ".")]
		devConfig.SrcAddress.Function = "0x" + devid[strings.Index(devid, ".")+1:]
		v, err := xml.MarshalIndent(devConfig, "", "  ")
		if err != nil {
			fmt.Println(err)
			return
		}
		devConfigXml := string(v)

		virtMachines := getVms(nil, 0)
		for _, vm := range virtMachines {
			dom, err := virtConn.LookupDomainByName(vm.name)
			if err != nil {
				fmt.Println(err)
				return
			}

			err = dom.DetachDevice(devConfigXml)
			dom.Free()
			if err == nil {
				fmt.Printf("%s dettach %s\n", vm.name, devid)
				break
			}
		}
	}
}
