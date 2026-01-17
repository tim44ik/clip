package main

import (
	"os"
	"os/exec"
)

func runIso(args []string) {
	cmd := exec.Command(args[2], args[3:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		println(err.Error())
		return
	}
}
