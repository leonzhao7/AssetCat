<script setup lang="ts">
import {
  AlertTriangle,
  ArrowLeft,
  Box,
  Bug,
  ChevronDown,
  Edit3,
  Globe2,
  Layers3,
  Plus,
  RefreshCw,
  Search,
  Server,
  ShieldAlert,
  Trash2,
  X,
} from 'lucide-vue-next'
import { computed, onMounted, reactive, ref } from 'vue'
import {
  addDomain,
  addDomainComponent,
  addDomainIP,
  addRisk,
  createAsset,
  deleteAsset,
  deleteDomainComponent,
  deleteDomainIP,
  deleteRisk,
  deleteDomain,
  fetchAssets,
  fetchAssetStats,
  updateAsset,
  updateDomainComponent,
  updateDomain,
  updateDomainIP,
  updateRisk,
} from './api'
import type { Asset, AssetStats, ComponentRecord, CreateAssetPayload, DomainRecord, IPRecord, RiskFinding, Severity } from './types'

const severities: Severity[] = ['critical', 'high', 'medium', 'low', 'info']

const assets = ref<Asset[]>([])
const selectedID = ref('')
const expandedDomain = ref('')
const assetStats = ref<AssetStats | null>(null)
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const query = ref('')
const severityFilter = ref('')
const showAssetForm = ref(false)
const showDomainForm = ref(false)
const showRiskForm = ref(false)
const showIPForm = ref(false)
const showComponentForm = ref(false)

const assetForm = reactive({
  mode: 'create' as 'create' | 'edit',
  name: '',
  primaryDomain: '',
  ip: '',
  port: 443,
  service: 'https',
  banner: '',
  componentName: '',
  componentVersion: '',
  proofURL: '',
  responseContent: '',
})

const domainForm = reactive({
  mode: 'create' as 'create' | 'edit',
  originalName: '',
  name: '',
  kind: 'subdomain' as DomainRecord['kind'],
})

const riskForm = reactive({
  mode: 'create' as 'create' | 'edit',
  id: '',
  domain: '',
  title: '',
  severity: 'high' as Severity,
  url: '',
  request: '',
  response: '',
})

const ipForm = reactive({
  mode: 'create' as 'create' | 'edit',
  domain: '',
  originalAddress: '',
  address: '',
  port: 443,
  protocol: 'tcp' as 'tcp' | 'udp',
  service: '',
  banner: '',
})

const componentForm = reactive({
  mode: 'create' as 'create' | 'edit',
  domain: '',
  id: '',
  name: '',
  version: '',
  proofURL: '',
  responseContent: '',
})

const selectedAsset = computed(() => assets.value.find((asset) => asset.id === selectedID.value))
const visibleDomains = computed(() => selectedAsset.value?.domains ?? [])
const visibleRisks = computed(() => visibleDomains.value.flatMap((domain) => (domain.risks ?? []).map((risk) => ({ ...risk, domain: domain.name }))))
const riskCount = computed(() => visibleRisks.value.length)
const viewTitle = computed(() => selectedAsset.value?.primary_domain ?? '资产列表')

const riskScore = computed(() => {
  const counts = assetStats.value?.by_severity ?? {}
  return (counts.critical ?? 0) * 100 + (counts.high ?? 0) * 60 + (counts.medium ?? 0) * 25 + (counts.low ?? 0) * 8
})

