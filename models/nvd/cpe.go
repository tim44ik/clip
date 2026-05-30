package nvd

type CPE struct {
	CPE_name string `gorm:"type:text;primaryKey;column:cpe_name"`
	Product  string `gorm:"type:text;column:product;index"`
	Version  string `gorm:"type:text;column:version;index"`
}

func (CPE) TableName() string { return "cpe" }
