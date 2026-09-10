# ADN 结算平台：项目上下文与协作约定

供新会话快速接手。先阅读本文件，再按任务定位相关代码；完整使用说明见 `README.md`。本文件记录已确认的事实，不代替当前代码或必要的验证。

## 接手方式

- 使用中文沟通。用户希望减少重复探索和 token 消耗。
- 不要为了了解项目重新启动服务、连接数据库、全量扫描目录或重跑全部测试。优先使用下面的代码索引，只检查与本次修改相关的文件。
- 已有验证结果属于历史快照。涉及相应行为的代码发生变化时，执行必要的针对性验证，不要将历史通过当成本次通过。
- 2026-09-10 独立 18081 测试实例已清理。172.16.3.34 的 ADN（0.0.0.0:18080）与 OpenResty（80）现已运行；已按用户要求开放 IP:18080 访问，重启 ADN 并确认跨机器页面 200、未登录 API 401，未操作 nginx。后续按需操作并记录状态，不复用历史 PID。
- 每次完成架构、数据库、业务规则或运行状态变更后，更新本文件对应内容及日期；保持精简，不追加聊天流水账。不保存密码、令牌或完整含密码的连接串。

## 项目概况

- 当前目录就是完整项目，名称 `adn-report-ai`；不要再创建一层项目目录。
- 2026-09-10 已迁移为 Go 1.25 + go-zero REST：`rest.RestConf`、`rest.Server`、`rest.Route`、框架中间件与优雅退出；`database/sql` + pgx v5，Excelize v2。已按 `handler → logic → model` 分层，`svc.ServiceContext` 注入模型接口与客户端，`middleware` 负责认证/来源/导入并发限制，`types` 定义请求响应；handler/logic 按 auth/account/report 模块组织。`server` 仅装配，不保存 SQL 或业务规则。
- 2026-09-10 前端已迁移为 Vue 3 Composition API + 单文件组件 + Vite，源码 `web/src/`，构建产物 `web/dist` 通过 `go:embed` 随 Go 服务发布。`make build/run` 先构建 Vue；首次先 `make web-install`。Node >=22.12；本机默认 Node 16，可用 `export PATH=/opt/homebrew/bin:$PATH` 切到已有 Node 23。
- 开发热更新用 `make backend CONFIG=etc/config.dev.yaml` + `make web-dev`；开发私有配置的 BaseURL 设为 `http://127.0.0.1:5173`，Vite 同源代理 `/api` 和 `/auth` 到 18080。详细步骤见 README。
- 支持 nginx 子路径部署：BaseURL 可为 `http://172.16.3.34/adn-report`，Origin 校验只取协议/主机，Cookie 和登录跳转带路径前缀；前端相对资源/API 地址兼容根路径和子路径。nginx 转发时去掉前缀。
- go-zero 请求上限固定 21 MiB（文件仍限 20 MB）、超时 180 秒；关闭可能泄露请求凭据的默认请求日志和 6060 调试服务。保留 Origin、Cookie 与现有安全响应头。
- 第一版已具备：钉钉授权、本地管理员登录、ADN 账户维护、Excel 校验预览、CSV 归档、结算入库、上传记录筛选、动态字段明细、CSV 下载、个人与管理员权限。
- 架构最初参考 `../MCP`；钉钉流程参考 `/Users/chemanyu/workspace/python/lumenx/src/apps/comic_gen/{auth.py,api.py}`。流程已经落地，无相关需求不必重新读取参考项目，不复制其密钥。

## 数据库与配置

