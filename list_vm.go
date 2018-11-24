package main

import (
	"fmt"
	"github.com/libvirt/libvirt-go"
	"github.com/urfave/cli"
	"log"
	"os"
	"sort"
	"strings"
)

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
		cli.StringSliceFlag{
			Name:  "name,n",
			Usage: "Virtual machine's name",
		},
	},
}

func listVm(c *cli.Context) {
	diskhome := "/home/libvirt"
	f, err := os.Open(diskhome)
	if err != nil {
		diskhome = "/opt/libvirt"
	}
	defer f.Close()

	verbose := c.Bool("verbose")
	machines := c.String("name")
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

		if machines == "[]" || strings.Contains(machines, name) {
			state, _, err := dom.GetState()
			if err != nil {
				log.Fatal(err)
			}
			states[name] = int(state)
			orderdNames = append(orderdNames, name)
			if verbose {
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

	if verbose {
		fmt.Printf("%-8s%-8s%-8s%-8s%-8s%-8s\n", "name", "state", "cpu", "mem(M)", "disk(G)", "interface")
		for _, name := range orderdNames {
			fmt.Printf("%-8s%-8s%-8d%-8d%-8d",
				name, stateTable[states[name]], vcpus[name], memories[name], disks[name])
			for _, inf := range infs[name] {
				fmt.Printf("%-8s ", inf)
			}
			fmt.Println("")
		}
		return
	}
	for _, name := range orderdNames {
		fmt.Printf("%-8s\t%s\n", name, stateTable[states[name]])
	}
}
