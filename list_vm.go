package main

import (
	"fmt"
	"github.com/libvirt/libvirt-go"
	"github.com/urfave/cli"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type virtMachine struct {
	name   string
	state  string
	vcpu   uint
	memory uint64
	disk   uint64
	infs   []string
}

var stateTable = []string{
	libvirt.DOMAIN_RUNNING:     "running",
	libvirt.DOMAIN_BLOCKED:     "blocked",
	libvirt.DOMAIN_PAUSED:      "paused",
	libvirt.DOMAIN_SHUTDOWN:    "shutdown",
	libvirt.DOMAIN_SHUTOFF:     "shutoff",
	libvirt.DOMAIN_CRASHED:     "crashed",
	libvirt.DOMAIN_PMSUSPENDED: "suspended",
}

var listCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"l"},
	Usage:   "list virtual machines",
	Action:  listVm,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose,v",
			Usage: "Display more vm information",
		},
		cli.BoolFlag{
			Name:  "all,a",
			Usage: "Display all vm, default running vm",
		},
		cli.StringSliceFlag{
			Name:  "name,n",
			Usage: "Virtual machine's name patterns",
		},
		cli.BoolFlag{
			Name:  "regexp,r",
			Usage: "Use regular expression match",
		},
	},
}

func matchName(name string, patterns []string, method int) bool {
	var matched bool
	var err error
	for _, p := range patterns {
		if method == 1 {
			matched, err = regexp.MatchString(p, name)
		} else {
			matched, err = filepath.Match(p, name)
		}
		if err == nil && matched {
			return true
		}
	}
	return false
}

func getVms(machines []string, method int) []virtMachine {
	diskhome := "/home/libvirt"
	f, err := os.Open(diskhome)
	if err != nil {
		diskhome = "/opt/libvirt"
	}
	defer f.Close()

	doms, err := virtConn.ListAllDomains(0)
	if err != nil {
		log.Fatal(err)
	}

	vcpus := make(map[string]uint)
	memories := make(map[string]uint64)
	disks := make(map[string]uint64)
	states := make(map[string]int)
	infs := make(map[string][]string)
	orderdNames := make([]string, 0)
	for _, dom := range doms {
		name, err := dom.GetName()
		if err != nil {
			log.Fatal(err)
		}

		if len(machines) != 0 {
			if matched := matchName(name, machines, method); !matched {
				dom.Free()
				continue
			}
		}
		state, _, err := dom.GetState()
		if err != nil {
			log.Fatal(err)
		}
		states[name] = int(state)
		orderdNames = append(orderdNames, name)
		di, err := dom.GetInfo()
		if err != nil {
			log.Fatal(err)
		}
		memories[name] = di.Memory / 1024
		vcpus[name] = di.NrVirtCpu
		bi, err := dom.GetBlockInfo(diskhome+"/disks/"+name+".img", 0)
		if err != nil {
			log.Fatal(err)
		}
		disks[name] = bi.Capacity / 1024 / 1024 / 1024

		dis, err := dom.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_AGENT)
		if err != nil {
			continue
		}
		infs[name] = make([]string, 0, len(dis))
		for _, di := range dis {
			if di.Name == "lo" {
				continue
			}
			for _, addr := range di.Addrs {
				if strings.Contains(addr.Addr, ":") {
					continue
				}
				infs[name] = append(infs[name], addr.Addr)
			}
		}

		dom.Free()
	}
	sort.Slice(orderdNames, func(i, j int) bool {
		less := orderdNames[i] < orderdNames[j]
		if orderdNames[i] == orderdNames[j] {
			less = states[orderdNames[i]] > states[orderdNames[j]]
		}
		return less
	})

	virtMachines := make([]virtMachine, len(orderdNames))
	for i, name := range orderdNames {
		virtMachines[i].name = name
		virtMachines[i].state = stateTable[states[name]]
		virtMachines[i].vcpu = vcpus[name]
		virtMachines[i].memory = memories[name]
		virtMachines[i].disk = disks[name]
		virtMachines[i].infs = infs[name]
	}
	return virtMachines
}

func listVm(c *cli.Context) {
	machines := c.StringSlice("name")
	for _, m := range c.Args() {
		machines = append(machines, m)
	}
	verbose := c.Bool("verbose")
	method := 0
	if c.Bool("regexp") {
		method = 1
	}
	all := c.Bool("all")

	virtMachines := getVms(machines, method)
	if verbose {
		fmt.Printf("%-16s%-8s%-8s%-8s%-8s%-8s\n", "name", "state", "cpu", "mem(M)", "disk(G)", "interface")
		for _, vm := range virtMachines {
			if !all && stateTable[libvirt.DOMAIN_RUNNING] != vm.state {
				continue
			}
			fmt.Printf("%-16s%-8s%-8d%-8d%-8d", vm.name, vm.state, vm.vcpu, vm.memory, vm.disk)
			for _, inf := range vm.infs {
				fmt.Printf("%-8s ", inf)
			}
			fmt.Println("")
		}
		return
	}
	for _, vm := range virtMachines {
		if !all && stateTable[libvirt.DOMAIN_RUNNING] != vm.state {
			continue
		}
		fmt.Printf("%-8s\t%s\n", vm.name, vm.state)
	}
}
