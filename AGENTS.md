# ADN 结算平台：项目上下文与协作约定

供新会话快速接手。先阅读本文件，再按任务定位相关代码；完整使用说明见 `README.md`。本文件记录已确认的事实，不代替当前代码或必要的验证。

## 接手方式

- 使用中文沟通。用户希望减少重复探索和 token 消耗。
- 不要为了了解项目重新启动服务、连接数据库、全量扫描目录或重跑全部测试。优先使用下面的代码索引，只检查与本次修改相关的文件。
- 已有验证结果属于历史快照。涉及相应行为的代码发生变化时，执行必要的针对性验证，不要将历史通过当成本次通过。
- 2026-09-10 独立 18081 测试实例已清理。172.16.3.34 的新版 ADN 已于 20:17:55 启动，当前 active；已登录后实测 `/api/template` 返回七列表头并退出。模板排查未重启服务或操作 OpenResty（80）。ADN BaseURL 为 `http://172.16.3.34:18080`。用户确认公网域名不指向此机后，已撤销本次 nginx `/adn/` 分流并平滑重载，LumenX 代理保留 17177。后续按需操作，不复用历史 PID。
- 每次完成架构、数据库、业务规则或运行状态变更后，更新本文件对应内容及日期；保持精简，不追加聊天流水账。不保存密码、令牌或完整含密码的连接串。

## 项目概况

- 当前目录就是完整项目，名称 `adn-report-ai`；不要再创建一层项目目录。
- 2026-09-10 已迁移为 Go 1.25 + go-zero REST：`rest.RestConf`、`rest.Server`、`rest.Route`、框架中间件与优雅退出；`database/sql` + pgx v5，Excelize v2。已按 `handler → logic → model` 分层，`svc.ServiceContext` 注入模型接口与客户端，`middleware` 负责认证/来源/导入并发限制，`types` 定义请求响应；handler/logic 按 auth/report 模块组织。`server` 仅装配，不保存 SQL 或业务规则。
- 2026-09-10 前端已迁移为 Vue 3 Composition API + 单文件组件 + Vite，源码 `web/src/`，构建产物 `web/dist` 通过 `go:embed` 随 Go 服务发布。`make build/run` 先构建 Vue；首次先 `make web-install`。Node >=22.12；本机默认 Node 16，可用 `export PATH=/opt/homebrew/bin:$PATH` 切到已有 Node 23。
- 开发热更新用 `make backend CONFIG=etc/config.dev.yaml` + `make web-dev`；开发私有配置的 BaseURL 设为 `http://127.0.0.1:5173`，Vite 同源代理 `/api` 和 `/auth` 到 18080。详细步骤见 README。
- 支持 nginx 子路径部署：BaseURL 可为 `http://172.16.3.34/adn-report`，Origin 校验只取协议/主机，Cookie 和登录跳转带路径前缀；前端相对资源/API 地址兼容根路径和子路径。nginx 转发时去掉前缀。
- go-zero 请求上限固定 21 MiB（文件仍限 20 MB）、超时 180 秒；关闭可能泄露请求凭据的默认请求日志和 6060 调试服务。保留 Origin、Cookie 与现有安全响应头。
- 第一版已具备：钉钉授权、本地管理员登录、广告主数据展示、Excel 校验预览、CSV 归档、结算入库、上传记录筛选、动态字段明细、CSV 下载、个人与管理员权限。
- 架构最初参考 `../MCP`；钉钉流程参考 `/Users/chemanyu/workspace/python/lumenx/src/apps/comic_gen/{auth.py,api.py}`。流程已经落地，无相关需求不必重新读取参考项目，不复制其密钥。

## 数据库与配置

