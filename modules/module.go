package modules

type Module struct {
	Name       string `json:"name"`
	Content    string `json:"content"`
	Output     string `json:"-"`
	MakeReport struct {
		Do      bool `json:"do"`
		Process bool `json:"process"`
	} `json:"makeReport"`
}

type ClipModules struct {
	MainModule   *Module   `json:"mainModule"`
	ChildModules []*Module `json:"childModules"`
	CurrentLang  string    `json:"currentLang"`
}

func CreateModule(name, content string) *Module {
	return &Module{Name: name, Content: content}
}

func (m *Module) AlterName(name string) {
	m.Name = name
}
