package core

func LangmapInit(a *SpuWindow) {
	a.langmap = make(map[string][]string)
	a.langmap["English"] =
		[]string{"Main",
			"Threads Number",
			"Make PDF report       ",
			"Edit", "Delete",
			"Load", "Load in new window",
			"Save", "Save as",
			"Begin scenario", "Break scenario",
			"Break scenario and make PDF",
			"Change language", "Exit",
			"Add module", "Module", "Scenario is already started",
			"Completed", "Scenario execution completed",
			"Scenario was not started",
			"Interrupted", "Scenario execution was interrupted",
			"Alter module name", "OK", "Cancel",
			"Enter new module name",
			"Add new module", "Cancelled",
			"Error occured while making PDF",
			"Change language", "Apply",
			"Choose language", "View full output",
			"Process output", "Choose options for PDF report"}

	a.langmap["Русский"] =
		[]string{"Главная",
			"Количество потоков",
			"Сформировать PDF отчёт",
			"Изменить", "Удалить",
			"Загрузить",
			"Загрузить в новом окне",
			"Сохранить", "Сохранить как",
			"Начать сценарий",
			"Прервать сценарий",
			"Прервать сценарий и сформировать отчёт в PDF",
			"Изменить язык", "Выйти",
			"Добавить модуль", "Модуль",
			"Сценарий уже запущен",
			"Выполнено",
			"Выполнение сценария окончено",
			"Сценарий не запущен",
			"Прервано",
			"Выполнение сценария было прервано",
			"Изменить название модуля", "OK",
			"Отмена", "Введите название",
			"Добавить новый модуль", "Отменено",
			"Ошибка при создании PDF",
			"Изменить язык", "Применить",
			"Выберите язык", "Посмотреть весь вывод",
			"Обработать вывод", "Выберите опции для PDF отчета"}

}