- **当前是 PostgreSQL，已完成 MySQL 切换。** 地址 `172.16.3.25:5432`，数据库 `inhouse_ads`，业务 schema `adn_report`。不要改回旧 MySQL，也不要把业务表放进 `public`。
- 私有配置：`etc/config.yaml`（已忽略版本控制、权限 0600）；模板：`etc/config.example.yaml`。连接配置为 `PostgreSQL.DSN`、`PostgreSQL.Schema`，环境变量 `DATABASE_URL` 可覆盖 DSN。
- 本地管理员用户名 `admin`，密码位置为私有配置中的 `Auth.AdminPassword`。读配置时避免将密钥输出到日志、文档或工具结果。
- 2026-09-10 曾按用户要求原样同步本地配置至 34（备份、SHA-256 一致、0600）；后续域名排障仅将远端 BaseURL 改为 `https://lumen.o.atdplus.cn/adn-report`，保留其他配置，数据库连接检查和 ADN 重启通过。同日按用户要求将远端 Host 改为 `0.0.0.0`、BaseURL 改为 `http://172.16.3.34:18080`，配置及 PG 连接校验、重启与跨机器访问检查通过；备份为远端 `etc/config.yaml.bak-ip-20260910164945`（0600）。随后曾为 nginx 子路径部署切到域名 `/adn`；因用户仍使用 IP:18080 导致退出来源校验 403，现已恢复 `http://172.16.3.34:18080`（详见测试机部署）。远端已不再与本地文件逐字相同；默认部署继续保留远端配置。
- 钉钉配置为 `Auth.ClientID/ClientSecret`，管理员名单为 `Auth.AdminUnionIDs`。截至最后交接，真实钉钉应用凭据和回调配置尚待用户完成；授权流程已有模拟钉钉响应的集成测试，不能称真实扫码已验证。
- PostgreSQL 启动初始化由 `store.Open` 执行：独立 schema、事务内建表、advisory lock 防并发初始化、自动同步中文注释。`store.Connect` 仅连接不建表。
- 2026-09-10 已完成 `172.16.3.25:5432/inhouse_ads` 的 `adn_report` 业务迁移：新增 agency/advertiser/task_name、结算三列允许 NULL、删除 source_values、调整索引并同步 41 个字段注释。随后按用户明确要求清空全部旧结算数据：删除 416 条明细和 3 个上传批次，业务表现为 0 批次/0 明细；保留 1 个 ADN 账户、2 个用户及登录配置。自增序列未重置。
- 迁移前备份 schema 为 `adn_report_backup_20260910_120913`，清空前备份为 `adn_report_backup_20260910_121253`（均在同库，时间戳 UTC，含数据及原字段/约束/索引/序列信息，复制指纹已校验）。34 的 3 个旧 CSV 已移至 `/data/adn-report-ai/backups/data-cleanup-20260910-121253/csv/`，业务 `data/csv/` 已为空。清理时旧 ADN 已停止；随后新版于 20:17:55 启动，模板实测通过。不能回退运行依赖旧表结构的旧版程序。
- 2026-09-11 当前代码为 6 张表、44 个字段，均有中文注释（移除 ADN 账户及同名文件复用版本尚未迁移业务库）。新增或修改字段必须同步其用途注释。旧脚本 `internal/store/migrations/mysql/` 仅作历史记录，不能在 PostgreSQL 执行。
- 数据关系：`users → uploads`，`uploads → settlement_rows/upload_history`；`sessions` 保存会话哈希，`oauth_states` 保存一次性授权状态。
- `uploads.user_id` 是实际上传身份和权限依据；`uploader_name` 是当时姓名快照；`operator_name` 为可修改的业务负责人，不能用它判断权限。
- 明细固定字段为 DATE + NUMERIC(24,6)，批次金额 NUMERIC(30,6)；JSONB `extra_fields` 保存七列之外的额外字段，已移除 `source_values`，`columns_json` 保留表头顺序；agency/advertiser/task_name 对应代理商/广告主/任务名称，结算三列允许 NULL。时间戳 TIMESTAMPTZ、连接 UTC，业务日期不做时区转换。
- 2026-09-10 切换前查得旧 MySQL 上传记录与 ADN 账户均为 0，因此没有迁移业务数据；旧库保留，旧登录会话不迁移。这是当时状态，不代表当前库永远为空。

## 已确定的业务规则

- 2026-09-10 已移除独立 ADN 账户逻辑：广告主即业务账户，上传仅传 file/sheet/operator；删除账户页面/弹窗/API/model 及 account_id 参数。列表/明细显示每批次的 advertisers，统计为 advertiser_count，可搜索广告主和代理商（匹配批次，统计为该批次全部有效行）。一份文件可含多个广告主。迁移将删除 uploads.account_id 和 adn_accounts，保留所有上传及明细；同一用户原不同手工账户的相同业务键在下次更新时合并。登录用户隔离保持不变。本次仅改代码和迁移定义，未操作业务库或运行服务。


