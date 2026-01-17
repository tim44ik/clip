package modules

type Module struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Output  string `json:"-"`
	MakePDF struct {
		Do      bool `json:"do"`
		Process bool `json:"process"`
	} `json:"makePDF"`
}
