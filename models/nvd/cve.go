package nvd

type CVE struct {
	ID          string `gorm:"type:text;primaryKey;column:id"`
	Description string `gorm:"type:text;column:description"`
	Severity    string `gorm:"type:text;column:severity;index"`
	References  string `gorm:"type:text;column:references"`
}

func (CVE) TableName() string { return "cve" }
