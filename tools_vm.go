package main

import (
	"fmt"
	"github.com/urfave/cli"
	"os"
	"os/exec"
	"strings"
)

var sshCmd = cli.Command{
	Name:      "ssh",
	Category:  "tools",
	Aliases:   []string{"s"},
	Usage:     "ssh to virtual machine",
	ArgsUsage: "vmName",
	Before:    checkArgs,
	Action:    sshVm,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "regexp,r",
			Usage: "Use regular expression match",
		},
	},
}

func checkArgs(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("No name or ip")
	}
	return nil
}

func sshVm(c *cli.Context) {
	method := 0
	if c.Bool("regexp") {
		method = 1
	}
	name := c.Args().First()
	virtMachines := getVms(nil, method)
	for _, vm := range virtMachines {
		matched := matchName(vm.name, []string{name}, method)
		if !matched {
			continue
		}
		if len(vm.infs) < 1 {
			continue
		}
		if matched || vm.infs[0] == name {
			fmt.Printf("login vm: %s/%s\n", vm.name, vm.infs[0])
			cmd := exec.Command("ssh", vm.infs[0])
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Println(err)
			}
			return
		}
	}
	fmt.Println("Can't find machine")
}

var cpCmd = cli.Command{
	Name:      "cp",
	Category:  "tools",
	Usage:     "scp file/dir to/from virtual machine",
	ArgsUsage: "[vmName:]/path [vmName:]/path",
	Before: func(c *cli.Context) error {
		if c.NArg() < 2 {
			return fmt.Errorf("invalid parameters")
		}
		return nil
	},
	Action: cpVm,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "regexp,r",
			Usage: "Use regular expression match",
		},
	},
}

func cpVm(c *cli.Context) {
	method := 0
	if c.Bool("regexp") {
		method = 1
	}

	path1 := c.Args().First()
	path2 := c.Args().Get(1)

	dir := "fromVm"
	vmAndPath := strings.Split(path1, ":")
	hostPath := path2
	if len(vmAndPath) < 2 {
		vmAndPath = strings.Split(path2, ":")
		if len(vmAndPath) < 2 {
			fmt.Println("invalid vm file path: c.Args().First()")
			return
		}
		dir = "fromHost"
		hostPath = path1
	}

	vmName := vmAndPath[0]
	vmPath := vmAndPath[1]
	virtMachines := getVms(nil, method)
	for _, vm := range virtMachines {
		matched := matchName(vm.name, []string{vmName}, method)
		if !matched {
			continue
		}
		if len(vm.infs) < 1 {
			if matched {
				fmt.Printf("Can't find %s's ip\n", vmName)
				return
			}
			continue
		}
		if matched || vm.infs[0] == vmName {
			cmd := exec.Command("scp", "-r", vm.infs[0]+":"+vmPath, hostPath)
			if dir == "fromHost" {
				cmd = exec.Command("scp", "-r", hostPath, vm.infs[0]+":"+vmPath)
			}
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Println(err)
			}
			return
		}
	}
	fmt.Println("Can't find machine: " + vmName)
}
