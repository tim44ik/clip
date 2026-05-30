package outputprocessor

import (
	"context"
)

type DB interface {
	GetPData(string, string) ([]*CVEInfo, error)
	GetVulnerabilities(string) ([]*CVEInfo, error)
}

func NewDB(dbtype string, ctx context.Context) DB {
	switch dbtype {
	case "NVD":
		return &NVDClient{
			ctx: ctx,
		}
	default:
		return nil
	}
}
