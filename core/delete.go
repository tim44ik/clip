package core

import "slices"

func Delete(a *SpuWindow) {
	if a.selectedModule == a.Modules.MainModule {
		return
	}
	a.Modules.ChildModules = slices.DeleteFunc(a.Modules.ChildModules, func(m *Module) bool {
		return m == a.selectedModule
	})
	a.selectMainModule()
	a.refreshModuleGui()
}
