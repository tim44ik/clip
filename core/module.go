package core

type Module struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	output  string
	MakePDF struct {
		Do      bool `json:"do"`
		Process bool `json:"process"`
	} `json:"makePDF"`
}
