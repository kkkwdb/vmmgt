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
}

func list_vm(c *cli.Context) {
	machines := c.GlobalString("name")
	doms, err := virtConn.ListAllDomains(0)
	if err != nil {
		log.Fatal(err)
	}

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
		}

		dom.Free()
	}
	sort.Slice(orderdNames, func(i, j int) bool { return states[orderdNames[i]] < states[orderdNames[j]] })
	for _, name := range orderdNames {
		fmt.Printf("%-16s\t%s\n", name, stateTable[states[name]])
	}
}
