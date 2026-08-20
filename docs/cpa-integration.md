# CPA（CLIProxyAPI）集成与凭证改造参考

> 本文档整理 new-api 与 CPA（CLI 代理管理服，仓库 `router-for-me/CLIProxyAPI`）之间的调用关系，
> 用于凭证相关的改造。代码位置均为写档时的实际路径，改动后请同步更新。

---

## 一、总体关系

- CPA 负责**持有并刷新** Codex 等账号凭证：自身有 5 秒周期的后台自动刷新循环，转发遇 401 时也会懒刷新；
  刷新后由 `FileTokenStore.Save` 把新 token 写回磁盘上的 auth 文件（`auth_dir/*.json`）。
- new-api 对 CPA 目前**只读**：通过 `/v0/management/auth-files`（列表）和 `/v0/management/auth-files/download`（下载）拿凭证元数据；
  Grok 额度再经 `POST /v0/management/api-call` 代请求 xAI billing 后解析展示。
  实际转发的 Codex 渠道凭证是管理员手动粘贴进渠道配置的（channel.key）。
- **没有显式"立即刷新"端点**。若改造后需要"点一下立刻刷"，要么依赖 CPA 自身的自律刷新，
  要么后续在 CPA 侧新增触发端点。

---

## 二、CPA 管理端点

前缀 `/v0/management`。鉴权：`Authorization: Bearer <management-key>` 或 `X-Management-Key` 头；
远程访问需 CPA 配置 `remote-management.allow-remote=true`；CPA 处于 Home 模式时全部 404。
路由注册：`internal/api/server_management.go`（写档时 :14），中间件：`internal/api/handlers/management/handler.go:265`。

### 2.1 凭证 / auth-files（核心）

| 方法 | 路径 | 说明 | 实现 |
|---|---|---|---|
| GET | `/auth-files` | 列表，支持 `?name=`、`?auth_index=` 过滤；返回 name/email/account/account_type/auth_index/status/status_message/last_refresh/next_retry_after/id_token{chatgpt_account_id,plan_type,chatgpt_subscription_active_until,...}，是 new-api 解析字段的超集 | `auth_files.go:90` `ListAuthFiles` |
| GET | `/auth-files/models?name=` | 查询某凭证可用的模型 | `auth_files.go:169` |
| GET | `/auth-files/download?name=xxx.json` | 下载磁盘上的 auth 原始 JSON（含 access_token/refresh_token/account_id/proxy_url/id_token/expired/last_refresh） | `auth_files_crud.go:26` |
| POST | `/auth-files` | 上传（multipart 或 raw JSON + `?name=`），写入 auth_dir | `auth_files_crud.go:51` |
| DELETE | `/auth-files?name=` 或 body `{name,names}` 或 `?all=true` | 删除 | `auth_files_crud.go:132` |
| PATCH | `/auth-files/status` | body `{name, auth_index?, disabled}`，禁用/启用 | `auth_files_fields.go:24` |
| PATCH | `/auth-files/fields` | body `{name, <field>: value, ...}`，支持嵌套路径如 `headers.X`、`priority`、`proxy_url` 等 | `auth_files_fields.go:203` |

### 2.2 OAuth 新增凭证（可选）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/codex-auth-url` | 发起 Codex OAuth 授权，返回登录 URL |
| GET/POST | `/oauth-callback` | 回调查询/提交 |
| GET | `/get-auth-status?state=` | 轮询授权状态 |
| DELETE | `/oauth-session?state=` | 取消授权 |
| GET | `/anthropic-auth-url`、`/antigravity-auth-url`、`/kimi-auth-url`、`/xai-auth-url` | 其他 provider 授权入口 |
| POST | `/vertex/import` | Vertex 凭证导入 |

实现集中在 `internal/api/handlers/management/auth_files_provider_oauth.go`（RequestCodexToken :196、GetAuthStatus :757、CancelAuthSession :743 等）。

### 2.3 其他管理端点（一般不需要动）

`/config`、`/config.yaml`(GET/PUT)、`/api-keys`(CRUD)、`/codex-api-key`、`/claude-api-key`、`/gemini-api-key`、
`/xai-api-key`、`/openai-compatibility`、`/vertex-api-key`、`/logs`、`/proxy-url`、`/routing/strategy`、
`/quota-exceeded/*`、`/reset-quota`、`/usage-queue`、`/ws-auth`、`/request-log`、`/plugins*`、`/model-definitions/:channel` 等。

