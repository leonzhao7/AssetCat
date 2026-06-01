package domain

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	ErrValidation = errors.New("validation failed")
	domainLabelRx = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
)

func NormalizeAsset(in Asset, now time.Time) (Asset, error) {
	asset := in
	asset.PrimaryDomain = NormalizeDomain(asset.PrimaryDomain)

	if asset.PrimaryDomain == "" {
		if len(asset.IPs) == 0 {
			return Asset{}, wrapValidation("primary_domain is required when ips is empty")
		}
		asset.PrimaryDomain = strings.TrimSpace(asset.IPs[0].Address)
	}
	if err := validateDomainOrIP(asset.PrimaryDomain); err != nil {
		return Asset{}, err
	}
	if asset.ID == "" {
		asset.ID = StableID(asset.PrimaryDomain)
	}
	if asset.Status == "" {
		asset.Status = "active"
	}
	if asset.CreatedAt.IsZero() {
		asset.CreatedAt = now
	}
	asset.UpdatedAt = now

	asset.Domains = normalizeDomains(asset.Domains, asset.PrimaryDomain, now)
	if len(asset.Domains) == 0 {
		kind := DomainKindPrimary
		if net.ParseIP(asset.PrimaryDomain) != nil {
			kind = DomainKindIPAlias
		}
		asset.Domains = append(asset.Domains, DomainRecord{
			Name:      asset.PrimaryDomain,
			Kind:      kind,
			FirstSeen: now,
			LastSeen:  now,
		})
	}

	for i := range asset.Domains {
		if err := NormalizeDomainRecord(&asset.Domains[i], asset.PrimaryDomain, now); err != nil {
			return Asset{}, err
		}
	}
	for i := range asset.IPs {
		if err := NormalizeIPRecord(&asset.IPs[i], now); err != nil {
			return Asset{}, err
		}
	}
	for i := range asset.Components {
		if err := NormalizeComponentRecord(&asset.Components[i], now); err != nil {
			return Asset{}, err
		}
	}
	moveAssetEvidenceToPrimaryDomain(&asset)
	sortAsset(&asset)
	return asset, nil
}

func NormalizeDomainRecord(record *DomainRecord, primary string, now time.Time) error {
	record.Name = NormalizeDomain(record.Name)
	if record.Name == "" {
		return wrapValidation("domain name is required")
	}
	if err := validateDomainOrIP(record.Name); err != nil {
		return err
	}
	if record.Kind == "" {
		record.Kind = DomainKindSubdomain
		if record.Name == primary {
			record.Kind = DomainKindPrimary
		}
		if net.ParseIP(record.Name) != nil {
			record.Kind = DomainKindIPAlias
		}
	}
	if record.FirstSeen.IsZero() {
		record.FirstSeen = now
	}
	record.LastSeen = now
	for i := range record.Risks {
		if err := NormalizeRiskFinding(&record.Risks[i], now); err != nil {
			return err
		}
	}
	for i := range record.IPs {
		if err := NormalizeIPRecord(&record.IPs[i], now); err != nil {
			return err
		}
	}
	for i := range record.Components {
		if err := NormalizeComponentRecord(&record.Components[i], now); err != nil {
			return err
		}
	}
	slices.SortFunc(record.Risks, func(a, b RiskFinding) int {
		return strings.Compare(a.ID, b.ID)
	})
	slices.SortFunc(record.IPs, func(a, b IPRecord) int {
		return strings.Compare(a.Address, b.Address)
	})
	slices.SortFunc(record.Components, func(a, b ComponentRecord) int {
		return strings.Compare(a.ID, b.ID)
	})
	return nil
}

