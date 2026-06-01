# 网络资产风险管理系统设计

## 目标

系统用于沉淀网络资产、暴露面组件、端口服务和风险证据。当前实现优先保证数据结构清晰、接口可用、证据可追溯，后续可以平滑替换为关系型数据库或搜索引擎。

## 核心对象

### 资产 Asset

资产以 `primary_domain` 标识：

- 有域名的资产使用主域名，例如 `example.com`。
- 没有域名的资产使用 IP 作为域名别名，例如 `203.0.113.10`。
- `id` 默认由主标识归一化后生成稳定哈希，也允许外部系统指定。

资产包含：

- `domains`: 主域名、子域名或 IP 别名。
- `ips`: IP 及端口服务。
- `components`: 指纹识别出的组件。
- `tags`, `owner`, `business_unit`, `status`, `metadata`: 运营补充字段。

### 域名 DomainRecord

每个域名或子域名保存自己的风险证据：

- `name`: 域名、子域名或 IP。
- `kind`: `primary`, `subdomain`, `ip_alias`。
- `risks`: 风险列表。

### 风险 RiskFinding

风险不是只存一个标题，而是必须绑定完整 HTTP 证据：

- `title`, `severity`, `status`: 风险定性。
- `url`: 触发风险的 URL。
- `request`: 原始请求。
- `response`: 原始响应。
- `component_id`, `cve`, `cwe`, `confidence`: 关联组件和漏洞知识库。

这样可以满足“每个域名/子域名都包含若干个风险、url、请求、响应的对应”。

### 组件 ComponentRecord

组件必须带证明：

- `name`, `version`, `category`: 组件身份。
- `proof_url`: 证明组件存在的 URL。
- `response_content`: 证明响应内容。
- `confidence`, `source`, `metadata`: 指纹置信度和来源。

### IP 与端口

`IPRecord` 包含多个 `PortRecord`：

- `address`: IP 地址。
- `ports`: 端口列表。
- `port`, `protocol`, `service`, `banner`, `tls`: 端口服务和 banner。

## 关键约束

- 所有域名统一小写并去掉末尾点。
- `primary_domain` 为空时，使用第一个 IP 作为主标识。
- 组件必须包含 `proof_url` 和 `response_content`。
- 风险必须包含 `url`, `request`, `response`。
- 端口范围必须是 `1-65535`，协议只支持 `tcp` 或 `udp`。
- 重复写入资产时按域名、IP、端口和组件 ID 合并。

## API

基础接口：

- `GET /healthz`: 健康检查。
- `GET /assets`: 资产列表，支持 `q`, `ip`, `component`, `severity` 过滤。
- `POST /assets`: 创建或合并资产。
- `GET /assets/{id}`: 获取资产详情。
- `GET /assets/{id}/stats`: 获取当前资产的域名、子域名、IP、端口、组件、风险统计。
- `PUT /assets/{id}`: 替换资产。
- `DELETE /assets/{id}`: 删除资产。

增量接口：

- `POST /assets/{id}/domains`: 追加或合并域名/子域名。
- `PUT /assets/{id}/domains/{domain}`: 更新资产内域名或子域名。
- `DELETE /assets/{id}/domains/{domain}`: 删除资产内域名或子域名，主域名不能通过该接口删除。
- `POST /assets/{id}/ips`: 追加或合并 IP 和端口。
- `POST /assets/{id}/components`: 追加或更新组件。
- `POST /assets/{id}/domains/{domain}/risks`: 给指定域名追加或更新风险。
- `GET /assets/{id}/risks`: 拉平查看资产下全部风险，支持 `severity` 过滤。

## 后续扩展建议

- 存储层替换为 PostgreSQL：资产、域名、IP、端口、组件、风险分别建表。
- 增加扫描任务：资产发现、端口扫描、HTTP 探测、组件指纹、漏洞验证拆成独立任务。
- 增加风险生命周期：`open`, `accepted`, `fixed`, `false_positive`。
- 增加证据归档：大响应内容可落对象存储，数据库只保存摘要和引用。
- 增加资产归属和 SLA：按业务线、负责人、严重级别生成修复看板。