- 新导入必须有七列：**代理商、广告主、日期、任务名称、结算数、结算单价、结算金额**。前四列为非空业务匹配字段；后三列允许 0 或空（空保留，不推算），不强制金额等于数量乘单价。单文件重复业务键拒绝并提示源行号。
- 根目录 6 个原始 Excel 示例均缺少部分标准列，因此被拒绝是预期行为。下载页面模板可体验完整流程；不要为了让样例通过而放松校验或修改原文件。
- 仅 `.xlsx`，20 MB、每工作表最多 20,000 个数据行位置、100 列；首行为表头，每次明确选择一个工作表；跳过空行，拒绝重复或空列名。
- 标准数字允许最多 18 位整数、6 位小数及负数调整。Go 有理数精确汇总，不用浮点累计金额；公式只读取文件内缓存值。
- 2026-09-10 页面明细和上传预览统一优先展示「代理商、广告主、日期、任务名称、结算数、结算单价、结算金额」，不展示 Excel 行号；结算数、结算单价、结算金额统一四舍五入显示两位小数（含统计、列表、预览、明细，使用字符串/BigInt 避免大数精度损失）；日期及额外字段不强制数字化，其他列保留相对顺序。仅调整 `DataTable.vue` 展示顺序，CSV 与数据库仍保留原列顺序和源行号。
- 额外列保留底层单元格值；CSV 用 UTF-8 BOM、标准日期与六位小数，保留原列顺序。归档在 `data/csv/`，不额外存原始 XLSX。
- 2026-09-10 已改为业务键逐行更新：仅在同一登录用户的数据内，按代理商 + 广告主 + 日期 + 任务名称匹配，名称去首尾空白、区分大小写且最多 255 字；文件名/工作表不参与。匹配则更新整行（包括以空值清空旧数值和替换 extra_fields），不匹配则新增，本次缺少的旧键保留。管理员上传也不能更新其他人的记录。
- 2026-09-11 同用户同文件名同工作表复用 uploads 记录（区分大小写），每次成功导入新增 upload_history 保存时间、人员、新增/更新行数和不可变归档信息。文件名/工作表不同新建记录，明细业务键更新及遗漏旧行保留规则不变。列表按 updated_at 排序，默认隐藏空记录，show_empty=true 可查看；详情折叠展示历史，API 返回 reused/inserted_rows/updated_rows。Snapshot 使用只读可重复读事务保证元数据、历史、明细和 CSV 一致。
- 2026-09-11 启动迁移合并旧同名同工作表批次，保留最新 ID、全部剩余明细与归档引用，旧重复 ID 不再可访问；旧新增/更新数未知则 NULL。表头以最新上传为先补充旧列，取消源行号唯一约束，排序使用 source_row/id，前端不再用源行号作唯一 key。新增文件范围唯一索引，用户事务锁保护复用及历史写入。升级统一停止旧实例并备份数据库；本次未操作业务库或运行服务。
- `SaveWithRows` 按 schema + 登录用户取得事务 advisory lock，避免同范围多实例并发和重叠键死锁；失败回滚数据库并清理当次新 CSV。迁移添加三个业务名称列、从旧 extra_fields 回填有效值、结算三列改可空、移除 source_values 和旧哈希约束/文件名索引，增加业务查询索引。历史缺键行保留，历史重复键只在明确更新该键时合并；索引不唯一以兼容历史数据。升级统一切换所有实例。
- 普通用户只能访问本人的列表、明细和 CSV；管理员看全部。广告主直接来自文件，不再维护或选择独立 ADN 账户。
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
| 预览、上传、列表、明细、下载 HTTP/业务处理 | `internal/handler/report/`、`internal/logic/report/`（按功能拆文件） |
| 业务 SQL 与事务 | `internal/model/{authmodel,uploadmodel}.go` |
| Vue 布局、页面组件、交互和样式 | `web/src/App.vue`、`web/src/components/`、`web/src/composables/useWorkspace.js`、`web/src/style.css` |
| 目标机部署、systemd、nginx | `deploy.sh`、`scripts/build-linux.sh`、`deploy_test.sh`、`deploy/activate.sh`、`deploy/adn-report-ai.service`、`deploy/nginx-location.conf`、`deploy/nginx-edge-location.conf` |
| 前端请求、构建与开发代理 | `web/src/api.js`、`web/package.json`、`web/vite.config.js` |
| SQL/API 集成与 go-zero 路由验证 | `internal/server/server_test.go`、`internal/server/routes_test.go` |
| Excel、配置、错误脱敏验证 | `internal/importer/excel_test.go`、`internal/config/config_test.go`、`internal/store/store_test.go` |

