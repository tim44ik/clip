package main

import (
	"os"
	"os/exec"
)

func runIso(args []string) {
	cmd := exec.Command("cmd.exe", args[2:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		println(err.Error())
		return
	}
}
