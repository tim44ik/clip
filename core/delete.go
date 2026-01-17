package core

import (
	"clip/modules"
	"slices"
)

func Delete(a *ClipWindow) {
	if a.selectedModule == a.Modules.MainModule {
		return
	}
	a.Modules.ChildModules = slices.DeleteFunc(a.Modules.ChildModules, func(m *modules.Module) bool {
		return m == a.selectedModule
	})
	a.selectMainModule()
	a.refreshModuleGui()
}
