package main

import (
	"fmt"
	"github.com/urfave/cli"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var dnatCmd = cli.Command{
	Name:     "dnat",
	Category: "tools",
	Aliases:  []string{"dn"},
	Usage:    "list/add/del dnat rule for vms",
	Action:   dnatList,
	Subcommands: []cli.Command{
		dnatListCmd,
		dnatAddCmd,
		dnatDelCmd,
	},
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "regexp,r",
			Usage: "Use regular expression match",
		},
	},
}

var dnatListCmd = cli.Command{
	Name:     "list",
	Category: "tools",
	Aliases:  []string{"l"},
	Usage:    "List dnat rules for vms",
	Action:   dnatList,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "all,a",
			Usage: "Display all dnat rules",
		},
		cli.BoolFlag{
			Name:  "verbose,v",
			Usage: "Display more dnat information",
		},
	},
}

func dnatList(c *cli.Context) {
	cmd := exec.Command("firewall-cmd", "--list-forward-ports")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
	}
	lines := strings.Split(string(output), "\n")

	if c.Bool("all") {
		fmt.Printf("%-16s%-16s%-16s%-16s\n", "Dip", "Sport", "Proto", "Dport")
		for _, line := range lines {
			fs := strings.Split(line, ":")
			if len(fs) < 3 {
				continue
			}
			fmt.Printf("%-16s%-16s%-16s%-16s\n", fs[3][7:], fs[0][5:], fs[1][6:], fs[2][7:])
		}
		return
	}

	verbose := c.Bool("verbose") || c.Parent().Bool("regexp")
	if verbose {
		fmt.Printf("%-16s%-16s%-16s%-16s%-16s\n", "Name", "Ip address", "Host port", "Proto", "Port")
	} else {
		fmt.Printf("%-16s%-16s%-16s%-16s\n", "Name", "Host port", "Proto", "Port")
	}
	method := 0
	if c.Parent().Bool("regexp") || c.Bool("regexp") {
		method = 1
	}
	name := c.Args().First()
	virtMachines := getVms(nil, method)
	for _, vm := range virtMachines {
		if len(vm.infs) < 1 {
			continue
		}
		matched := name == "" || vm.infs[0] == name || matchName(vm.name, []string{name}, method)
		if !matched {
			continue
		}
		for _, line := range lines {
			fs := strings.Split(line, ":")
			if len(fs) < 3 {
				continue
			}
			if strings.Index(line, "toaddr="+vm.infs[0]) > 0 {
				if verbose {
					fmt.Printf("%-16s%-16s%-16s%-16s%-16s\n", vm.name, vm.infs[0], fs[0][5:], fs[1][6:], fs[2][7:])
				} else {
					fmt.Printf("%-16s%-16s%-16s%-16s\n", vm.name, fs[0][5:], fs[1][6:], fs[2][7:])
				}
			}
		}
	}
}

var dnatAddCmd = cli.Command{
	Name:     "add",
	Category: "tools",
	Aliases:  []string{"a"},
	Usage:    "Add dnat rule for vm",
	Flags: []cli.Flag{
		cli.IntFlag{
			Name:  "sport,s",
			Usage: "Host port",
		},
		cli.IntFlag{
			Name:  "dport,d",
			Usage: "Virtual machine port",
		},
	},
	Action: dnatAdd,
	Before: func(c *cli.Context) error {
		if c.NArg() < 1 {
			return fmt.Errorf("No name or ip")
		}
		return nil
	},
}

func dnatAdd(c *cli.Context) {
	sport := strconv.Itoa(c.Int("sport"))
	dport := strconv.Itoa(c.Int("dport"))
	method := 0
	if c.Parent().Bool("regexp") {
		method = 1
	}
	name := c.Args().First()
	virtMachines := getVms(nil, method)
	for _, vm := range virtMachines {
		if len(vm.infs) < 1 {
			continue
		}
		matched := matchName(vm.name, []string{name}, method) || vm.infs[0] == name
		if !matched {
			continue
		}
		arg := "--add-forward-port=port=" + sport + ":proto=tcp:toport=" + dport + ":toaddr=" + vm.infs[0]

		cmd := exec.Command("firewall-cmd", arg)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println(err)
		}

		cmd = exec.Command("firewall-cmd", "--permanent", arg)
		if err := cmd.Run(); err != nil {
			fmt.Println(err)
		}
		return
	}
	fmt.Println("Can't find machine")
}

var dnatDelCmd = cli.Command{
	Name:     "del",
	Category: "tools",
	Aliases:  []string{"d"},
	Usage:    "Delete dnat rule for vm",
	Flags: []cli.Flag{
		cli.IntFlag{
			Name:  "sport,s",
			Value: 0,
			Usage: "Host port",
		},
		cli.IntFlag{
			Name:  "dport,d",
			Value: 0,
			Usage: "Virtual machine port",
		},
	},
	Action: dnatDel,
	Before: func(c *cli.Context) error {
		if c.NArg() < 1 {
			return fmt.Errorf("No name or ip")
		}
		return nil
	},
}

func dnatDel(c *cli.Context) {
	cmd := exec.Command("firewall-cmd", "--list-forward-ports")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
	}
	lines := strings.Split(string(output), "\n")

	sport := strconv.Itoa(c.Int("sport"))
	dport := strconv.Itoa(c.Int("dport"))
	method := 0
	if c.Parent().Bool("regexp") {
		method = 1
	}
	name := c.Args().First()
	virtMachines := getVms(nil, method)
	for _, vm := range virtMachines {
		if len(vm.infs) < 1 {
			continue
		}
		matched := matchName(vm.name, []string{name}, method) || vm.infs[0] == name
		if !matched {
			continue
		}

		arg := ""
		for _, line := range lines {
			fs := strings.Split(line, ":")
			if len(fs) < 3 {
				continue
			}
			if fs[3][7:] == vm.infs[0] &&
				(sport == "0" || fs[0][5:] == sport) && (dport == "0" || fs[2][7:] == dport) {
				arg = "--remove-forward-port=" + line
				break
			}
		}
		if arg == "" {
			continue
		}

		cmd := exec.Command("firewall-cmd", arg)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println(err)
		}

		cmd = exec.Command("firewall-cmd", "--permanent", arg)
		if err := cmd.Run(); err != nil {
			fmt.Println(err)
		}
		return
	}
	fmt.Println("Can't find Rule")
}
