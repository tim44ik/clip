package outputprocessor

import (
	"context"
	"net/http"
	"time"
)

type DB interface {
	GetMaxRate() int
	Lookup(string) ([]string, error)
	Fetch(string, string) ([]*CVEInfo, error)
}

func NewDB(dbtype string, ctx context.Context) DB {
	switch dbtype {
	case "NVD":
		return &NVDClient{
			maxRate: 5,
			ctx:     ctx,
			http:    &http.Client{Timeout: 30 * time.Second},
		}
	default:
		return nil
	}
}
