package main

import (
	"fmt"
	"github.com/libvirt/libvirt-go"
	"github.com/urfave/cli"
	"log"
	"os"
)

var deleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"d", "del"},
	Usage:   "delete a virtual machine",
	Before:  deleteCheck,
	Action:  deleteVm,
}

func deleteCheck(c *cli.Context) error {
	delnames := c.GlobalStringSlice("name")
	if len(delnames) == 0 {
		log.Fatal("name is empty")
	}

	for _, delname := range delnames {
		_, err := virtConn.LookupDomainByName(delname)
		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}

func deleteVm(c *cli.Context) error {
	delnames := c.GlobalStringSlice("name")
	for _, delname := range delnames {
		dom, err := virtConn.LookupDomainByName(delname)
		if err != nil {
			log.Fatal(err)
		}
		state, _, err := dom.GetState()
		if err != nil {
			log.Fatal(err)
		}
		if state == libvirt.DOMAIN_RUNNING || state == libvirt.DOMAIN_BLOCKED ||
			state == libvirt.DOMAIN_PAUSED {
			err := dom.Destroy()
			if err != nil {
				log.Fatal(err)
			}
		}
		err = dom.Undefine()
		if err != nil {
			log.Fatal(err)
		}
		os.Remove("/opt/libvirt/disks/" + delname + ".img")
	}
	fmt.Println("delete vm", c.GlobalStringSlice("name"))
	return nil
}
