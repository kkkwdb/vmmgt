package main

import (
	"fmt"
	"github.com/urfave/cli"
	"log"
	netlib "net"
	"sort"
	"strings"
)

var networkCmd = cli.Command{
	Name:    "network",
	Aliases: []string{"n"},
	Usage:   "network init/list/delete",
	Subcommands: []cli.Command{
		listNetCmd,
	},
}

var listNetCmd = cli.Command{
	Name:    "list",
	Aliases: []string{"l"},
	Usage:   "list networks",
	Action:  listNetworks,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose,v",
			Usage: "Display more network information",
		},
		cli.StringSliceFlag{
			Name:  "name,n",
			Usage: "Network name",
		},
	},
}

func listNetworks(c *cli.Context) {
	verbose := c.Bool("verbose")
	networks := c.String("name")
	nets, err := virtConn.ListAllNetworks(0)
	if err != nil {
		log.Fatal(err)
	}

	actives := make(map[string]bool)
	orderdNames := make([]string, 0)
	persistents := make(map[string]bool)
	autostarts := make(map[string]bool)
	brNames := make(map[string]string)
	brAddrs := make(map[string]string)
	for _, net := range nets {
		name, err := net.GetName()
		if err != nil {
			log.Fatal(err)
		}

		if networks == "[]" || strings.Contains(networks, name) {
			orderdNames = append(orderdNames, name)

			active, err := net.IsActive()
			if err != nil {
				log.Fatal(err)
			}
			actives[name] = active

			brNames[name], err = net.GetBridgeName()
			if err != nil {
				log.Fatal(err)
			}

			brAddrs[name] = ""
			if active {
				inf, err := netlib.InterfaceByName(brNames[name])
				if err == nil {
					addrs, err := inf.Addrs()
					if err == nil && len(addrs) >= 1 {
						brAddrs[name] = addrs[0].String()
					}
				}
			}

			if verbose {
				persistents[name], err = net.IsPersistent()
				if err != nil {
					log.Fatal(err)
				}

				autostarts[name], err = net.GetAutostart()
				if err != nil {
					log.Fatal(err)
				}
			}
		}

		net.Free()
	}
	sort.Slice(orderdNames, func(i, j int) bool {
		less := orderdNames[i] < orderdNames[j]
		if orderdNames[i] == orderdNames[j] {
			less = actives[orderdNames[i]]
		}
		return less
	})

	if verbose {
		fmt.Printf("%-8s\t%-8s\t%-8s\t%-8s\t%-8s\t%-8s\n",
			"name", "active", "persistent", "autostart", "bridge", "addr")
		for _, name := range orderdNames {
			fmt.Printf("%-8s\t%-8t\t%-8t\t%-8t\t%-8s\t%-8s\n",
				name, actives[name], persistents[name], autostarts[name], brNames[name], brAddrs[name])
		}
		return
	}
	for _, name := range orderdNames {
		fmt.Printf("%-8s\t%s\n", name, brAddrs[name])
	}
}
