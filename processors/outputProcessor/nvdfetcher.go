package outputprocessor

import (
	"clip/models/nvd"
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type NVDClient struct {
	ctx      context.Context
	database *gorm.DB
}

func (n *NVDClient) GetPData(product, version string) ([]*CVEInfo, error) {
	var cpeList []nvd.CPE
	err := n.database.WithContext(n.ctx).
		Where("product = ? AND ver = ?", product, version).
		Find(&cpeList).Error
	if err != nil {
		return nil, fmt.Errorf("query CPE failed: %w", err)
	}
	if len(cpeList) == 0 {
		return nil, nil
	}

	cveMap := make(map[string]*CVEInfo)
	for _, cpe := range cpeList {
		var cveModels []nvd.CVE
		err := n.database.WithContext(n.ctx).
			Table("cpe_cve").
			Select("cve.*").
			Joins("JOIN cve ON cpe_cve.cve_id = cve.id").
			Where("cpe_cve.cpe_name = ?", cpe.CPE_name).
			Find(&cveModels).Error
		if err != nil {
			return nil, fmt.Errorf("query CVE for CPE %s failed: %w", cpe.CPE_name, err)
		}
		for _, cve := range cveModels {
			if _, exists := cveMap[cve.ID]; !exists {
				cveMap[cve.ID] = modelToCVEInfo(&cve)
			}
		}
	}

	result := make([]*CVEInfo, 0, len(cveMap))
	for _, info := range cveMap {
		result = append(result, info)
	}
	return result, nil
}

func (n *NVDClient) GetVulnerabilities(cveID string) ([]*CVEInfo, error) {
	var cveModel nvd.CVE
	err := n.database.WithContext(n.ctx).Where("id = ?", cveID).First(&cveModel).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("query CVE %s failed: %w", cveID, err)
	}
	return []*CVEInfo{modelToCVEInfo(&cveModel)}, nil
}

func modelToCVEInfo(cv *nvd.CVE) *CVEInfo {
	var links []string
	if cv.References != "" {
		links = strings.Split(cv.References, "\n")
	}
	return &CVEInfo{
		ID:          cv.ID,
		Description: cv.Description,
		Severity:    cv.Severity,
		Links:       links,
	}
}