## 线上仓库部署

- 2026-09-10 模板排查发现 `dist-linux/adn-report` 仍为 17:20 的旧产物，现已用当前源码和 Vue 构建重新生成 Linux amd64 产物；尚未提交/推送或部署该产物。34 已运行另一次部署的新版本，实测模板为七列。使用 `deploy.sh` 时必须同时更新预编译产物和源码。

- 2026-09-10 `deploy.sh` 已由其他项目适配为 ADN：在 Linux 仓库执行 git pull，使用 `dist-linux/adn-report` 产物，保留 `etc/config.yaml`，先 `-check` 再安装/重启 `adn-report-ai` systemd 服务，以仓库为工作目录运行 `bin/adn-report -f <仓库>/etc/config.yaml`，日志进入 journal。健康检查 `/api/auth/config` 默认 18080，可通过 PORT 指定；不操作 nginx。失败恢复旧二进制与 unit，备份在 `.cache/deploy-backups/`。
- 开发机执行 `bash scripts/build-linux.sh`（兼容 Node 自动选择、Vue 构建、Linux 默认 amd64 交叉编译），将产物与源码一起提交推送；服务器无需 Go/Node。本次仅修改脚本和说明，bash 语法检查及模拟成功部署/健康失败回退/缺配置停止通过，未生成或提交二进制、未实际部署线上机器。

## 测试机部署

- 2026-09-10 简化部署：脚本仅构建、上传、启动并检查远端应用 18080，不再操作 nginx；首次部署及 `--sync-config` 原样复制本地私有配置，默认保留远端配置。main 启动日志显示 Host/Port，不把 BaseURL 当作监听地址。本次未启动本地服务或操作远端运行状态。

- 2026-09-10 已执行 `deploy_test.sh --stage`：目标 `root@172.16.3.34`，Linux amd64 二进制 `/data/adn-report-ai/adn-report`（与本地产物 SHA-256 一致），私有配置 `etc/config.yaml` 权限 0600，CSV `data/csv/`。服务 `adn-report-ai.service`，监听配置为 `0.0.0.0:18080`（当前新版 active）；日志 `/data/log/go/adt-go/adn-report-ai/`。
- 2026-09-10 当前访问入口及远端 BaseURL 为 `http://172.16.3.34:18080`。用户已确认 `lumen.o.atdplus.cn` 不能到达此机，不再作为当前部署入口。为域名 `/adn` 临时修改 BaseURL 曾导致 IP 入口退出请求来源校验 403，恢复后该 Origin 的无会话退出请求正常返回 401；配置/PG 校验及重启通过。
- 2026-09-10 按用户要求回退 34 的 `/usr/local/openresty/nginx/conf/sites-enable/lumenx.conf`：删除本次新增 `/adn` 和 `/adn/` location，LumenX 原业务代理为 17177，首页保留 LumenX 静态文件，无 ADN 路由；旧 `/adn-report/` snippet 未引用。nginx -t、平滑重载、首页及 ADN 认证配置 API 检查通过。回退前备份为 `lumenx.conf.bak-rollback-20260910171334`。仓库 nginx 子路径/公网入口示例为历史方案，非当前部署配置。
- 2026-09-10 `deploy_test.sh` 已增加 Node 版本预检与自动选择（`scripts/node_env.sh`）：当前 Node 不满足 >=22.12 时依次检查 Homebrew/nvm 已安装版本，在 SSH 前完成检查，不修改用户全局 Node 设置。
- 新增 `adn-report -f etc/config.yaml -check`：仅校验配置和 PG 连接，不建表、不启动 HTTP。子路径变更已通过 Vue 构建、Go 测试/vet，补充 `TestBaseURLPrefix` 与 `TestReverseProxyPrefix` 验证 URL、Origin、Cookie 和回跳。部署脚本语法检查及实际文件暂存流程通过。

