export type Severity = 'info' | 'low' | 'medium' | 'high' | 'critical'

export interface AssetStats {
  asset_id: string
  primary_domain: string
  domains: number
  subdomains: number
  ips: number
  ports: number
  components: number
  risks: number
  by_severity: Partial<Record<Severity, number>>
  last_updated?: string
}

export interface Asset {
  id: string
  name?: string
  primary_domain: string
  domains: DomainRecord[]
  ips: IPRecord[]
  components: ComponentRecord[]
  tags?: string[]
  owner?: string
  business_unit?: string
  status?: string
  metadata?: Record<string, string>
  created_at: string
  updated_at: string
}

export interface DomainRecord {
  name: string
  kind: 'primary' | 'subdomain' | 'ip_alias'
  risks?: RiskFinding[]
  first_seen: string
  last_seen: string
}

export interface RiskFinding {
  id?: string
  title: string
  severity: Severity
  url: string
  request: string
  response: string
  description?: string
  remediation?: string
  status?: string
  cve?: string
  cwe?: string
  component_id?: string
  confidence?: number
  discovered_by?: string
  first_seen?: string
  last_seen?: string
}

export interface ComponentRecord {
  id?: string
  name: string
  version?: string
  category?: string
  proof_url: string
  response_content: string
  confidence?: number
  source?: string
  metadata?: Record<string, string>
  first_seen?: string
  last_seen?: string
}

export interface IPRecord {
  address: string
  ports?: PortRecord[]
  asn?: string
  isp?: string
  geo?: string
  first_seen?: string
  last_seen?: string
}

export interface PortRecord {
  port: number
  protocol: 'tcp' | 'udp'
  service?: string
  banner?: string
  tls?: boolean
  first_seen?: string
  last_seen?: string
}

export interface CreateAssetPayload {
  name?: string
  primary_domain?: string
  owner?: string
  business_unit?: string
  tags?: string[]
  ips?: IPRecord[]
  domains?: Partial<DomainRecord>[]
  components?: ComponentRecord[]
}
