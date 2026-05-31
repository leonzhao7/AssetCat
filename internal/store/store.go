package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"asset-risk-system/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	mu     sync.RWMutex
	path   string
	assets map[string]domain.Asset
	now    func() time.Time
}

func New(path string) (*Store, error) {
	store := &Store{
		path:   path,
		assets: make(map[string]domain.Asset),
		now:    time.Now,
	}
	if path == "" {
		return store, nil
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) List() []domain.Asset {
	s.mu.RLock()
	defer s.mu.RUnlock()

	assets := make([]domain.Asset, 0, len(s.assets))
	for _, asset := range s.assets {
		assets = append(assets, cloneAsset(asset))
	}
	slices.SortFunc(assets, func(a, b domain.Asset) int {
		return strings.Compare(a.PrimaryDomain, b.PrimaryDomain)
	})
	return assets
}

func (s *Store) Get(id string) (domain.Asset, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	asset, ok := s.assets[id]
	if !ok {
		return domain.Asset{}, false
	}
	return cloneAsset(asset), true
}

func (s *Store) Upsert(asset domain.Asset) (domain.Asset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	normalized, err := domain.NormalizeAsset(asset, now)
	if err != nil {
		return domain.Asset{}, err
	}
	if existing, ok := s.assets[normalized.ID]; ok {
		normalized.CreatedAt = existing.CreatedAt
		normalized, err = mergeAsset(existing, normalized, now)
		if err != nil {
			return domain.Asset{}, err
		}
	}
	s.assets[normalized.ID] = normalized
	return cloneAsset(normalized), s.saveLocked()
}

func (s *Store) Replace(id string, asset domain.Asset) (domain.Asset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.assets[id]
	if !ok {
		return domain.Asset{}, ErrNotFound
	}
	asset.ID = id
	if asset.CreatedAt.IsZero() {
		asset.CreatedAt = existing.CreatedAt
	}
	normalized, err := domain.NormalizeAsset(asset, s.now().UTC())
	if err != nil {
		return domain.Asset{}, err
	}
	s.assets[id] = normalized
	return cloneAsset(normalized), s.saveLocked()
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.assets[id]; !ok {
		return ErrNotFound
	}
	delete(s.assets, id)
	return s.saveLocked()
}

func (s *Store) AddDomain(assetID string, record domain.DomainRecord) (domain.Asset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	asset, ok := s.assets[assetID]
	if !ok {
		return domain.Asset{}, ErrNotFound
	}
	now := s.now().UTC()
	if err := domain.NormalizeDomainRecord(&record, asset.PrimaryDomain, now); err != nil {
		return domain.Asset{}, err
	}
	replaced := false
	for i := range asset.Domains {
		if asset.Domains[i].Name == record.Name {
			merged, err := domain.MergeDomain(asset.Domains[i], record, asset.PrimaryDomain, now)
			if err != nil {
				return domain.Asset{}, err
			}
			asset.Domains[i] = merged
			replaced = true
			break
		}
	}
	if !replaced {
		asset.Domains = append(asset.Domains, record)
	}
	asset.UpdatedAt = now
	asset, err := domain.NormalizeAsset(asset, now)
	if err != nil {
		return domain.Asset{}, err
	}
	s.assets[assetID] = asset
	return cloneAsset(asset), s.saveLocked()
}

func (s *Store) AddIP(assetID string, record domain.IPRecord) (domain.Asset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	asset, ok := s.assets[assetID]
	if !ok {
		return domain.Asset{}, ErrNotFound
	}
	now := s.now().UTC()
	if err := domain.NormalizeIPRecord(&record, now); err != nil {
		return domain.Asset{}, err
	}
	replaced := false
	for i := range asset.IPs {
		if asset.IPs[i].Address == record.Address {
			merged, err := domain.MergeIP(asset.IPs[i], record, now)
			if err != nil {
				return domain.Asset{}, err
			}
			asset.IPs[i] = merged
			replaced = true
			break
		}
	}
	if !replaced {
		asset.IPs = append(asset.IPs, record)
	}
	asset.UpdatedAt = now
	asset, err := domain.NormalizeAsset(asset, now)
	if err != nil {
		return domain.Asset{}, err
	}
	s.assets[assetID] = asset
	return cloneAsset(asset), s.saveLocked()
}

func (s *Store) AddComponent(assetID string, component domain.ComponentRecord) (domain.Asset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	asset, ok := s.assets[assetID]
	if !ok {
		return domain.Asset{}, ErrNotFound
	}
	now := s.now().UTC()
	if err := domain.NormalizeComponentRecord(&component, now); err != nil {
		return domain.Asset{}, err
	}
	replaced := false
	for i := range asset.Components {
		if asset.Components[i].ID == component.ID {
			merged, err := domain.MergeComponent(asset.Components[i], component, now)
			if err != nil {
				return domain.Asset{}, err
			}
			asset.Components[i] = merged
			replaced = true
			break
		}
	}
	if !replaced {
		asset.Components = append(asset.Components, component)
	}
	asset.UpdatedAt = now
	asset, err := domain.NormalizeAsset(asset, now)
	if err != nil {
		return domain.Asset{}, err
	}
	s.assets[assetID] = asset
	return cloneAsset(asset), s.saveLocked()
}

func (s *Store) AddRisk(assetID string, domainName string, finding domain.RiskFinding) (domain.Asset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	asset, ok := s.assets[assetID]
	if !ok {
		return domain.Asset{}, ErrNotFound
	}
	now := s.now().UTC()
	domainName = domain.NormalizeDomain(domainName)
	if domainName == "" {
		return domain.Asset{}, fmt.Errorf("%w: domain name is required", domain.ErrValidation)
	}
	if err := domain.NormalizeRiskFinding(&finding, now); err != nil {
		return domain.Asset{}, err
	}

	found := false
	for i := range asset.Domains {
		if asset.Domains[i].Name != domainName {
			continue
		}
		found = true
		replaced := false
		for j := range asset.Domains[i].Risks {
			if asset.Domains[i].Risks[j].ID == finding.ID {
				asset.Domains[i].Risks[j] = finding
				replaced = true
				break
			}
		}
		if !replaced {
			asset.Domains[i].Risks = append(asset.Domains[i].Risks, finding)
		}
		asset.Domains[i].LastSeen = now
		break
	}
	if !found {
		record := domain.DomainRecord{
			Name:      domainName,
			Kind:      domain.DomainKindSubdomain,
			Risks:     []domain.RiskFinding{finding},
			FirstSeen: now,
			LastSeen:  now,
		}
		if domainName == asset.PrimaryDomain {
			record.Kind = domain.DomainKindPrimary
		}
		if err := domain.NormalizeDomainRecord(&record, asset.PrimaryDomain, now); err != nil {
			return domain.Asset{}, err
		}
		asset.Domains = append(asset.Domains, record)
	}
	asset.UpdatedAt = now
	asset, err := domain.NormalizeAsset(asset, now)
	if err != nil {
		return domain.Asset{}, err
	}
	s.assets[assetID] = asset
	return cloneAsset(asset), s.saveLocked()
}

func (s *Store) Summary() domain.AssetSummary {
	return domain.Summary(s.List())
}

func (s *Store) load() error {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		return nil
	}

	var assets []domain.Asset
	if err := json.Unmarshal(content, &assets); err != nil {
		return fmt.Errorf("load asset store: %w", err)
	}
	now := s.now().UTC()
	for _, asset := range assets {
		normalized, err := domain.NormalizeAsset(asset, now)
		if err != nil {
			return fmt.Errorf("load asset %q: %w", asset.ID, err)
		}
		s.assets[normalized.ID] = normalized
	}
	return nil
}