---

## 三、new-api 侧代码位置

### 3.1 后端

| 层 | 位置 | 内容 |
|---|---|---|
| 路由 | `router/api-router.go:309-310` | `GET/POST /api/dashboard/cpa-quotas`（仅 `UserAuth`） |
| 路由 | `router/channel-router.go:63` | `POST /api/channel/:id/codex/refresh`（`authz.ChannelSensitiveWrite`） |
| 控制器 | `controller/dashboard.go:31-41` | `GetDashboardCPAQuotas`(45s 超时) / `RefreshDashboardCPAQuotaStatus`(60s 超时)，都直调 `service.GetDashboardCPAQuotaData` |
| 控制器 | `controller/codex_usage.go` | 渠道用量/重置（内部含 401 自刷） |
| 控制器 | `controller/channel.go:541` | `RefreshCodexChannelCredential`（手动刷新入口） |
| 控制器 | `controller/option.go:406-426` | `console_setting.cpa_*` 三项的写入校验 |
| 服务 | `service/dashboard.go` | **CPA 集成核心**：列表/下载 auth-files、Codex WHAM 用量、Grok 走 `applyCPAGrokQuota` |
| 服务 | `service/cpa_grok_quota.go` | Grok 额度：下载 xAI 凭证识别付费档，再经 `api-call` 拉周/月 billing 并解析百分比 |
| 服务 | `service/codex_oauth.go` | `RefreshCodexOAuthTokenWithProxy`(:33)、Codex JWT 解析 |
| 服务 | `service/codex_credential_refresh.go:42` | 写回 channel.key 的刷新 |
| 服务 | `service/codex_credential_refresh_task.go` | 10 分钟后台自动刷新任务（仅 master 节点） |
| 服务 | `service/codex_channel_models.go` / `service/codex_wham_usage.go` | 模型发现（含 401 自刷）/ 打 chatgpt.com wham 用量 |
| 模型 | `model/option.go:179-181` | `console_setting.cpa_base_url / cpa_management_key / cpa_channel_ids` 默认值 |
| 中继 | `relay/channel/codex/adaptor.go:154-198` | 转发时从 channel.key 的 JSON 取 access_token/account_id 拼请求头 |

### 3.2 前端

| 位置 | 内容 |
|---|---|
| `web/src/features/dashboard/api.ts:120-133` | `getDashboardCPAQuotas` / `refreshDashboardCPAQuotaStatus` |
| `web/src/features/dashboard/types.ts:341-385` | `DashboardCPAQuotaAccount/Window/Data/ChannelItem` |
| `web/src/features/dashboard/components/overview/cpa-quota-panel.tsx` | 额度预览面板（含手动刷新按钮） |
| `web/src/features/system-settings/integrations/cpa-settings-section.tsx` | CPA 配置表单（base_url + management_key + channel_ids） |
| `web/src/features/system-settings/operations/section-registry.tsx:44-60`、`index.tsx:33-35`、`types.ts:339-341` | CPA 配置项注册 |
| `web/src/features/channels/api.ts:311-349` | codex 渠道的 refresh/usage/reset 调用 |
| `web/src/features/channels/lib/channel-form.ts:155` | Codex 凭证 JSON 校验（要求 access_token+account_id 非空） |
| `web/src/features/channels/constants.ts:80,403` | Codex 渠道类型（57）、凭证粘贴文案 |

---

## 四、改造要点备忘

- **"由 CPA 刷、new-api 只读"可行**：CPA 自律刷新会写回 auth 文件，new-api 每次调 download 即可拿到最新 token，
  不必自己调 OAuth 再写回 channel.key。
- 若采用该方向，应停用 new-api 自己的四条自刷路径：手动 `/codex/refresh`、10 分钟后台任务、
  渠道用量/模型发现的 401 懒刷新、dashboard 额度获取的 401 重试。
- 双主问题：CPA auth 文件与 new-api channel.key 各持一份 refresh_token 时，任意一侧刷新都会使另一侧旧 refresh_token 失效，
  需在改造中明确"唯一刷新方"。
- 权限注意：`/api/dashboard/cpa-quotas` 目前挂在 `middleware.UserAuth()`，若新增"拉取/写入凭证"能力需收紧为管理员权限。