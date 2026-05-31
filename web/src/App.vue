<script setup lang="ts">
import {
  AlertTriangle,
  BarChart3,
  Box,
  Bug,
  Globe2,
  Layers3,
  Network,
  Plus,
  RefreshCw,
  Search,
  Server,
  ShieldAlert,
  X,
} from 'lucide-vue-next'
import { computed, onMounted, reactive, ref } from 'vue'
import { addRisk, createAsset, fetchAssets, fetchSummary } from './api'
import type { Asset, AssetSummary, CreateAssetPayload, RiskFinding, Severity } from './types'

const severities: Severity[] = ['critical', 'high', 'medium', 'low', 'info']

const summary = ref<AssetSummary | null>(null)
const assets = ref<Asset[]>([])
const selectedID = ref('')
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const query = ref('')
const severityFilter = ref('')
const showAssetForm = ref(false)
const showRiskForm = ref(false)

const assetForm = reactive({
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

const riskForm = reactive({
  domain: '',
  title: '',
  severity: 'high' as Severity,
  url: '',
  request: '',
  response: '',
})

const selectedAsset = computed(() => assets.value.find((asset) => asset.id === selectedID.value) ?? assets.value[0])
const visibleDomains = computed(() => selectedAsset.value?.domains ?? [])
const visibleRisks = computed(() => visibleDomains.value.flatMap((domain) => (domain.risks ?? []).map((risk) => ({ ...risk, domain: domain.name }))))
const riskCount = computed(() => visibleRisks.value.length)

const score = computed(() => {
  const counts = summary.value?.by_severity ?? {}
  return (counts.critical ?? 0) * 100 + (counts.high ?? 0) * 60 + (counts.medium ?? 0) * 25 + (counts.low ?? 0) * 8
})

const topRiskLabel = computed(() => {
  const counts = summary.value?.by_severity ?? {}
  const severity = severities.find((item) => (counts[item] ?? 0) > 0)
  return severity ? severity.toUpperCase() : 'CLEAR'
})

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    const [nextSummary, nextAssets] = await Promise.all([
      fetchSummary(),
      fetchAssets({ q: query.value.trim(), severity: severityFilter.value }),
    ])
    summary.value = nextSummary
    assets.value = nextAssets
    if (!assets.value.some((asset) => asset.id === selectedID.value)) {
      selectedID.value = assets.value[0]?.id ?? ''
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    loading.value = false
  }
}

async function submitAsset() {
  saving.value = true
  error.value = ''
  try {
    const payload: CreateAssetPayload = {
      name: valueOrUndefined(assetForm.name),
      primary_domain: valueOrUndefined(assetForm.primaryDomain),
      ips: assetForm.ip
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
      components: assetForm.componentName
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
    const created = await createAsset(payload)
    resetAssetForm()
    showAssetForm.value = false
    selectedID.value = created.id
    await loadData()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存资产失败'
  } finally {
    saving.value = false
  }
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
    const domain = riskForm.domain || asset.primary_domain
    const updated = await addRisk(asset.id, domain, payload)
    resetRiskForm()
    showRiskForm.value = false
    selectedID.value = updated.id
    await loadData()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存风险失败'
  } finally {
    saving.value = false
  }
}

function selectAsset(asset: Asset) {
  selectedID.value = asset.id
  riskForm.domain = asset.primary_domain
}

function resetAssetForm() {
  Object.assign(assetForm, {
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
}

function resetRiskForm() {
  Object.assign(riskForm, {
    domain: selectedAsset.value?.primary_domain ?? '',
    title: '',
    severity: 'high',
    url: '',
    request: '',
    response: '',
  })
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

onMounted(loadData)
</script>

<template>
  <main class="shell">
    <aside class="sidebar">
      <div class="brand">
        <ShieldAlert :size="28" />
        <div>
          <strong>AssetCat</strong>
          <span>Risk Operations</span>
        </div>
      </div>

      <nav class="nav">
        <button class="nav-item active" type="button">
          <BarChart3 :size="18" />
          <span>总览</span>
        </button>
        <button class="nav-item" type="button">
          <Globe2 :size="18" />
          <span>资产</span>
        </button>
        <button class="nav-item" type="button">
          <Bug :size="18" />
          <span>风险</span>
        </button>
      </nav>

      <section class="risk-meter">
        <span>风险指数</span>
        <strong>{{ score }}</strong>
        <small>{{ topRiskLabel }}</small>
      </section>
    </aside>

    <section class="workspace">
      <header class="topbar">
        <div>
          <p class="eyebrow">网络资产风险管理</p>
          <h1>资产暴露面</h1>
        </div>
        <div class="actions">
          <button class="icon-button" type="button" title="刷新" @click="loadData">
            <RefreshCw :size="18" :class="{ spin: loading }" />
          </button>
          <button class="primary-button" type="button" @click="showAssetForm = true">
            <Plus :size="18" />
            <span>新建资产</span>
          </button>
        </div>
      </header>

      <p v-if="error" class="alert">
        <AlertTriangle :size="18" />
        <span>{{ error }}</span>
      </p>

      <section class="stats-grid">
        <div class="stat">
          <Globe2 :size="22" />
          <span>资产</span>
          <strong>{{ summary?.assets ?? 0 }}</strong>
        </div>
        <div class="stat">
          <Network :size="22" />
          <span>域名</span>
          <strong>{{ summary?.domains ?? 0 }}</strong>
        </div>
        <div class="stat">
          <Server :size="22" />
          <span>端口</span>
          <strong>{{ summary?.ports ?? 0 }}</strong>
        </div>
        <div class="stat">
          <Bug :size="22" />
          <span>风险</span>
          <strong>{{ summary?.risks ?? 0 }}</strong>
        </div>
      </section>

      <section class="content-grid">
        <div class="asset-list">
          <div class="filters">
            <label class="search-box">
              <Search :size="18" />
              <input v-model="query" type="search" placeholder="搜索域名、资产名、ID" @keyup.enter="loadData" />
            </label>
            <select v-model="severityFilter" @change="loadData">
              <option value="">全部风险</option>
              <option v-for="severity in severities" :key="severity" :value="severity">{{ severity }}</option>
            </select>
          </div>

          <div class="table-head">
            <span>资产</span>
            <span>IP / 组件 / 风险</span>
            <span>更新时间</span>
          </div>

          <button
            v-for="asset in assets"
            :key="asset.id"
            class="asset-row"
            :class="{ selected: selectedAsset?.id === asset.id }"
            type="button"
            @click="selectAsset(asset)"
          >
            <span>
              <strong>{{ asset.primary_domain }}</strong>
              <small>{{ asset.name || asset.owner || asset.id }}</small>
            </span>
            <span class="metrics">
              <b>{{ asset.ips.length }}</b> IP
              <b>{{ asset.components.length }}</b> 组件
              <b>{{ asset.domains.reduce((count, domain) => count + (domain.risks?.length ?? 0), 0) }}</b> 风险
            </span>
            <time>{{ formatTime(asset.updated_at) }}</time>
          </button>

          <div v-if="!assets.length && !loading" class="empty">暂无资产</div>
        </div>

        <article class="detail" v-if="selectedAsset">
          <div class="detail-head">
            <div>
              <p class="eyebrow">Asset Detail</p>
              <h2>{{ selectedAsset.primary_domain }}</h2>
              <span>{{ selectedAsset.id }}</span>
            </div>
            <button class="primary-button compact" type="button" @click="showRiskForm = true; resetRiskForm()">
              <Plus :size="17" />
              <span>登记风险</span>
            </button>
          </div>

          <div class="detail-kpis">
            <span><Globe2 :size="16" /> {{ selectedAsset.domains.length }} 域名</span>
            <span><Server :size="16" /> {{ selectedAsset.ips.length }} IP</span>
            <span><Box :size="16" /> {{ selectedAsset.components.length }} 组件</span>
            <span><Bug :size="16" /> {{ riskCount }} 风险</span>
          </div>

          <section class="panel">
            <h3>域名与风险</h3>
            <div class="domain-line" v-for="domain in visibleDomains" :key="domain.name">
              <span>
                <strong>{{ domain.name }}</strong>
                <small>{{ domain.kind }}</small>
              </span>
              <em>{{ domain.risks?.length ?? 0 }}</em>
            </div>
          </section>

          <section class="panel">
            <h3>IP 端口服务</h3>
            <div class="ip-block" v-for="ip in selectedAsset.ips" :key="ip.address">
              <strong>{{ ip.address }}</strong>
              <div class="port-list">
                <span v-for="port in ip.ports" :key="`${ip.address}-${port.port}-${port.protocol}`">
                  {{ port.port }}/{{ port.protocol }} {{ port.service || 'unknown' }}
                </span>
              </div>
            </div>
          </section>

          <section class="panel">
            <h3>组件证明</h3>
            <div class="component" v-for="component in selectedAsset.components" :key="component.id">
              <Layers3 :size="17" />
              <span>
                <strong>{{ component.name }} {{ component.version }}</strong>
                <small>{{ component.proof_url }}</small>
              </span>
            </div>
          </section>

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
        </article>
      </section>
    </section>

    <div v-if="showAssetForm" class="overlay" @click.self="showAssetForm = false">
      <form class="drawer" @submit.prevent="submitAsset">
        <header>
          <h2>新建资产</h2>
          <button class="icon-button" type="button" title="关闭" @click="showAssetForm = false">
            <X :size="18" />
          </button>
        </header>
        <label>资产名称<input v-model="assetForm.name" placeholder="生产官网" /></label>
        <label>主域名<input v-model="assetForm.primaryDomain" placeholder="example.com，纯 IP 资产可留空" /></label>
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
        <button class="primary-button full" type="submit" :disabled="saving">
          <Plus :size="18" />
          <span>{{ saving ? '保存中' : '保存资产' }}</span>
        </button>
      </form>
    </div>

    <div v-if="showRiskForm" class="overlay" @click.self="showRiskForm = false">
      <form class="drawer" @submit.prevent="submitRisk">
        <header>
          <h2>登记风险</h2>
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
  </main>
</template>
