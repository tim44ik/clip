package core

func (a *ClipWindow) langmapInit() {
	a.langmap = make(map[string][]string)
	a.langmap["en"] =
		[]string{"Main", //0
			"Threads Number",                     //1
			"Make report          ",              //2
			"Edit",                               //3
			"Delete",                             //4
			"Load",                               //5
			"Load in new window",                 //6
			"Save",                               //7
			"Save as",                            //8
			"Begin scenario",                     //9
			"Break scenario",                     //10
			"Break scenario and make report",     //11
			"Change language",                    //12
			"Exit",                               //13
			"Add module",                         //14
			"Module",                             //15
			"Scenario is already running",        //16
			"Completed",                          //17
			"Scenario execution completed",       //18
			"Scenario was not started",           //19
			"Interrupted",                        //20
			"Scenario execution was interrupted", //21
			"Alter module name",                  //22
			"OK",                                 //23
			"Cancel",                             //24
			"Enter new module name",              //25
			"Add new module",                     //26
			"Cancelled",                          //27
			"Error occured while making report",  //28
			"Change language",                    //29
			"Apply",                              //30
			"Choose language",                    //31
			"View full output",                   //32
			"Full output",                        //33
			"Process output",                     //34
			"Creating report",                    //35
			"Queue is not declarated in module",  //36
			"No commands provided for execution in module", //37
			"Data format error in ",                        //38
			"Loading profile error",                        //39
			"Saving profile error",                         //40
			"Choose output file type",                      //41
			"Choose encryption type",                       //42
			"Enter password",                               //43
			"Encryption error",                             //44
			"Unknown cipher",                               //45
			"Password was not provided",                    //46
			"No encryption",                                //47
			"Choose database for vulnerabilities look up"}  //48

	a.langmap["ru"] =
		[]string{"Главная",
			"Количество потоков",
			"Сформировать отчёт   ",
			"Изменить",
			"Удалить",
			"Загрузить",
			"Загрузить в новом окне",
			"Сохранить",
			"Сохранить как",
			"Начать сценарий",
			"Прервать сценарий",
			"Прервать сценарий и сформировать отчёт",
			"Изменить язык",
			"Выйти",
			"Добавить модуль",
			"Модуль",
			"Сценарий уже запущен",
			"Выполнено",
			"Выполнение сценария окончено",
			"Сценарий не запущен",
			"Прервано",
			"Выполнение сценария было прервано",
			"Изменить название модуля",
			"OK",
			"Отмена",
			"Введите название",
			"Добавить новый модуль",
			"Отменено",
			"Ошибка при формировании отчета",
			"Изменить язык",
			"Применить",
			"Выберите язык",
			"Посмотреть весь вывод",
			"Весь вывод программы",
			"Обработать вывод",
			"Идет формирование отчета",
			"Номер очереди не задан в модуле",
			"Не заданы команды для исполнения в модуле",
			"Ошибка формата данных в ",
			"Ошибка загрузки профиля",
			"Ошибка сохранения профиля",
			"Выберите тип файла для сохранения",
			"Выберите тип шифрования",
			"Введите пароль",
			"Ошибка шифрования",
			"Неизвестный шифр",
			"Пароль не был предоставлен",
			"Без шифрования",
			"Выберите базу данных для поиска уязвимостей"}
}
