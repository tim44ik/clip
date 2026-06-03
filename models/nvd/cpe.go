package nvd

type CPE struct {
	CPE_name string `gorm:"type:text;primaryKey;column:cpe_name"`
	Vendor   string `gorm:"type:text;column:vendor;not null;index:idx_cpe_vendor_product_version,priority:1"`
	Product  string `gorm:"type:text;column:product;index:idx_cpe_product_version,priority:1;index:idx_cpe_vendor_product_version,priority:2"`
	Version  string `gorm:"type:text;column:ver;index:idx_cpe_product_version,priority:2;index:idx_cpe_vendor_product_version,priority:3"`
}

func (CPE) TableName() string { return "cpe" }
