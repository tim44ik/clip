package core

// modules := []*Module{{Name: "Главная", Content: input.Text},
// 	{Name: "Добавить модуль", Content: ""},
// 	{Name: "NMAP", Content: "nmap"}}

// leftPanel := container.NewVBox()
// leftScroll := container.NewVScroll(leftPanel)
// leftScroll.SetMinSize(fyne.NewSize(180, 0))
// var UpdateLeftPanel func()
// UpdateLeftPanel = func() {
// 	modules = slices.DeleteFunc(modules, func(module *Module) bool {
// 		return module.Name == "Добавить модуль"
// 	})
// 	modules = append(modules, &Module{Name: "Добавить модуль", Content: ""})
// 	leftPanel.Objects = nil
// 	oldInput := input.Text
// 	for _, item := range modules {
// 		btn := widget.NewButton(item.Name, func() {
// 			switch item.Name {
// 			case "Главная":
// 				title.Text = ("Информация об изучаемом объекте:")
// 				input.SetText(item.Content)
// 				confirmCheck.Show()
// 				threadInput.Show()
// 				submitBtn.Text = "Отправить"
// 				submitBtn.OnTapped = (func() {
// 					if oldInput != input.Text {
// 						input.Text = strings.ReplaceAll(input.Text, oldInput, "")
// 						for _, jtem := range modules {
// 							if (jtem.Name != "Главная") && (jtem.Name != "Добавить модуль") {
// 								jtem.Content = jtem.Content + input.Text
// 							}
// 						}
// 						oldInput += input.Text
// 					} else {
// 						for _, jtem := range modules {
// 							if (jtem.Name != "Главная") && (jtem.Name != "Добавить модуль") {
// 								jtem.Content = jtem.Content + input.Text
// 							}
// 						}
// 					}
// 				})
// 			case "Добавить модуль":
// 				AddModulePage(myApp, func(newModule *Module) {
// 					modules = append(modules, newModule)
// 					UpdateLeftPanel()
// 					leftPanel.Refresh()
// 				})
// 			default:
// 				title.Text = ("Введите запрос" + item.Name)
// 				input.SetText(item.Content)
// 				submitBtn.Text = "Сохранить"
// 				submitBtn.OnTapped = (func() {
// 					fmt.Println("Сохранено")
// 					item.Content = input.Text
// 					fmt.Println(item.Content)
// 				})

// 				confirmCheck.Hide()
// 				threadInput.Hide()

// 			}
// 			submitBtn.Refresh()
// 			input.Refresh()
// 			title.Refresh()
// 		})
// 		leftPanel.Add(btn)
// 	}

// 	leftPanel.Refresh()
// }
// UpdateLeftPanel()
