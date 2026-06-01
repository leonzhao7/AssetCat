package domain

import (
	"testing"
	"time"
)

func TestNormalizeAssetUsesIPAsPrimaryDomain(t *testing.T) {
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	asset, err := NormalizeAsset(Asset{
		IPs: []IPRecord{{
			Address: "203.0.113.10",
			Ports: []PortRecord{{
				Port:     443,
				Protocol: "tcp",
				Service:  "https",
				Banner:   "nginx",
			}},
		}},
	}, now)
	if err != nil {
		t.Fatalf("NormalizeAsset returned error: %v", err)
	}
	if asset.PrimaryDomain != "203.0.113.10" {
		t.Fatalf("PrimaryDomain = %q, want ip alias", asset.PrimaryDomain)
	}
	if len(asset.Domains) != 1 || asset.Domains[0].Kind != DomainKindIPAlias {
		t.Fatalf("Domains = %#v, want one ip alias domain", asset.Domains)
	}
}

func TestNormalizeComponentRequiresProof(t *testing.T) {
	component := ComponentRecord{Name: "nginx"}
	err := NormalizeComponentRecord(&component, time.Now())
	if err == nil {
		t.Fatal("NormalizeComponentRecord succeeded without proof_url and response_content")
	}
}

func TestNormalizeRiskRequiresHTTPRequestEvidence(t *testing.T) {
	finding := RiskFinding{
		Title:    "exposed admin",
		Severity: SeverityHigh,
		URL:      "https://admin.example.com/",
		Request:  "GET / HTTP/1.1\r\nHost: admin.example.com\r\n\r\n",
		Response: "HTTP/1.1 200 OK\r\n\r\nadmin",
	}
	if err := NormalizeRiskFinding(&finding, time.Now()); err != nil {
		t.Fatalf("NormalizeRiskFinding returned error: %v", err)
	}
	if finding.ID == "" {
		t.Fatal("NormalizeRiskFinding did not generate stable id")
	}
}

func TestNormalizeAssetMovesAssetEvidenceToPrimaryDomain(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	asset, err := NormalizeAsset(Asset{
		PrimaryDomain: "example.com",
		IPs: []IPRecord{{
			Address: "203.0.113.10",
			Ports: []PortRecord{{
				Port:     443,
				Protocol: "tcp",
			}},
		}},
		Components: []ComponentRecord{{
			Name:            "nginx",
			ProofURL:        "https://example.com/",
			ResponseContent: "HTTP/1.1 200 OK\r\n\r\n",
		}},
	}, now)
	if err != nil {
		t.Fatalf("NormalizeAsset returned error: %v", err)
	}
	if len(asset.IPs) != 0 || len(asset.Components) != 0 {
		t.Fatalf("asset-level evidence = ips:%d components:%d, want moved to domain", len(asset.IPs), len(asset.Components))
	}
	if len(asset.Domains) != 1 || len(asset.Domains[0].IPs) != 1 || len(asset.Domains[0].Components) != 1 {
		t.Fatalf("domain evidence = %#v, want primary domain with ip and component", asset.Domains)
	}
}
