package main

import (
	"fmt"
	"github.com/urfave/cli"
	"os"
	"os/exec"
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
