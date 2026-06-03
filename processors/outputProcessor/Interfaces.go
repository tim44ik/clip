package outputprocessor

import (
	"context"

	"gorm.io/gorm"
)

type DB interface {
	GetPData(string, string) ([]*CVEInfo, error)
	GetVulnerabilities(string) ([]*CVEInfo, error)
}

func NewDB(database *gorm.DB, dbtype string, ctx context.Context) DB {
	switch dbtype {
	case "NVD":
		return &NVDClient{
			ctx:      ctx,
			database: database,
		}
	default:
		return nil
	}
}