- **当前是 PostgreSQL，已完成 MySQL 切换。** 地址 `172.16.3.25:5432`，数据库 `inhouse_ads`，业务 schema `adn_report`。不要改回旧 MySQL，也不要把业务表放进 `public`。
- 私有配置：`etc/config.yaml`（已忽略版本控制、权限 0600）；模板：`etc/config.example.yaml`。连接配置为 `PostgreSQL.DSN`、`PostgreSQL.Schema`，环境变量 `DATABASE_URL` 可覆盖 DSN。
- 本地管理员用户名 `admin`，密码位置为私有配置中的 `Auth.AdminPassword`。读配置时避免将密钥输出到日志、文档或工具结果。
- 2026-09-10 曾按用户要求原样同步本地配置至 34（备份、SHA-256 一致、0600）；后续域名排障仅将远端 BaseURL 改为 `https://lumen.o.atdplus.cn/adn-report`，保留其他配置，数据库连接检查和 ADN 重启通过。同日按用户要求将远端 Host 改为 `0.0.0.0`、BaseURL 改为 `http://172.16.3.34:18080`，配置及 PG 连接校验、重启与跨机器访问检查通过；备份为远端 `etc/config.yaml.bak-ip-20260910164945`（0600）。远端已不再与本地文件逐字相同；默认部署继续保留远端配置。
- 钉钉配置为 `Auth.ClientID/ClientSecret`，管理员名单为 `Auth.AdminUnionIDs`。截至最后交接，真实钉钉应用凭据和回调配置尚待用户完成；授权流程已有模拟钉钉响应的集成测试，不能称真实扫码已验证。
- PostgreSQL 启动初始化由 `store.Open` 执行：独立 schema、事务内建表、advisory lock 防并发初始化、自动同步中文注释。`store.Connect` 仅连接不建表。
- 6 张表、39 个字段都有中文注释。新增或修改字段必须同步其用途注释。旧脚本 `internal/store/migrations/mysql/` 仅作历史记录，不能在 PostgreSQL 执行。
- 数据关系：`users → uploads ← adn_accounts`，`uploads → settlement_rows`；`sessions` 保存会话哈希，`oauth_states` 保存一次性授权状态。
- `uploads.user_id` 是实际上传身份和权限依据；`uploader_name` 是当时姓名快照；`operator_name` 为可修改的业务负责人，不能用它判断权限。
- 明细固定字段为 DATE + NUMERIC(24,6)，批次金额 NUMERIC(30,6)；JSONB `extra_fields` 保存额外列，`source_values` 保存整行原始值，`columns_json` 保留表头顺序。时间戳 TIMESTAMPTZ、连接 UTC，业务日期不做时区转换。
- 2026-09-10 切换前查得旧 MySQL 上传记录与 ADN 账户均为 0，因此没有迁移业务数据；旧库保留，旧登录会话不迁移。这是当时状态，不代表当前库永远为空。

## 已确定的业务规则

- 必须有四列：**日期、结算数、结算单价、结算金额**。严格校验，不自动将其他列映射成标准列，不推算缺失数值或强制金额等于数量乘单价。
- 根目录 6 个原始 Excel 示例均缺少部分标准列，因此被拒绝是预期行为。下载页面模板可体验完整流程；不要为了让样例通过而放松校验或修改原文件。
- 仅 `.xlsx`，20 MB、每工作表最多 20,000 个数据行位置、100 列；首行为表头，每次明确选择一个工作表；跳过空行，拒绝重复或空列名。
- 标准数字允许最多 18 位整数、6 位小数及负数调整。Go 有理数精确汇总，不用浮点累计金额；公式只读取文件内缓存值。
- 2026-09-10 页面明细和上传预览统一优先展示「日期、结算数、结算单价、结算金额」，不展示 Excel 行号；结算数、结算单价、结算金额统一四舍五入显示两位小数（含统计、列表、预览、明细，使用字符串/BigInt 避免大数精度损失）；日期及额外字段不强制数字化，其他列保留相对顺序。仅调整 `DataTable.vue` 展示顺序，CSV 与数据库仍保留原列顺序和源行号。
- 额外列保留底层单元格值；CSV 用 UTF-8 BOM、标准日期与六位小数，保留原列顺序。归档在 `data/csv/`，不额外存原始 XLSX。
- 同一上传人、ADN 账户、文件哈希、工作表防重复。批次与明细事务写入；失败清理当次 CSV。文件系统与数据库不是共同事务，异常强杀可能留无引用 CSV。
- 普通用户只能访问本人的列表、明细和 CSV；管理员看全部。ADN 账户目录对登录用户共享，但不授予他人上传记录权限。
- OAuth state 绑定浏览器、10 分钟过期且一次性使用；登录 Cookie 为 HttpOnly/SameSite、7 天有效；写接口检查 Origin 和 `X-Requested-With: ADN`。