## 命令与已有验证

- 2026-09-11 同名文件复用版本已通过 Vue 生产构建、相关 Go 测试/vet 及真实 PostgreSQL 独立 schema 集成：同 ID 复用、新增与更新计数、遗漏旧行保留、源行号重叠、额外列/CSV 保留、空记录筛选、历史合并及重复迁移、权限隔离、八实例并发与失败回滚。测试 schema 自动清理；Linux amd64 产物已更新。本次未执行浏览器验收，未提交、部署或迁移业务库。

- 2026-09-10 移除 ADN 账户版本已通过相关 Go 检查、Vue 构建及真实 PostgreSQL 独立 schema 集成：无账户参数上传、多广告主批次与统计、广告主搜索、登录用户隔离、旧账户表/字段迁移保留明细。浏览器验收脚本已适配，但本次未执行完整浏览器验收。`dist-linux/adn-report` 已重新生成并包含移除账户后的前后端；尚未部署或迁移业务库。

- 2026-09-10 七列业务键版本已通过解析/逻辑/路由测试、Vue 构建和真实 PostgreSQL 独立 schema 集成：跨文件/工作表更新、未匹配旧行保留、NULL/0 区分、账号及管理员写入隔离、CSV/明细一致性、事务失败回滚、历史字段迁移及八实例并发。测试 schema 自动清理；业务 schema 随后已迁移完成（见数据库与配置），34 新版已运行，实际模板接口验证通过，完整生产上传流程未在本次排查中执行。

- `make web-install`：按 `web/package-lock.json` 安装前端依赖；`make web-build`：只构建 Vue。首次直接运行 Go 构建/测试前须生成 `web/dist`，该目录不纳入版本控制。
- `make run`：初始化缺失的私有配置、构建 Vue 并启动，默认 `http://127.0.0.1:18080`。写请求来源要求地址与 `BaseURL` 一致，不混用 localhost。
- `make build`：构建 Vue 后生成 `bin/adn-report`；`./bin/adn-report -f etc/config.yaml` 启动二进制。
- `make check`：Vue 生产构建、Go 测试与 go vet；`make test`：Vue 构建与 Go 测试。Makefile 已把 GOCACHE 指向 `.cache/go-build`。
- `make integration`：实际连接 PostgreSQL，用随机 `adn_report_test_*` schema 验证并自动删除。默认读本地配置，可用 `ADN_TEST_DATABASE_URL` 指定专用测试库；不在业务 schema 填测试数据。
- 截至 2026-09-10：PostgreSQL 版本编译、Go 检查、真实数据库集成测试通过；含建表与注释、重复初始化、入库、防重复、筛选分页、JSONB、权限、会话与模拟 OAuth。实际运行服务的管理员登录、账户列表、上传列表、退出也已验证。
- 2026-09-10 Vue/go-zero 版本已通过生产构建、Go 测试及 vet、真实 PostgreSQL 集成、独立 18081 Chrome 完整验收（登录、账户、上传预览及入库、金额、明细、CSV、缺列提示、移动端和退出，无 JS 运行错误）。分层后再次通过 Go 检查与 PostgreSQL 集成；最终分层版尚未完成全流程浏览器复验，后续须按当前私有配置重新验证。`internal/logic/report/uploadlogic_test.go` 用模型接口替身独立验证上传身份和 CSV 成功保留/失败清理。补充路由测试覆盖静态资源、未登录访问、来源校验及框架默认 1 MB 限制兼容。真实钉钉扫码仍未验证。
- `scripts/browser_smoke.cjs` 依赖 `.cache/browser` 下 Playwright、Node >=18、Chrome 和独立 18081 测试实例；支持 `ADN_E2E_CONFIG`（YAML/JSON）、`ADN_E2E_OUTPUT`、`ADN_E2E_BASE_URL`。本次临时环境为 `.cache/vue-e2e`，schema 已删除；历史 `.cache/e2e` 可能含旧配置，均不可直接当当前环境启动。
- `.cache/`、`bin/`、`data/` 均为本地产物，不是理解业务的首选阅读入口。文档修改无需启动或运行应用测试。