func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	assets := make([]domain.Asset, 0, len(s.assets))
	for _, asset := range s.assets {
		assets = append(assets, asset)
	}
	slices.SortFunc(assets, func(a, b domain.Asset) int {
		return strings.Compare(a.ID, b.ID)
	})
	content, err := json.MarshalIndent(assets, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func mergeAsset(existing, incoming domain.Asset, now time.Time) (domain.Asset, error) {
	merged := existing
	merged.Name = firstNonEmpty(incoming.Name, existing.Name)
	merged.PrimaryDomain = incoming.PrimaryDomain
	merged.Tags = mergeStrings(existing.Tags, incoming.Tags)
	merged.Owner = firstNonEmpty(incoming.Owner, existing.Owner)
	merged.BusinessUnit = firstNonEmpty(incoming.BusinessUnit, existing.BusinessUnit)
	merged.Status = firstNonEmpty(incoming.Status, existing.Status)
	merged.Metadata = mergeMap(existing.Metadata, incoming.Metadata)
	merged.UpdatedAt = now

	domains := make(map[string]domain.DomainRecord, len(existing.Domains)+len(incoming.Domains))
	for _, record := range existing.Domains {
		domains[record.Name] = record
	}
	for _, record := range incoming.Domains {
		mergedDomain, err := domain.MergeDomain(domains[record.Name], record, incoming.PrimaryDomain, now)
		if err != nil {
			return domain.Asset{}, err
		}
		domains[mergedDomain.Name] = mergedDomain
	}
	merged.Domains = merged.Domains[:0]
	for _, record := range domains {
		merged.Domains = append(merged.Domains, record)
	}

	ips := make(map[string]domain.IPRecord, len(existing.IPs)+len(incoming.IPs))
	for _, record := range existing.IPs {
		ips[record.Address] = record
	}
	for _, record := range incoming.IPs {
		mergedIP, err := domain.MergeIP(ips[record.Address], record, now)
		if err != nil {
			return domain.Asset{}, err
		}
		ips[mergedIP.Address] = mergedIP
	}
	merged.IPs = merged.IPs[:0]
	for _, record := range ips {
		merged.IPs = append(merged.IPs, record)
	}

	components := make(map[string]domain.ComponentRecord, len(existing.Components)+len(incoming.Components))
	for _, component := range existing.Components {
		components[component.ID] = component
	}
	for _, component := range incoming.Components {
		mergedComponent, err := domain.MergeComponent(components[component.ID], component, now)
		if err != nil {
			return domain.Asset{}, err
		}
		components[mergedComponent.ID] = mergedComponent
	}
	merged.Components = merged.Components[:0]
	for _, component := range components {
		merged.Components = append(merged.Components, component)
	}
	return domain.NormalizeAsset(merged, now)
}

func cloneAsset(asset domain.Asset) domain.Asset {
	content, _ := json.Marshal(asset)
	var cloned domain.Asset
	_ = json.Unmarshal(content, &cloned)
	return cloned
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func mergeStrings(left, right []string) []string {
	seen := make(map[string]struct{}, len(left)+len(right))
	out := make([]string, 0, len(left)+len(right))
	for _, value := range append(left, right...) {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}

func mergeMap(left, right map[string]string) map[string]string {
	if len(left) == 0 && len(right) == 0 {
		return nil
	}
	out := make(map[string]string, len(left)+len(right))
	for key, value := range left {
		out[key] = value
	}
	for key, value := range right {
		out[key] = value
	}
	return out
}
