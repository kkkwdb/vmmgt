package main

import (
	"fmt"
	"github.com/urfave/cli"
	"os"
	"os/exec"
	"strings"
)

var sshCmd = cli.Command{
	Name:     "ssh",
	Category: "tools",
	Aliases:  []string{"s"},
	Usage:    "ssh to virtual machine",
	Before:   checkArgs,
	Action:   sshVm,
}

func checkArgs(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("No name or ip")
	}
	return nil
}

func sshVm(c *cli.Context) {
	name := c.Args().First()
	virtMachines := getVms("[]")
	for _, vm := range virtMachines {
		if len(vm.infs) < 1 {
			continue
		}
		if vm.name == name || vm.infs[0] == name {
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
	Name:     "cp",
	Category: "tools",
	Usage:    "scp file/dir to/from virtual machine",
	Before: func(c *cli.Context) error {
		if c.NArg() < 2 {
			return fmt.Errorf("invalid parameters")
		}
		return nil
	},
	Action: cpVm,
}

func cpVm(c *cli.Context) {
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
	virtMachines := getVms("[]")
	for _, vm := range virtMachines {
		if len(vm.infs) < 1 {
			continue
		}
		if vm.name == vmName || vm.infs[0] == vmName {
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
	fmt.Println("Can't find machine")
}