## 按任务定位

| 修改内容 | 文件 |
| --- | --- |
| 启动、嵌入页面、会话清理 | `main.go` |
| 配置和环境变量 | `internal/config/config.go` |
| PostgreSQL 连接、初始化、表字段注释 | `internal/store/store.go`、`internal/store/schema.sql` |
| Excel 解析、日期数字校验、CSV 输出 | `internal/importer/excel.go` |
| go-zero 服务装配、路由 | `internal/server/server.go`、`internal/handler/routes.go` |
| 依赖注入、请求响应类型、HTTP 编解码 | `internal/svc/servicecontext.go`、`internal/types/`、`internal/response/` |
| 认证、请求来源、导入并发限制 | `internal/middleware/` |
| 钉钉、本地登录、会话与管理员判断 | `internal/handler/auth/`、`internal/logic/auth/`、`internal/dingtalk/client.go` |
| 账户 HTTP/业务处理 | `internal/handler/account/`、`internal/logic/account/` |
| 预览、上传、列表、明细、下载 HTTP/业务处理 | `internal/handler/report/`、`internal/logic/report/`（按功能拆文件） |
| 业务 SQL 与事务 | `internal/model/{authmodel,accountmodel,uploadmodel}.go` |
| Vue 布局、页面组件、交互和样式 | `web/src/App.vue`、`web/src/components/`、`web/src/composables/useWorkspace.js`、`web/src/style.css` |
| 目标机部署、systemd、nginx | `deploy_test.sh`、`deploy/activate.sh`、`deploy/adn-report-ai.service`、`deploy/nginx-location.conf`、`deploy/nginx-edge-location.conf` |
| 前端请求、构建与开发代理 | `web/src/api.js`、`web/package.json`、`web/vite.config.js` |
| SQL/API 集成与 go-zero 路由验证 | `internal/server/server_test.go`、`internal/server/routes_test.go` |
| Excel、配置、错误脱敏验证 | `internal/importer/excel_test.go`、`internal/config/config_test.go`、`internal/store/store_test.go` |

## 测试机部署

- 2026-09-10 简化部署：脚本仅构建、上传、启动并检查远端应用 18080，不再操作 nginx；首次部署及 `--sync-config` 原样复制本地私有配置，默认保留远端配置。main 启动日志显示 Host/Port，不把 BaseURL 当作监听地址。本次未启动本地服务或操作远端运行状态。

