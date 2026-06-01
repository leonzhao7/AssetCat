package domain

import "time"

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type DomainKind string

const (
	DomainKindPrimary   DomainKind = "primary"
	DomainKindSubdomain DomainKind = "subdomain"
	DomainKindIPAlias   DomainKind = "ip_alias"
)

type Asset struct {
	ID            string            `json:"id"`
	Name          string            `json:"name,omitempty"`
	PrimaryDomain string            `json:"primary_domain"`
	Domains       []DomainRecord    `json:"domains"`
	IPs           []IPRecord        `json:"ips"`
	Components    []ComponentRecord `json:"components"`
	Tags          []string          `json:"tags,omitempty"`
	Owner         string            `json:"owner,omitempty"`
	BusinessUnit  string            `json:"business_unit,omitempty"`
	Status        string            `json:"status,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

type DomainRecord struct {
	Name      string        `json:"name"`
	Kind      DomainKind    `json:"kind"`
	Risks     []RiskFinding `json:"risks,omitempty"`
	FirstSeen time.Time     `json:"first_seen"`
	LastSeen  time.Time     `json:"last_seen"`
}

type RiskFinding struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Severity     Severity  `json:"severity"`
	URL          string    `json:"url"`
	Request      string    `json:"request"`
	Response     string    `json:"response"`
	Description  string    `json:"description,omitempty"`
	Remediation  string    `json:"remediation,omitempty"`
	Status       string    `json:"status,omitempty"`
	CVE          string    `json:"cve,omitempty"`
	CWE          string    `json:"cwe,omitempty"`
	ComponentID  string    `json:"component_id,omitempty"`
	Confidence   float64   `json:"confidence,omitempty"`
	DiscoveredBy string    `json:"discovered_by,omitempty"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
}

type ComponentRecord struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Version         string            `json:"version,omitempty"`
	Category        string            `json:"category,omitempty"`
	ProofURL        string            `json:"proof_url"`
	ResponseContent string            `json:"response_content"`
	Confidence      float64           `json:"confidence,omitempty"`
	Source          string            `json:"source,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	FirstSeen       time.Time         `json:"first_seen"`
	LastSeen        time.Time         `json:"last_seen"`
}

type IPRecord struct {
	Address   string       `json:"address"`
	Ports     []PortRecord `json:"ports,omitempty"`
	ASN       string       `json:"asn,omitempty"`
	ISP       string       `json:"isp,omitempty"`
	Geo       string       `json:"geo,omitempty"`
	FirstSeen time.Time    `json:"first_seen"`
	LastSeen  time.Time    `json:"last_seen"`
}

type PortRecord struct {
	Port      int       `json:"port"`
	Protocol  string    `json:"protocol"`
	Service   string    `json:"service,omitempty"`
	Banner    string    `json:"banner,omitempty"`
	TLS       bool      `json:"tls,omitempty"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

type AssetStats struct {
	AssetID       string           `json:"asset_id"`
	PrimaryDomain string           `json:"primary_domain"`
	Domains       int              `json:"domains"`
	Subdomains    int              `json:"subdomains"`
	IPs           int              `json:"ips"`
	Ports         int              `json:"ports"`
	Components    int              `json:"components"`
	Risks         int              `json:"risks"`
	BySeverity    map[Severity]int `json:"by_severity"`
	LastUpdated   time.Time        `json:"last_updated,omitempty"`
}
