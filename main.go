package main

import core "smartpentestutility/core"

type CorePointer struct {
}

func main() {
	core.SpuAppInstance = core.CreateApp()
	core.SpuAppInstance.Window.ShowAndRun()
}