- 2026-09-10 已执行 `deploy_test.sh --stage`：目标 `root@172.16.3.34`，Linux amd64 二进制 `/data/adn-report-ai/adn-report`（与本地产物 SHA-256 一致），私有配置 `etc/config.yaml` 权限 0600，CSV `data/csv/`。服务 `adn-report-ai.service`，后端现监听 `0.0.0.0:18080`（同日按用户要求调整）；日志 `/data/log/go/adt-go/adn-report-ai/`。
- 当前直接访问入口为 `http://172.16.3.34:18080/`，BaseURL 与其一致。原域名入口 `https://lumen.o.atdplus.cn/adn-report/` 的 nginx 配置保留，但当前登录来源与回跳以 IP 入口为准。34 的 `/usr/local/openresty/nginx/conf/sites-enable/lumenx.conf` 已引用 `conf/snippets/adn-report.location.conf`；本机以域名 Host 访问该路径返回 ADN，API 返回 JSON。配置已备份且 nginx -t 通过。
- 公网入口尚未修复：域名解析 106.75.66.84，公网 `/adn-report/` 及其 API 均返回 LumenX HTML；唯一标记请求未出现于 34 的 nginx 日志或针对 80/17177/3000 端口的捕获，本机对照请求可见。用户无线上入口权限；已提供 `deploy/nginx-edge-location.conf` 给入口管理员，仅代理 ADN 前缀至 34，不能安装到 34 本机。
- 2026-09-10 `deploy_test.sh` 已增加 Node 版本预检与自动选择（`scripts/node_env.sh`）：当前 Node 不满足 >=22.12 时依次检查 Homebrew/nvm 已安装版本，在 SSH 前完成检查，不修改用户全局 Node 设置。
- 新增 `adn-report -f etc/config.yaml -check`：仅校验配置和 PG 连接，不建表、不启动 HTTP。子路径变更已通过 Vue 构建、Go 测试/vet，补充 `TestBaseURLPrefix` 与 `TestReverseProxyPrefix` 验证 URL、Origin、Cookie 和回跳。部署脚本语法检查及实际文件暂存流程通过。

## 命令与已有验证

- `make web-install`：按 `web/package-lock.json` 安装前端依赖；`make web-build`：只构建 Vue。首次直接运行 Go 构建/测试前须生成 `web/dist`，该目录不纳入版本控制。
- `make run`：初始化缺失的私有配置、构建 Vue 并启动，默认 `http://127.0.0.1:18080`。写请求来源要求地址与 `BaseURL` 一致，不混用 localhost。
- `make build`：构建 Vue 后生成 `bin/adn-report`；`./bin/adn-report -f etc/config.yaml` 启动二进制。
- `make check`：Vue 生产构建、Go 测试与 go vet；`make test`：Vue 构建与 Go 测试。Makefile 已把 GOCACHE 指向 `.cache/go-build`。
- `make integration`：实际连接 PostgreSQL，用随机 `adn_report_test_*` schema 验证并自动删除。默认读本地配置，可用 `ADN_TEST_DATABASE_URL` 指定专用测试库；不在业务 schema 填测试数据。
- 截至 2026-09-10：PostgreSQL 版本编译、Go 检查、真实数据库集成测试通过；含建表与注释、重复初始化、入库、防重复、筛选分页、JSONB、权限、会话与模拟 OAuth。实际运行服务的管理员登录、账户列表、上传列表、退出也已验证。
- 2026-09-10 Vue/go-zero 版本已通过生产构建、Go 测试及 vet、真实 PostgreSQL 集成、独立 18081 Chrome 完整验收（登录、账户、上传预览及入库、金额、明细、CSV、缺列提示、移动端和退出，无 JS 运行错误）。分层后再次通过 Go 检查与 PostgreSQL 集成；最终分层版尚未完成全流程浏览器复验，后续须按当前私有配置重新验证。`internal/logic/report/uploadlogic_test.go` 用模型接口替身独立验证上传身份和 CSV 成功保留/失败清理。补充路由测试覆盖静态资源、未登录访问、来源校验及框架默认 1 MB 限制兼容。真实钉钉扫码仍未验证。
- `scripts/browser_smoke.cjs` 依赖 `.cache/browser` 下 Playwright、Node >=18、Chrome 和独立 18081 测试实例；支持 `ADN_E2E_CONFIG`（YAML/JSON）、`ADN_E2E_OUTPUT`、`ADN_E2E_BASE_URL`。本次临时环境为 `.cache/vue-e2e`，schema 已删除；历史 `.cache/e2e` 可能含旧配置，均不可直接当当前环境启动。
- `.cache/`、`bin/`、`data/` 均为本地产物，不是理解业务的首选阅读入口。文档修改无需启动或运行应用测试。
