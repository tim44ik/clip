package frontend

import (
	"clip/models/modules"
	"slices"
)

func (a *ClipWindow) deleteModule() {
	if a.selectedModule == a.modules.MainModule {
		return
	}
	a.modules.ChildModules = slices.DeleteFunc(
		a.modules.ChildModules,
		func(m *modules.Module) bool {
			return m == a.selectedModule
		})
	a.selectMainModule()
	a.refreshModuleGui()
}