func NormalizeRiskFinding(finding *RiskFinding, now time.Time) error {
	finding.URL = strings.TrimSpace(finding.URL)
	finding.Title = strings.TrimSpace(finding.Title)
	finding.Request = strings.TrimSpace(finding.Request)
	finding.Response = strings.TrimSpace(finding.Response)
	finding.Status = strings.TrimSpace(finding.Status)
	if finding.Title == "" {
		return wrapValidation("risk title is required")
	}
	if finding.URL == "" {
		return wrapValidation("risk url is required")
	}
	parsed, err := url.ParseRequestURI(finding.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return wrapValidation("risk url must be an absolute http(s) url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return wrapValidation("risk url scheme must be http or https")
	}
	if finding.Request == "" {
		return wrapValidation("risk request is required")
	}
	if finding.Response == "" {
		return wrapValidation("risk response is required")
	}
	if finding.Severity == "" {
		finding.Severity = SeverityInfo
	}
	if !ValidSeverity(finding.Severity) {
		return wrapValidation("invalid risk severity")
	}
	if finding.Status == "" {
		finding.Status = "open"
	}
	if finding.ID == "" {
		finding.ID = StableID(finding.Title + "|" + finding.URL + "|" + finding.Request)
	}
	if finding.FirstSeen.IsZero() {
		finding.FirstSeen = now
	}
	finding.LastSeen = now
	return nil
}

func NormalizeComponentRecord(component *ComponentRecord, now time.Time) error {
	component.Name = strings.TrimSpace(component.Name)
	component.ProofURL = strings.TrimSpace(component.ProofURL)
	component.ResponseContent = strings.TrimSpace(component.ResponseContent)
	if component.Name == "" {
		return wrapValidation("component name is required")
	}
	if component.ProofURL == "" {
		return wrapValidation("component proof_url is required")
	}
	parsed, err := url.ParseRequestURI(component.ProofURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return wrapValidation("component proof_url must be an absolute http(s) url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return wrapValidation("component proof_url scheme must be http or https")
	}
	if component.ResponseContent == "" {
		return wrapValidation("component response_content is required")
	}
	if component.ID == "" {
		component.ID = StableID(component.Name + "|" + component.Version + "|" + component.ProofURL)
	}
	if component.FirstSeen.IsZero() {
		component.FirstSeen = now
	}
	component.LastSeen = now
	return nil
}

func NormalizeIPRecord(record *IPRecord, now time.Time) error {
	record.Address = strings.TrimSpace(record.Address)
	if net.ParseIP(record.Address) == nil {
		return wrapValidation("ip address is invalid")
	}
	if record.FirstSeen.IsZero() {
		record.FirstSeen = now
	}
	record.LastSeen = now
	for i := range record.Ports {
		if err := NormalizePortRecord(&record.Ports[i], now); err != nil {
			return err
		}
	}
	slices.SortFunc(record.Ports, func(a, b PortRecord) int {
		if a.Port == b.Port {
			return strings.Compare(a.Protocol, b.Protocol)
		}
		return a.Port - b.Port
	})
	return nil
}

func NormalizePortRecord(port *PortRecord, now time.Time) error {
	port.Protocol = strings.ToLower(strings.TrimSpace(port.Protocol))
	if port.Port < 1 || port.Port > 65535 {
		return wrapValidation("port must be between 1 and 65535")
	}
	if port.Protocol == "" {
		port.Protocol = "tcp"
	}
	if port.Protocol != "tcp" && port.Protocol != "udp" {
		return wrapValidation("port protocol must be tcp or udp")
	}
	if port.FirstSeen.IsZero() {
		port.FirstSeen = now
	}
	port.LastSeen = now
	return nil
}

func NormalizeDomain(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimSuffix(value, ".")
	return value
}

func StableID(value string) string {
	sum := sha1.Sum([]byte(strings.ToLower(strings.TrimSpace(value))))
	return hex.EncodeToString(sum[:])[:16]
}

func ValidSeverity(severity Severity) bool {
	switch severity {
	case SeverityInfo, SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical:
		return true
	default:
		return false
	}
}

func MergeDomain(existing DomainRecord, incoming DomainRecord, primary string, now time.Time) (DomainRecord, error) {
	if existing.Name == "" {
		if err := NormalizeDomainRecord(&incoming, primary, now); err != nil {
			return DomainRecord{}, err
		}
		return incoming, nil
	}
	existing.Kind = incoming.Kind
	if existing.Kind == "" {
		existing.Kind = DomainKindSubdomain
	}
	if existing.FirstSeen.IsZero() {
		existing.FirstSeen = incoming.FirstSeen
	}
	existing.LastSeen = now
	seen := make(map[string]RiskFinding, len(existing.Risks)+len(incoming.Risks))
	for _, risk := range existing.Risks {
		seen[risk.ID] = risk
	}
	for _, risk := range incoming.Risks {
		if err := NormalizeRiskFinding(&risk, now); err != nil {
			return DomainRecord{}, err
		}
		seen[risk.ID] = risk
	}
	existing.Risks = existing.Risks[:0]
	for _, risk := range seen {
		existing.Risks = append(existing.Risks, risk)
	}
	ips := make(map[string]IPRecord, len(existing.IPs)+len(incoming.IPs))
	for _, ip := range existing.IPs {
		ips[ip.Address] = ip
	}
	for _, ip := range incoming.IPs {
		merged, err := MergeIP(ips[ip.Address], ip, now)
		if err != nil {
			return DomainRecord{}, err
		}
		ips[merged.Address] = merged
	}
	existing.IPs = existing.IPs[:0]
	for _, ip := range ips {
		existing.IPs = append(existing.IPs, ip)
	}
	components := make(map[string]ComponentRecord, len(existing.Components)+len(incoming.Components))
	for _, component := range existing.Components {
		components[component.ID] = component
	}
	for _, component := range incoming.Components {
		merged, err := MergeComponent(components[component.ID], component, now)
		if err != nil {
			return DomainRecord{}, err
		}
		components[merged.ID] = merged
	}
	existing.Components = existing.Components[:0]
	for _, component := range components {
		existing.Components = append(existing.Components, component)
	}
	if err := NormalizeDomainRecord(&existing, primary, now); err != nil {
		return DomainRecord{}, err
	}
	return existing, nil
}

func MergeIP(existing IPRecord, incoming IPRecord, now time.Time) (IPRecord, error) {
	if existing.Address == "" {
		if err := NormalizeIPRecord(&incoming, now); err != nil {
			return IPRecord{}, err
		}
		return incoming, nil
	}
	if existing.FirstSeen.IsZero() {
		existing.FirstSeen = incoming.FirstSeen
	}
	existing.ASN = firstNonEmpty(incoming.ASN, existing.ASN)
	existing.ISP = firstNonEmpty(incoming.ISP, existing.ISP)
	existing.Geo = firstNonEmpty(incoming.Geo, existing.Geo)
	existing.LastSeen = now

	ports := make(map[string]PortRecord, len(existing.Ports)+len(incoming.Ports))
	for _, port := range existing.Ports {
		ports[portKey(port)] = port
	}
	for _, port := range incoming.Ports {
		if err := NormalizePortRecord(&port, now); err != nil {
			return IPRecord{}, err
		}
		ports[portKey(port)] = port
	}
	existing.Ports = existing.Ports[:0]
	for _, port := range ports {
		existing.Ports = append(existing.Ports, port)
	}
	if err := NormalizeIPRecord(&existing, now); err != nil {
		return IPRecord{}, err
	}
	return existing, nil
}

func MergeComponent(existing ComponentRecord, incoming ComponentRecord, now time.Time) (ComponentRecord, error) {
	if existing.ID == "" {
		if err := NormalizeComponentRecord(&incoming, now); err != nil {
			return ComponentRecord{}, err
		}
		return incoming, nil
	}
	incoming.ID = firstNonEmpty(incoming.ID, existing.ID)
	incoming.FirstSeen = existing.FirstSeen
	if incoming.FirstSeen.IsZero() {
		incoming.FirstSeen = now
	}
	incoming.LastSeen = now
	if err := NormalizeComponentRecord(&incoming, now); err != nil {
		return ComponentRecord{}, err
	}
	return incoming, nil
}

func Stats(asset Asset) AssetStats {
	result := AssetStats{
		AssetID:       asset.ID,
		PrimaryDomain: asset.PrimaryDomain,
		Domains:       len(asset.Domains),
		BySeverity:    make(map[Severity]int),
		LastUpdated:   asset.UpdatedAt,
	}
	for _, record := range asset.Domains {
		if record.Kind == DomainKindSubdomain {
			result.Subdomains++
		}
		result.IPs += len(record.IPs)
		result.Components += len(record.Components)
		for _, ip := range record.IPs {
			result.Ports += len(ip.Ports)
		}
		for _, risk := range record.Risks {
			result.Risks++
			result.BySeverity[risk.Severity]++
		}
	}
	return result
}

func FlattenRisks(asset Asset) []RiskFinding {
	var risks []RiskFinding
	for _, domain := range asset.Domains {
		risks = append(risks, domain.Risks...)
	}
	slices.SortFunc(risks, func(a, b RiskFinding) int {
		if a.Severity == b.Severity {
			return strings.Compare(a.ID, b.ID)
		}
		return severityRank(b.Severity) - severityRank(a.Severity)
	})
	return risks
}

func DomainStats(record DomainRecord) AssetStats {
	stats := AssetStats{
		PrimaryDomain: record.Name,
		Domains:       1,
		IPs:           len(record.IPs),
		Components:    len(record.Components),
		BySeverity:    make(map[Severity]int),
		LastUpdated:   record.LastSeen,
	}
	for _, ip := range record.IPs {
		stats.Ports += len(ip.Ports)
	}
	for _, risk := range record.Risks {
		stats.Risks++
		stats.BySeverity[risk.Severity]++
	}
	return stats
}

func normalizeDomains(domains []DomainRecord, primary string, now time.Time) []DomainRecord {
	seen := make(map[string]DomainRecord, len(domains)+1)
	if primary != "" {
		kind := DomainKindPrimary
		if net.ParseIP(primary) != nil {
			kind = DomainKindIPAlias
		}
		seen[primary] = DomainRecord{Name: primary, Kind: kind, FirstSeen: now, LastSeen: now}
	}
	for _, record := range domains {
		name := NormalizeDomain(record.Name)
		if name == "" {
			continue
		}
		if prior, ok := seen[name]; ok {
			if prior.FirstSeen.IsZero() {
				prior.FirstSeen = record.FirstSeen
			}
			prior.LastSeen = now
			prior.Risks = append(prior.Risks, record.Risks...)
			prior.IPs = append(prior.IPs, record.IPs...)
			prior.Components = append(prior.Components, record.Components...)
			if record.Kind != "" {
				prior.Kind = record.Kind
			}
			seen[name] = prior
			continue
		}
		record.Name = name
		seen[name] = record
	}
	out := make([]DomainRecord, 0, len(seen))
	for _, record := range seen {
		out = append(out, record)
	}
	return out
}

func validateDomainOrIP(value string) error {
	if net.ParseIP(value) != nil {
		return nil
	}
	if len(value) > 253 {
		return wrapValidation("domain is too long")
	}
	labels := strings.Split(value, ".")
	if len(labels) < 2 {
		return wrapValidation("domain must contain at least two labels, or be an ip address")
	}
	for _, label := range labels {
		if !domainLabelRx.MatchString(label) {
			return wrapValidation("domain contains invalid label")
		}
	}
	return nil
}

func sortAsset(asset *Asset) {
	slices.SortFunc(asset.Domains, func(a, b DomainRecord) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(asset.IPs, func(a, b IPRecord) int {
		return strings.Compare(a.Address, b.Address)
	})
	slices.SortFunc(asset.Components, func(a, b ComponentRecord) int {
		return strings.Compare(a.ID, b.ID)
	})
}

func moveAssetEvidenceToPrimaryDomain(asset *Asset) {
	if len(asset.IPs) == 0 && len(asset.Components) == 0 {
		return
	}
	for i := range asset.Domains {
		if asset.Domains[i].Name != asset.PrimaryDomain {
			continue
		}
		now := asset.UpdatedAt
		for _, ip := range asset.IPs {
			merged, err := MergeIP(findIP(asset.Domains[i].IPs, ip.Address), ip, now)
			if err == nil {
				asset.Domains[i].IPs = upsertIP(asset.Domains[i].IPs, merged)
			}
		}
		for _, component := range asset.Components {
			merged, err := MergeComponent(findComponent(asset.Domains[i].Components, component.ID), component, now)
			if err == nil {
				asset.Domains[i].Components = upsertComponent(asset.Domains[i].Components, merged)
			}
		}
		asset.IPs = nil
		asset.Components = nil
		return
	}
}

func findIP(records []IPRecord, address string) IPRecord {
	for _, record := range records {
		if record.Address == address {
			return record
		}
	}
	return IPRecord{}
}

func upsertIP(records []IPRecord, record IPRecord) []IPRecord {
	for i := range records {
		if records[i].Address == record.Address {
			records[i] = record
			return records
		}
	}
	return append(records, record)
}

func findComponent(records []ComponentRecord, id string) ComponentRecord {
	for _, record := range records {
		if record.ID == id {
			return record
		}
	}
	return ComponentRecord{}
}

func upsertComponent(records []ComponentRecord, record ComponentRecord) []ComponentRecord {
	for i := range records {
		if records[i].ID == record.ID {
			records[i] = record
			return records
		}
	}
	return append(records, record)
}

func portKey(port PortRecord) string {
	return strconv.Itoa(port.Port) + "/" + port.Protocol
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func severityRank(severity Severity) int {
	switch severity {
	case SeverityCritical:
		return 5
	case SeverityHigh:
		return 4
	case SeverityMedium:
		return 3
	case SeverityLow:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

func wrapValidation(message string) error {
	return fmt.Errorf("%w: %s", ErrValidation, message)
}
