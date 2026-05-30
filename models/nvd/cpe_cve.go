package nvd

type CPE_CVE struct {
	CPE_name string `gorm:"type:text;primaryKey;column:cpe_name;constraint:OnDelete:CASCADE"`
	CVE_name string `gorm:"type:text;primaryKey;column:cve_id;constraint:OnDelete:CASCADE"`
}

func (CPE_CVE) TableName() string { return "cpe_cve" }
