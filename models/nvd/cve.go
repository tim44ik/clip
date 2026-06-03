package nvd

type CVE struct {
	ID          string `gorm:"type:text;primaryKey;column:id"`
	Description string `gorm:"type:text;column:descr"`
	Severity    string `gorm:"type:text;column:severity;index:idx_cve_severity"`
	References  string `gorm:"type:text;column:refs"`
}

func (CVE) TableName() string { return "cve" }
