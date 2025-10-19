package main

import core "smartpentestutility/core"

func main() {
	core.SpuAppInstance = core.CreateApp()
	core.SpuAppInstance.Window.ShowAndRun()
}
