package main

import (
	"fmt"
	"github.com/libvirt/libvirt-go"
	"github.com/urfave/cli"
	"log"
	"os"
	"strings"
)

var deleteCmd = cli.Command{
	Name:    "delete",
	Aliases: []string{"d", "del"},
	Usage:   "delete a virtual machine",
	Before:  deleteCheck,
	Action:  deleteVm,
	Flags: []cli.Flag{
		cli.StringSliceFlag{
			Name:  "name,n",
			Usage: "Virtual machine's name",
		},
		cli.StringFlag{
			Name:   "names",
			Hidden: true,
		},
	},
}

func deleteCheck(c *cli.Context) error {
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

	for _, name := range names {
		_, err := virtConn.LookupDomainByName(name)
		if err != nil {
			log.Fatal(err)
		}
	}

	err := c.Set("names", strings.Join(names, " "))
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func deleteVm(c *cli.Context) error {
	delnames := c.String("names")
	for _, delname := range strings.Split(delnames, " ") {
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
		os.Remove(getDiskHome() + "/" + delname + ".img")
	}
	fmt.Println("delete vm", c.StringSlice("name"))
	return nil
}
