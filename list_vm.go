package main

import (
	"fmt"
	"github.com/libvirt/libvirt-go"
	"github.com/urfave/cli"
	"log"
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
	Action:  list_vm,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose,v",
			Usage: "Display more vm information",
		},
	},
}

func list_vm(c *cli.Context) {
	verbose := c.Bool("verbose")
	machines := c.GlobalString("name")
	doms, err := virtConn.ListAllDomains(0)
	if err != nil {
		log.Fatal(err)
	}

	vcpus := make(map[string]uint)
	memories := make(map[string]uint64)
	disks := make(map[string]uint64)
	states := make(map[string]int)
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
				bi, err := dom.GetBlockInfo("/opt/libvirt/disks/"+name+".img", 0)
				if err != nil {
					log.Fatal(err)
				}
				disks[name] = bi.Capacity / 1024 / 1024 / 1024
			}
		}

		dom.Free()
	}
	sort.Strings(orderdNames)
	sort.Slice(orderdNames, func(i, j int) bool { return states[orderdNames[i]] < states[orderdNames[j]] })

	if verbose {
		fmt.Printf("%-8s\t%-8s\t%-8s\t%-8s\t%-8s\n", "name", "state", "cpu", "memory(M)", "disk(G)")
		for _, name := range orderdNames {
			fmt.Printf("%-8s\t%-8s\t%-8d\t%-8d\t%-8d\n", name, stateTable[states[name]], vcpus[name], memories[name], disks[name])
		}
		return
	}
	for _, name := range orderdNames {
		fmt.Printf("%-8s\t%s\n", name, stateTable[states[name]])
	}
}