async function loadAssets() {
  loading.value = true
  error.value = ''
  try {
    assets.value = await fetchAssets({ q: query.value.trim(), severity: severityFilter.value })
    if (selectedID.value && !assets.value.some((asset) => asset.id === selectedID.value)) {
      selectedID.value = ''
      assetStats.value = null
    }
    if (selectedID.value) {
      await loadSelectedStats()
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    loading.value = false
  }
}

async function loadSelectedStats() {
  if (!selectedID.value) {
    assetStats.value = null
    return
  }
  assetStats.value = await fetchAssetStats(selectedID.value)
}

function openAsset(asset: Asset) {
  selectedID.value = asset.id
  expandedDomain.value = asset.domains[0]?.name ?? ''
  riskForm.domain = asset.primary_domain
  void loadSelectedStats()
}

function backToList() {
  selectedID.value = ''
  expandedDomain.value = ''
  assetStats.value = null
}

function openCreateAsset() {
  Object.assign(assetForm, {
    mode: 'create',
    name: '',
    primaryDomain: '',
    ip: '',
    port: 443,
    service: 'https',
    banner: '',
    componentName: '',
    componentVersion: '',
    proofURL: '',
    responseContent: '',
  })
  showAssetForm.value = true
}

function openEditAsset() {
  const asset = selectedAsset.value
  if (!asset) return
  Object.assign(assetForm, {
    mode: 'edit',
    name: asset.name ?? '',
    primaryDomain: asset.primary_domain,
    ip: '',
    port: 443,
    service: 'https',
    banner: '',
    componentName: '',
    componentVersion: '',
    proofURL: '',
    responseContent: '',
  })
  showAssetForm.value = true
}

async function submitAsset() {
  saving.value = true
  error.value = ''
  try {
    const selected = selectedAsset.value
    const payload: CreateAssetPayload = {
      name: valueOrUndefined(assetForm.name),
      primary_domain: valueOrUndefined(assetForm.primaryDomain),
      ips:
        assetForm.mode === 'edit'
          ? selected?.ips
          : assetForm.ip
            ? [
                {
                  address: assetForm.ip,
                  ports: [
                    {
                      port: Number(assetForm.port),
                      protocol: 'tcp',
                      service: valueOrUndefined(assetForm.service),
                      banner: valueOrUndefined(assetForm.banner),
                    },
                  ],
                },
              ]
            : undefined,
      domains: assetForm.mode === 'edit' ? selected?.domains : undefined,
      components:
        assetForm.mode === 'edit'
          ? selected?.components
          : assetForm.componentName
            ? [
                {
                  name: assetForm.componentName,
                  version: valueOrUndefined(assetForm.componentVersion),
                  proof_url: assetForm.proofURL,
                  response_content: assetForm.responseContent,
                },
              ]
            : undefined,
    }
    const saved =
      assetForm.mode === 'edit' && selected
        ? await updateAsset(selected.id, payload)
        : await createAsset(payload)
    showAssetForm.value = false
    selectedID.value = saved.id
    await loadAssets()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存资产失败'
  } finally {
    saving.value = false
  }
}

async function removeSelectedAsset() {
  const asset = selectedAsset.value
  if (!asset || !window.confirm(`删除资产 ${asset.primary_domain}？`)) return
  saving.value = true
  error.value = ''
  try {
    await deleteAsset(asset.id)
    selectedID.value = ''
    assetStats.value = null
    await loadAssets()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '删除资产失败'
  } finally {
    saving.value = false
  }
}

function openCreateDomain() {
  Object.assign(domainForm, {
    mode: 'create',
    originalName: '',
    name: '',
    kind: 'subdomain',
  })
  showDomainForm.value = true
}

function openEditDomain(record: DomainRecord) {
  Object.assign(domainForm, {
    mode: 'edit',
    originalName: record.name,
    name: record.name,
    kind: record.kind,
  })
  showDomainForm.value = true
}

async function submitDomain() {
  const asset = selectedAsset.value
  if (!asset) return
  saving.value = true
  error.value = ''
  try {
    const payload = {
      name: domainForm.name,
      kind: domainForm.kind,
    }
    const updated =
      domainForm.mode === 'edit'
        ? await updateDomain(asset.id, domainForm.originalName, payload)
        : await addDomain(asset.id, payload)
    showDomainForm.value = false
    selectedID.value = updated.id
    await loadAssets()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存域名失败'
  } finally {
    saving.value = false
  }
}

async function removeDomain(domainName: string) {
  const asset = selectedAsset.value
  if (!asset || !window.confirm(`删除域名 ${domainName}？`)) return
  saving.value = true
  error.value = ''
  try {
    const updated = await deleteDomain(asset.id, domainName)
    selectedID.value = updated.id
    await loadAssets()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '删除域名失败'
  } finally {
    saving.value = false
  }
}

function openRiskForm(domainName: string) {
  Object.assign(riskForm, {
    mode: 'create',
    id: '',
    domain: domainName,
    title: '',
    severity: 'high',
    url: '',
    request: '',
    response: '',
  })
  showRiskForm.value = true
}

function openEditRisk(domainName: string, risk: RiskFinding) {
  Object.assign(riskForm, {
    mode: 'edit',
    id: risk.id ?? '',
    domain: domainName,
    title: risk.title,
    severity: risk.severity,
    url: risk.url,
    request: risk.request,
    response: risk.response,
  })
  showRiskForm.value = true
}

async function submitRisk() {
  const asset = selectedAsset.value
  if (!asset) return
  saving.value = true
  error.value = ''
  try {
    const payload: RiskFinding = {
      title: riskForm.title,
      severity: riskForm.severity,
      url: riskForm.url,
      request: riskForm.request,
      response: riskForm.response,
    }
    const updated =
      riskForm.mode === 'edit'
        ? await updateRisk(asset.id, riskForm.domain, riskForm.id, payload)
        : await addRisk(asset.id, riskForm.domain, payload)
    showRiskForm.value = false
    selectedID.value = updated.id
    await loadAssets()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存风险失败'
  } finally {
    saving.value = false
  }
}

async function removeRisk(domainName: string, riskID?: string) {
  const asset = selectedAsset.value
  if (!asset || !riskID || !window.confirm('删除风险？')) return
  saving.value = true
  error.value = ''
  try {
    const updated = await deleteRisk(asset.id, domainName, riskID)
    selectedID.value = updated.id
    await loadAssets()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '删除风险失败'
  } finally {
    saving.value = false
  }
}

function openIPForm(domainName: string, ip?: IPRecord) {
  Object.assign(ipForm, {
    mode: ip ? 'edit' : 'create',
    domain: domainName,
    originalAddress: ip?.address ?? '',
    address: ip?.address ?? '',
    port: ip?.ports?.[0]?.port ?? 443,
    protocol: ip?.ports?.[0]?.protocol ?? 'tcp',
    service: ip?.ports?.[0]?.service ?? '',
    banner: ip?.ports?.[0]?.banner ?? '',
  })
  showIPForm.value = true
}

async function submitIP() {
  const asset = selectedAsset.value
  if (!asset) return
  const payload: IPRecord = {
    address: ipForm.address,
    ports: [
      {
        port: Number(ipForm.port),
        protocol: ipForm.protocol,
        service: valueOrUndefined(ipForm.service),
        banner: valueOrUndefined(ipForm.banner),
      },
    ],
  }
  saving.value = true
  error.value = ''
  try {
    const updated =
      ipForm.mode === 'edit'
        ? await updateDomainIP(asset.id, ipForm.domain, ipForm.originalAddress, payload)
        : await addDomainIP(asset.id, ipForm.domain, payload)
    showIPForm.value = false
    selectedID.value = updated.id
    await loadAssets()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存 IP 失败'
  } finally {
    saving.value = false
  }
}

async function removeIP(domainName: string, address: string) {
  const asset = selectedAsset.value
  if (!asset || !window.confirm(`删除 IP ${address}？`)) return
  saving.value = true
  error.value = ''
  try {
    const updated = await deleteDomainIP(asset.id, domainName, address)
    selectedID.value = updated.id
    await loadAssets()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '删除 IP 失败'
  } finally {
    saving.value = false
  }
}

function openComponentForm(domainName: string, component?: ComponentRecord) {
  Object.assign(componentForm, {
    mode: component ? 'edit' : 'create',
    domain: domainName,
    id: component?.id ?? '',
    name: component?.name ?? '',
    version: component?.version ?? '',
    proofURL: component?.proof_url ?? '',
    responseContent: component?.response_content ?? '',
  })
  showComponentForm.value = true
}

async function submitComponent() {
  const asset = selectedAsset.value
  if (!asset) return
  const payload: ComponentRecord = {
    id: componentForm.id || undefined,
    name: componentForm.name,
    version: valueOrUndefined(componentForm.version),
    proof_url: componentForm.proofURL,
    response_content: componentForm.responseContent,
  }
  saving.value = true
  error.value = ''
  try {
    const updated =
      componentForm.mode === 'edit'
        ? await updateDomainComponent(asset.id, componentForm.domain, componentForm.id, payload)
        : await addDomainComponent(asset.id, componentForm.domain, payload)
    showComponentForm.value = false
    selectedID.value = updated.id
    await loadAssets()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存组件失败'
  } finally {
    saving.value = false
  }
}

async function removeComponent(domainName: string, componentID?: string) {
  const asset = selectedAsset.value
  if (!asset || !componentID || !window.confirm('删除组件？')) return
  saving.value = true
  error.value = ''
  try {
    const updated = await deleteDomainComponent(asset.id, domainName, componentID)
    selectedID.value = updated.id
    await loadAssets()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '删除组件失败'
  } finally {
    saving.value = false
  }
}

function valueOrUndefined(value: string) {
  const trimmed = value.trim()
  return trimmed ? trimmed : undefined
}

function formatTime(value?: string) {
  if (!value) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

function countRisks(asset: Asset) {
  return asset.domains.reduce((count, domain) => count + (domain.risks?.length ?? 0), 0)
}

function countIPs(asset: Asset) {
  return asset.domains.reduce((count, domain) => count + (domain.ips?.length ?? 0), 0)
}

function countComponents(asset: Asset) {
  return asset.domains.reduce((count, domain) => count + (domain.components?.length ?? 0), 0)
}

function toggleDomain(domainName: string) {
  expandedDomain.value = expandedDomain.value === domainName ? '' : domainName
}

function latestIP(domain: DomainRecord) {
  const latest = [...(domain.ips ?? [])].sort((a, b) => Date.parse(b.last_seen ?? '') - Date.parse(a.last_seen ?? ''))[0]
  return latest?.address ?? '-'
}

function domainRiskCount(domain: DomainRecord) {
  return domain.risks?.length ?? 0
}

function domainServices(domain: DomainRecord) {
  const services = new Set<string>()
  for (const ip of domain.ips ?? []) {
    for (const port of ip.ports ?? []) {
      services.add(`${port.port}/${port.protocol} ${port.service || 'unknown'}`)
    }
  }
  return [...services]
}

onMounted(loadAssets)
</script>

<template>
  <main class="shell">
    <section class="workspace">
      <header class="topbar">
        <div class="title-group">
          <button v-if="selectedAsset" class="icon-button" type="button" title="返回资产列表" @click="backToList">
            <ArrowLeft :size="18" />
          </button>
          <div>
            <p class="eyebrow">AssetCat</p>
            <h1>{{ viewTitle }}</h1>
          </div>
        </div>
        <div class="actions">
          <button class="icon-button" type="button" title="刷新" @click="loadAssets">
            <RefreshCw :size="18" :class="{ spin: loading }" />
          </button>
          <template v-if="selectedAsset">
            <button class="icon-button" type="button" title="编辑资产" @click="openEditAsset">
              <Edit3 :size="18" />
            </button>
            <button class="icon-button danger" type="button" title="删除资产" :disabled="saving" @click="removeSelectedAsset">
              <Trash2 :size="18" />
            </button>
          </template>
          <button v-else class="primary-button" type="button" @click="openCreateAsset">
            <Plus :size="18" />
            <span>新建资产</span>
          </button>
        </div>
      </header>

      <p v-if="error" class="alert">
        <AlertTriangle :size="18" />
        <span>{{ error }}</span>
      </p>

      <section v-if="!selectedAsset" class="asset-home">
        <div class="filters">
          <label class="search-box">
            <Search :size="18" />
            <input v-model="query" type="search" placeholder="搜索资产、域名、ID" @keyup.enter="loadAssets" />
          </label>
          <select v-model="severityFilter" @change="loadAssets">
            <option value="">全部风险</option>
            <option v-for="severity in severities" :key="severity" :value="severity">{{ severity }}</option>
          </select>
        </div>

        <div class="asset-grid">
          <button v-for="asset in assets" :key="asset.id" class="asset-card" type="button" @click="openAsset(asset)">
            <span class="asset-card-head">
              <ShieldAlert :size="20" />
              <strong>{{ asset.primary_domain }}</strong>
            </span>
            <small>{{ asset.name || asset.owner || asset.id }}</small>
            <span class="asset-card-metrics">
              <b>{{ asset.domains.length }}</b> 域名
              <b>{{ countIPs(asset) }}</b> IP
              <b>{{ countComponents(asset) }}</b> 组件
              <b>{{ countRisks(asset) }}</b> 风险
            </span>
            <time>{{ formatTime(asset.updated_at) }}</time>
          </button>
        </div>

        <div v-if="!assets.length && !loading" class="empty">暂无资产</div>
      </section>

      <section v-else class="asset-detail">
        <section class="stats-grid">
          <div class="stat">
            <Globe2 :size="22" />
            <span>域名</span>
            <strong>{{ assetStats?.domains ?? selectedAsset.domains.length }}</strong>
          </div>
          <div class="stat">
            <Server :size="22" />
            <span>端口</span>
            <strong>{{ assetStats?.ports ?? 0 }}</strong>
          </div>
          <div class="stat">
            <Bug :size="22" />
            <span>风险</span>
            <strong>{{ assetStats?.risks ?? riskCount }}</strong>
          </div>
          <div class="stat">
            <Box :size="22" />
            <span>风险指数</span>
            <strong>{{ riskScore }}</strong>
          </div>
        </section>

        <div class="detail-layout">
          <section class="domain-board">
            <div class="panel-title">
              <h2>域名</h2>
              <button class="primary-button compact" type="button" @click="openCreateDomain">
                <Plus :size="17" />
                <span>新增域名</span>
              </button>
            </div>
            <div class="domain-table-head">
              <span>域名</span>
              <span>最新 IP</span>
              <span>风险数量</span>
              <span>更多信息</span>
            </div>
            <template v-for="domain in visibleDomains" :key="domain.name">
              <div class="domain-row" :class="{ expanded: expandedDomain === domain.name }">
                <button class="domain-name-button" type="button" @click="toggleDomain(domain.name)">
                  <strong>{{ domain.name }}</strong>
                  <small>{{ domain.kind }}</small>
                </button>
                <span>{{ latestIP(domain) }}</span>
                <span>{{ domainRiskCount(domain) }}</span>
                <span class="row-actions">
                  <button class="icon-button small" type="button" title="展开详情" @click="toggleDomain(domain.name)">
                    <ChevronDown :size="15" :class="{ rotated: expandedDomain === domain.name }" />
                  </button>
                  <button class="icon-button small" type="button" title="新增 IP" @click="openIPForm(domain.name)">
                    <Server :size="15" />
                  </button>
                  <button class="icon-button small" type="button" title="新增组件" @click="openComponentForm(domain.name)">
                    <Layers3 :size="15" />
                  </button>
                  <button class="icon-button small" type="button" title="登记风险" @click="openRiskForm(domain.name)">
                    <Bug :size="15" />
                  </button>
                  <button class="icon-button small" type="button" title="编辑域名" @click="openEditDomain(domain)">
                    <Edit3 :size="15" />
                  </button>
                  <button class="icon-button small danger" type="button" title="删除域名" :disabled="saving" @click="removeDomain(domain.name)">
                    <Trash2 :size="15" />
                  </button>
                </span>
              </div>

              <div v-if="expandedDomain === domain.name" class="domain-expand">
                <div class="mini-panel">
                  <h3>历史 IP</h3>
                  <div class="mini-row" v-for="ip in domain.ips" :key="ip.address">
                    <span>
                      <strong>{{ ip.address }}</strong>
                      <small>{{ formatTime(ip.last_seen) }}</small>
                    </span>
                    <span class="row-actions">
                      <button class="icon-button small" type="button" title="编辑 IP" @click="openIPForm(domain.name, ip)">
                        <Edit3 :size="14" />
                      </button>
                      <button class="icon-button small danger" type="button" title="删除 IP" @click="removeIP(domain.name, ip.address)">
                        <Trash2 :size="14" />
                      </button>
                    </span>
                  </div>
                  <div v-if="!domain.ips?.length" class="empty mini">暂无 IP</div>
                </div>

                <div class="mini-panel">
                  <h3>服务</h3>
                  <div class="mini-row single" v-for="service in domainServices(domain)" :key="service">
                    <span>
                      <strong>{{ service }}</strong>
                      <small>{{ domain.name }}</small>
                    </span>
                  </div>
                  <div v-if="!domainServices(domain).length" class="empty mini">暂无服务</div>
                </div>

                <div class="mini-panel">
                  <h3>组件</h3>
                  <div class="mini-row" v-for="component in domain.components" :key="component.id">
                    <span>
                      <strong>{{ component.name }} {{ component.version }}</strong>
                      <small>{{ component.proof_url }}</small>
                    </span>
                    <span class="row-actions">
                      <button class="icon-button small" type="button" title="编辑组件" @click="openComponentForm(domain.name, component)">
                        <Edit3 :size="14" />
                      </button>
                      <button class="icon-button small danger" type="button" title="删除组件" @click="removeComponent(domain.name, component.id)">
                        <Trash2 :size="14" />
                      </button>
                    </span>
                  </div>
                  <div v-if="!domain.components?.length" class="empty mini">暂无组件</div>
                </div>

                <div class="mini-panel">
                  <h3>风险</h3>
                  <div class="mini-row" v-for="risk in domain.risks" :key="risk.id">
                    <span>
                      <strong>{{ risk.title }}</strong>
                      <small>{{ risk.severity }} · {{ risk.url }}</small>
                    </span>
                    <span class="row-actions">
                      <button class="icon-button small" type="button" title="编辑风险" @click="openEditRisk(domain.name, risk)">
                        <Edit3 :size="14" />
                      </button>
                      <button class="icon-button small danger" type="button" title="删除风险" @click="removeRisk(domain.name, risk.id)">
                        <Trash2 :size="14" />
                      </button>
                    </span>
                  </div>
                  <div v-if="!domain.risks?.length" class="empty mini">暂无风险</div>
                </div>
              </div>
            </template>
          </section>

          <aside class="asset-side">
            <section class="panel">
              <h3>最新风险</h3>
              <div class="risk-item" v-for="risk in visibleRisks.slice(0, 6)" :key="risk.id || `${risk.domain}-${risk.title}`">
                <span :class="['severity', risk.severity]">{{ risk.severity }}</span>
                <div>
                  <strong>{{ risk.title }}</strong>
                  <small>{{ risk.domain }} · {{ risk.url }}</small>
                </div>
              </div>
              <div v-if="!visibleRisks.length" class="empty small">暂无风险</div>
            </section>
          </aside>
        </div>
      </section>
    </section>

    <div v-if="showAssetForm" class="overlay" @click.self="showAssetForm = false">
      <form class="drawer" @submit.prevent="submitAsset">
        <header>
          <h2>{{ assetForm.mode === 'edit' ? '编辑资产' : '新建资产' }}</h2>
          <button class="icon-button" type="button" title="关闭" @click="showAssetForm = false">
            <X :size="18" />
          </button>
        </header>
        <label>资产名称<input v-model="assetForm.name" placeholder="生产官网" /></label>
        <label>资产域名<input v-model="assetForm.primaryDomain" placeholder="example.com，纯 IP 资产可留空" /></label>
        <template v-if="assetForm.mode === 'create'">
          <div class="form-grid">
            <label>IP<input v-model="assetForm.ip" placeholder="203.0.113.10" /></label>
            <label>端口<input v-model.number="assetForm.port" type="number" min="1" max="65535" /></label>
          </div>
          <div class="form-grid">
            <label>服务<input v-model="assetForm.service" placeholder="https" /></label>
            <label>Banner<input v-model="assetForm.banner" placeholder="nginx/1.24" /></label>
          </div>
          <label>组件<input v-model="assetForm.componentName" placeholder="nginx" /></label>
          <label>组件版本<input v-model="assetForm.componentVersion" placeholder="1.24" /></label>
          <label>证明 URL<input v-model="assetForm.proofURL" placeholder="https://example.com/" /></label>
          <label>响应内容<textarea v-model="assetForm.responseContent" rows="5" placeholder="HTTP/1.1 200 OK..."></textarea></label>
        </template>
        <button class="primary-button full" type="submit" :disabled="saving">
          <Plus :size="18" />
          <span>{{ saving ? '保存中' : '保存资产' }}</span>
        </button>
      </form>
    </div>

    <div v-if="showDomainForm" class="overlay" @click.self="showDomainForm = false">
      <form class="drawer" @submit.prevent="submitDomain">
        <header>
          <h2>{{ domainForm.mode === 'edit' ? '编辑域名' : '新增域名' }}</h2>
          <button class="icon-button" type="button" title="关闭" @click="showDomainForm = false">
            <X :size="18" />
          </button>
        </header>
        <label>域名<input v-model="domainForm.name" required placeholder="api.example.com" /></label>
        <label>类型
          <select v-model="domainForm.kind">
            <option value="primary">primary</option>
            <option value="subdomain">subdomain</option>
            <option value="ip_alias">ip_alias</option>
          </select>
        </label>
        <button class="primary-button full" type="submit" :disabled="saving">
          <Plus :size="18" />
          <span>{{ saving ? '保存中' : '保存域名' }}</span>
        </button>
      </form>
    </div>

    <div v-if="showRiskForm" class="overlay" @click.self="showRiskForm = false">
      <form class="drawer" @submit.prevent="submitRisk">
        <header>
          <h2>{{ riskForm.mode === 'edit' ? '编辑风险' : '登记风险' }}</h2>
          <button class="icon-button" type="button" title="关闭" @click="showRiskForm = false">
            <X :size="18" />
          </button>
        </header>
        <label>域名<input v-model="riskForm.domain" required /></label>
        <label>标题<input v-model="riskForm.title" required placeholder="admin console exposed" /></label>
        <label>严重级别
          <select v-model="riskForm.severity">
            <option v-for="severity in severities" :key="severity" :value="severity">{{ severity }}</option>
          </select>
        </label>
        <label>URL<input v-model="riskForm.url" required placeholder="https://api.example.com/admin" /></label>
        <label>请求<textarea v-model="riskForm.request" required rows="5"></textarea></label>
        <label>响应<textarea v-model="riskForm.response" required rows="6"></textarea></label>
        <button class="primary-button full" type="submit" :disabled="saving">
          <Plus :size="18" />
          <span>{{ saving ? '保存中' : '保存风险' }}</span>
        </button>
      </form>
    </div>

    <div v-if="showIPForm" class="overlay" @click.self="showIPForm = false">
      <form class="drawer" @submit.prevent="submitIP">
        <header>
          <h2>{{ ipForm.mode === 'edit' ? '编辑 IP' : '新增 IP' }}</h2>
          <button class="icon-button" type="button" title="关闭" @click="showIPForm = false">
            <X :size="18" />
          </button>
        </header>
        <label>域名<input v-model="ipForm.domain" required /></label>
        <label>IP<input v-model="ipForm.address" required placeholder="203.0.113.10" /></label>
        <div class="form-grid">
          <label>端口<input v-model.number="ipForm.port" type="number" min="1" max="65535" /></label>
          <label>协议
            <select v-model="ipForm.protocol">
              <option value="tcp">tcp</option>
              <option value="udp">udp</option>
            </select>
          </label>
        </div>
        <label>服务<input v-model="ipForm.service" placeholder="https" /></label>
        <label>Banner<input v-model="ipForm.banner" placeholder="nginx/1.24" /></label>
        <button class="primary-button full" type="submit" :disabled="saving">
          <Plus :size="18" />
          <span>{{ saving ? '保存中' : '保存 IP' }}</span>
        </button>
      </form>
    </div>

    <div v-if="showComponentForm" class="overlay" @click.self="showComponentForm = false">
      <form class="drawer" @submit.prevent="submitComponent">
        <header>
          <h2>{{ componentForm.mode === 'edit' ? '编辑组件' : '新增组件' }}</h2>
          <button class="icon-button" type="button" title="关闭" @click="showComponentForm = false">
            <X :size="18" />
          </button>
        </header>
        <label>域名<input v-model="componentForm.domain" required /></label>
        <label>组件<input v-model="componentForm.name" required placeholder="nginx" /></label>
        <label>版本<input v-model="componentForm.version" placeholder="1.24" /></label>
        <label>证明 URL<input v-model="componentForm.proofURL" required placeholder="https://example.com/" /></label>
        <label>响应内容<textarea v-model="componentForm.responseContent" required rows="6"></textarea></label>
        <button class="primary-button full" type="submit" :disabled="saving">
          <Plus :size="18" />
          <span>{{ saving ? '保存中' : '保存组件' }}</span>
        </button>
      </form>
    </div>
  </main>
</template>
