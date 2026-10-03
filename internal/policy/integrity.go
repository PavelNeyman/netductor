package policy

// IntegrityIssue is a soft problem (unknown service id, empty endpoints).
type IntegrityIssue struct {
	Subject string `json:"subject"`
	Kind    string `json:"kind"` // user|edge|catalog
	Detail  string `json:"detail"`
}

// CheckCatalogIntegrity warns on empty internal endpoints.
func CheckCatalogIntegrity(cat *Catalog) []IntegrityIssue {
	if cat == nil {
		return nil
	}
	var out []IntegrityIssue
	for _, s := range cat.Services {
		if s.Disabled || s.Kind != KindInternal {
			continue
		}
		if len(s.Endpoints) == 0 {
			out = append(out, IntegrityIssue{Subject: s.ID, Kind: "catalog", Detail: "no endpoints"})
			continue
		}
		for _, ep := range s.Endpoints {
			if ep.Port <= 0 {
				out = append(out, IntegrityIssue{Subject: s.ID, Kind: "catalog", Detail: "endpoint port unset"})
			}
		}
	}
	return out
}

// CheckPolicyAgainstCatalog returns issues when subject references unknown service ids.
func CheckPolicyAgainstCatalog(subject, kind string, p AccessPolicy, cat *Catalog) []IntegrityIssue {
	p.Normalize()
	if cat == nil || p.ServicesMode == "all" {
		return nil
	}
	if err := p.Validate(cat); err != nil {
		return []IntegrityIssue{{Subject: subject, Kind: kind, Detail: err.Error()}}
	}
	return nil
}
