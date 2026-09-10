# ADN 结算数据平台

第一版：钉钉登录、Excel 校验预览、运营人员与 ADN 账户关联、CSV 归档、PostgreSQL 结算明细、上传记录和明细页面。

后端使用 Go 1.25、go-zero REST（服务启动、路由、中间件和 YAML 配置）、`database/sql` + pgx PostgreSQL driver，Excel 使用 Excelize。前端使用 Vue 3 Composition API、单文件组件和 Vite；构建产物 `web/dist` 通过 `go:embed` 打包到 Go 服务，部署时只需二进制、私有配置及 CSV 数据目录。

## 启动

构建需要 Node >=22.12（建议 Node 22 LTS）及 npm。当前机器默认终端为 Node 16，已有 `/opt/homebrew/bin/node` 为 Node 23，可先执行 `export PATH=/opt/homebrew/bin:$PATH` 并用 `node --version` 确认。

```sh
make web-install   # 首次安装或 package-lock.json 更新后执行，使用锁定版本
make run           # 构建 Vue 后启动 go-zero 服务
```

打开 **http://127.0.0.1:18080**。首次运行会创建 `etc/config.yaml` 并生成随机本地管理员密码；已有配置不会覆盖。本项目当前目录已经生成配置。

- 本地管理员账号：`admin`。
- 密码：`etc/config.yaml` 中 `Auth.AdminPassword`。
- 当前 PostgreSQL：`172.16.3.25:5432/inhouse_ads`，用户 `inhouse_ads`，连接密码仅在本地 `etc/config.yaml` 中保存。
- 服务连接已有数据库 `inhouse_ads`，自动创建独立的 `adn_report` schema、6 张业务表及中文注释，不要求创建数据库的权限。
- CSV 保存在 `data/csv/`，默认仅应用进程用户可读写。
- `etc/config.yaml`、`data/`、`bin/` 和缓存已加入 `.gitignore`。

请使用与 `BaseURL` 完全一致的访问地址（默认 `127.0.0.1`，不是 `localhost`），写请求会校验来源。如果更换端口或域名，同时修改 `BaseURL`、`Host`、`Port` 和钉钉回调配置。

```sh
make build
./bin/adn-report -f etc/config.yaml
```

## 开发结构与 Vue 热更新

| 位置 | 职责 |
| --- | --- |
| `main.go` | 嵌入前端构建产物、启动 go-zero REST、定期清理会话及优雅退出 |
| `internal/config/config.go` | 嵌入 `rest.RestConf`，扩展 PostgreSQL、登录和站点配置 |
| `internal/server/server.go` | go-zero 服务与前端静态文件装配，不包含业务处理 |
| `internal/handler/routes.go` | 路由及中间件绑定，路径参数为 `:id` |
| `internal/handler/{auth,account,report}` | 解析 HTTP/JSON/multipart、调用 logic、设置 Cookie/响应/下载头 |
| `internal/logic/{auth,account,report}` | 登录与结算业务规则、数据权限、Excel 校验及 CSV 归档协调；不依赖 HTTP 请求或数据库连接 |
| `internal/model/` | Auth/Account/Upload 模型接口与 PostgreSQL 实现；所有业务 SQL、批次明细和会话事务 |
| `internal/svc/servicecontext.go` | 注入配置、模型接口、钉钉客户端与共享限流状态 |
| `internal/middleware/` | 登录会话解析、请求来源校验、安全响应头和导入并发限制 |
| `internal/types/` | 请求与响应结构；与数据库实体分开定义 |
| `internal/dingtalk/`、`internal/response/` | 钉钉外部 API 适配、统一 HTTP 编解码与错误输出 |
| `internal/store`、`internal/importer` | 数据库连接/建表，以及独立的 Excel/CSV 解析工具 |
| `web/src/App.vue` | 工作台布局和共享状态注入 |
| `web/src/components/` | 登录、列表、账户、上传和明细组件 |
| `web/src/composables/useWorkspace.js` | Vue 响应式状态、异步操作与过期响应保护 |
| `web/src/api.js` | 同源 API 请求与错误处理 |
| `web/src/style.css` | 页面样式 |

例如上传请求的调用链是 `handler/report/reporthandler.go → logic/report/uploadlogic.go → model/uploadmodel.go`。handler 只读取文件和表单；logic 完成校验、身份绑定、CSV 写入及失败清理；model 用同一事务保存批次与全部明细。登录 Cookie 属于 handler，登录身份与管理员判断属于 logic，用户/会话/OAuth state 的 SQL 属于 model。业务模块按 `auth/account/report` 组织，数据访问层使用 `model` 命名。

前端使用原有界面与接口协议，金额继续按字符串展示，动态列由 Vue 模板转义渲染。两块工作区使用组件切换，目前无需 Vue Router 或 Pinia。

开发时可分开启动后端和 Vite：

1. 复制私有配置为 `etc/config.dev.yaml`（权限 0600），将其中 `BaseURL` 设为 `http://127.0.0.1:5173`，后端 `Host/Port` 仍为 `127.0.0.1:18080`。如需数据隔离，另设开发 schema 和 DataDir。
2. 终端一执行 `make backend CONFIG=etc/config.dev.yaml`。
3. 终端二执行 `make web-dev`，浏览器访问 **http://127.0.0.1:5173**。

Vite 将 `/api` 与 `/auth` 代理到后端，保留浏览器 Origin；无需放宽来源校验。后端地址变化时通过 `ADN_API_TARGET` 设置代理目标。钉钉开发回调需相应配置为 `http://127.0.0.1:5173/auth/callback`。生产启动继续使用原有 `etc/config.yaml` 与 `make run`。开发完成后分别 Ctrl+C 停止两个进程。

`make build` 总是先构建 Vue；修改前端后须重新构建二进制。`web/dist` 与 `web/node_modules` 不纳入版本控制。直接 `go build` / `go test ./...` 前，首次需执行 `make web-build`。`npm --prefix web run format` 可统一格式化前端源码。

go-zero REST 的请求上限固定为 21 MiB（包含 multipart 开销），实际文件仍限制 20 MB，请求超时 180 秒；保留连接读写时限。禁用可能输出请求凭据的框架请求日志和默认 6060 调试服务；应用错误日志继续保留。数据库结构、登录 Cookie、业务规则与既有配置兼容。

## 线上服务器部署（deploy.sh）

`deploy.sh` 在 Linux 服务器的项目仓库中执行：拉取代码、校验预编译产物和数据库连接、安装 systemd 服务、重启并检查 `/api/auth/config`。服务器需要 git、curl、systemd 和 sudo/root 权限，无需 Go 或 Node；脚本不操作 nginx。

先在开发机生成包含 Vue 页面的 Linux 产物（默认 amd64；ARM64 服务器可设 `GOARCH=arm64`）：

```bash
bash scripts/build-linux.sh
# 将本次代码修改和 dist-linux/adn-report 一起提交、推送到部署分支
```

`dist-linux/adn-report` 用于此部署方式的版本分发，不含私有配置；应随对应源码一并提交。首次在服务器准备 `etc/config.yaml`（参考模板，配置 PostgreSQL、登录和实际 BaseURL），然后运行：

```bash
./deploy.sh
# 如果配置使用其他端口，健康检查端口须保持一致
PORT=18081 ./deploy.sh
```

服务名 `adn-report-ai`，运行文件 `bin/adn-report`，以仓库为工作目录，启动参数为 `-f <仓库>/etc/config.yaml`；相对 DataDir 也按仓库目录解析。监听应允许本机回环健康检查（默认 `127.0.0.1:18080`，需要 IP 直连时设 `0.0.0.0:18080`，BaseURL 使用实际访问地址）。脚本保留私有配置，以 root 运行服务，日志用 `sudo journalctl -u adn-report-ai` 查看。更新前备份至 `.cache/deploy-backups/`，启动或健康检查失败时恢复旧程序和服务配置。

## 部署到 172.16.3.34

部署脚本参考 `../MCP/deploy_test.sh`，执行 `./deploy_test.sh`：安装锁定前端依赖、构建 Vue、交叉编译 Linux amd64、通过 SSH 上传，并用 systemd 管理服务。需要本机 Node >=22.12、Go 和到目标机的 root SSH 权限。脚本优先使用当前兼容 Node，版本过低时自动检查 Homebrew 和 nvm 已安装版本，只调整脚本进程的 PATH；找不到兼容版本时会在连接远端前报错。

- 访问地址：`https://lumen.o.atdplus.cn/adn-report/`。
- 程序/私有配置/归档：`/data/adn-report-ai/adn-report`、`etc/config.yaml`、`data/csv/`。
- 服务：`adn-report-ai.service`；后端只监听 `127.0.0.1:18080`。
- 日志：`/data/log/go/adt-go/adn-report-ai/`；部署备份：`/data/adn-report-ai/backups/`。
- nginx 在既有 `/usr/local/openresty/nginx/conf/sites-enable/lumenx.conf` 的 server 块内包含 `conf/snippets/adn-report.location.conf`，通过 `^~ /adn-report/` 独立代理，不占用 lumenx 的 `/auth` 等路径。

```sh
# 部署脚本会自动选择本机已有的兼容 Node，无需手动切换 PATH
./deploy_test.sh --stage        # 暂存应用文件，不启动/重启服务
./deploy_test.sh                # 默认保留远端已有私有配置，检查 PG 后启动并验证
./deploy_test.sh --sync-config  # 原样同步本地配置到远端，先备份旧配置
```

首次部署或指定 `--sync-config` 时，原样复制本地 `etc/config.yaml`，权限 0600，不改写域名、数据库或数据目录；默认保留已有远端配置。部署配置的 `Host/Port` 应为 `127.0.0.1:18080`，相对 DataDir 按 systemd 的工作目录 `/data/adn-report-ai` 解析。

部署只负责构建、上传和启动应用，在目标机器检查 `http://127.0.0.1:18080/api/auth/config`。启动前 `-check` 校验配置及数据库连接；替换程序和配置前备份，失败恢复已有文件。脚本不安装、修改或重载 nginx。`main.go` 按配置的 Host/Port 监听；BaseURL 用于登录与请求来源校验，不决定监听地址。

公网域名前面还有一层 HTTPS 入口代理。`deploy/nginx-edge-location.conf` 是给该入口管理员的配置，应放在 `lumen.o.atdplus.cn` 的 server 块内，**不能装到 172.16.3.34 的 nginx**。入口必须把 `/adn-report/` 完整转发至 `http://172.16.3.34`（`proxy_pass` 不加尾部 `/`），然后由 34 的既有配置转发至 18080 并去掉路径前缀。入口管理员执行 nginx 配置检查并 reload 后，才能通过域名访问。只检查 34 本机成功不能代替公网访问验收。

路径前缀是对外入口的一部分：nginx 转发时移除 `/adn-report/`，应用按 BaseURL 设置 Cookie 路径和登录回跳，Origin 只校验协议与主机。前端使用相对资源和 API 地址，同一个构建产物兼容根路径与子路径。钉钉部署回调应配置为 `https://lumen.o.atdplus.cn/adn-report/auth/callback`。

## PostgreSQL 配置

```yaml
PostgreSQL:
  DSN: "postgres://用户名:密码@主机:5432/数据库?sslmode=disable"
  Schema: adn_report
```

`DATABASE_URL` 环境变量可以覆盖 `PostgreSQL.DSN`。示例配置中的 DSN 留空，新环境首次执行 `make init` 后填写实际连接串再启动；当前私有配置状态见 `AGENTS.md`，启动前确认已填写 PostgreSQL.DSN。原有的钉钉和本地管理员配置保持不变。

应用只使用配置的业务 schema，连接的 `search_path` 不包含 `public`。账号需具备该 schema 的创建及表、索引、注释维护权限。启动建表在事务内执行，通过事务级 advisory lock 避免多个实例并发初始化同一 schema。手动执行 `schema.sql` 时先创建业务 schema 并设置 `search_path`。

此次切换前检查旧 MySQL：上传记录与 ADN 账户均为 0，无业务记录需要搬迁。旧 MySQL 保留，登录会话不迁移，切换后重新登录即可，原本地管理员密码有效。后续如需从其他已有数据库搬迁数据，需要另外迁移数据和对应 CSV 文件。

测试可以通过 `ADN_TEST_DATABASE_URL` 指定专用 PostgreSQL 数据库；未设置时读取本地 `etc/config.yaml`。测试只创建和清理自身随机命名的 `adn_report_test_*` schema。

## 使用流程

1. 钉钉登录或使用本地管理员登录。
2. 在「ADN 账户」中维护实际账户名称、账号或 ID，也可在上传弹窗新增。
3. 点击「上传结算文件」，选择 `.xlsx` 文件；多工作表文件需明确选择本次入库的工作表。
4. 查看校验结果和前 10 行预览；运营人员默认填入登录姓名，可修改。选择本次结算对应的 ADN 账户。
5. 确认后重新在服务端完整校验，保存 CSV，并在同一数据库事务中写入批次及全部明细。
6. 按文件名、运营人员、上传人员或 ADN 账户筛选记录；查看动态字段明细和下载 CSV。

页面不预置虚构结算记录；可下载模板验证正常上传流程。原目录 6 个示例文件保持不变。

## 文件规范与现有示例

首行必须为表头，必须包含 **日期、结算数、结算单价、结算金额**，校验时忽略列名前后空白。其他列名、顺序和原始单元格值保留，不根据列名猜测业务含义。

| 示例 | 缺失标准列 |
| --- | --- |
| 微博-汽水、微博-虎扑 | 结算数、结算单价、结算金额 |
| 汇川-闲鱼拉活 | 结算数、结算单价、结算金额 |
| 浙江飞猪网络-飞猪-拉活 | 结算数、结算单价 |
| 快应用-三只兔 | 结算数 |
| 淘宝闪购链接-饿了么（两个工作表） | 日期、结算数、结算单价、结算金额 |

当前按需求严格拒绝缺列文件。请由业务确认「推广量」「结算量级」「实际转化数」等字段对应关系，再在 Excel 中补充标准列；不会自动推算单价、结算数或金额，也不会强制金额等于结算数乘单价。

- 最大 20 MB、每工作表最多 20,000 个数据行位置、100 列；每次导入一个工作表。
- 空行跳过；重复列名、空表头、无表头的额外单元格、无数据行拒绝。
- 日期支持 Excel 日期序号（含 1904 日期系统）、`YYYY-MM-DD`、`YYYY/M/D`、中文年月日、常见日期时间。日期区间不会自动合并或截断。
- 数字必须明确填写，支持 18 位整数和最多 6 位小数；允许负数表示结算调整。空值、千分位、科学计数文本、超精度数值拒绝。
- 金额使用 Go 有理数精确汇总、PostgreSQL `NUMERIC`（SQL 中 `DECIMAL` 为同义类型） 保存，不用浮点数累计。
- 公式使用 Excel 文件内已缓存的值，不在服务器执行或重新计算公式；请在 Excel 中完成计算后保存。
- CSV 使用 UTF-8 BOM，标准列日期统一为 `YYYY-MM-DD`，数字统一 6 位小数；其他列保留读取到的底层单元格值，逗号与换行按 CSV 规则转义。原始公式表达式、样式、合并单元格和格式化显示不属于 CSV 数据。
- 源文件 SHA-256 + 工作表 + ADN 账户 + 上传用户构成防重复约束；文件内容变化视为新文件。
- 不额外保留原始 XLSX 文件，数据库保留原文件名、哈希、工作表、列顺序和每行原始值。

## 登录和权限

钉钉实现参考 `/Users/chemanyu/workspace/python/lumenx/src/apps/comic_gen/auth.py` 和 `api.py`，采用浏览器授权码、用户 token、用户个人信息流程。应用凭据从本项目配置读取，不复制或硬编码参考项目密钥。

配置 `etc/config.yaml`：

```yaml
BaseURL: http://127.0.0.1:18080
Auth:
  ClientID: "应用 AppKey / Client ID"
  ClientSecret: "应用 AppSecret / Client Secret"
  AdminUnionIDs:
    - "管理员的 unionId"
  AdminUsername: admin
  AdminPassword: "本地管理员密码，至少12位，空字符串禁用"
```

也支持环境变量 `DINGTALK_CLIENT_ID`、`DINGTALK_CLIENT_SECRET`、`ADN_ADMIN_PASSWORD`、`ADN_ADMIN_UNION_IDS`（逗号分隔）和 `DATABASE_URL`（完整 PostgreSQL 连接串）。

在钉钉开放平台为应用配置本服务回调 **`http://127.0.0.1:18080/auth/callback`**，或实际部署域名下的 `/auth/callback`，并开通登录与用户个人信息所需权限。服务器需要访问钉钉 API。真实扫码需完成这些外部配置后联调。

相关官方文档：[获取用户 token](https://open.dingtalk.com/document/orgapp-server/obtain-user-token)、[获取用户通讯录个人信息](https://open.dingtalk.com/document/orgapp-server/dingtalk-retrieve-user-information)。

- 以钉钉 `unionId` 唯一识别用户；用户表身份字段加 `ding:` 前缀，本地管理员加 `local:` 前缀。
- 管理员通过 `AdminUnionIDs` 配置，不按昵称识别。登录后左下角「查看登录身份」可查看 `ding:<unionId>`，配置时只填写冒号后的部分。
- 普通用户只能查看本人上传的批次、明细和 CSV；管理员查看全部上传。
- 「运营人员」是业务归属信息，可由上传者填写；**上传身份始终由服务器从会话确定**，不能通过运营姓名改变数据访问权限。
- ADN 账户目录对登录成员共享，可新增并选择；账户本身不授予他人的上传记录访问权。第一版不做账户编辑或删除，以保持历史关联稳定。
- OAuth state 保存在 PostgreSQL，并绑定发起授权的浏览器、10 分钟过期且仅能使用一次。回调兼容 `authCode` 与 `code`。
- 登录会话采用随机 HttpOnly、SameSite Cookie，数据库仅存 token 哈希，7 天有效，退出立即失效；HTTPS 站点自动设置 Secure。写请求校验 Origin 和自定义请求头。
- 本地管理员不提供公开注册；登录接口每分钟最多 20 次尝试。

## 数据模型

建表定义：[internal/store/schema.sql](internal/store/schema.sql)。

6 张表和 39 个字段均包含中文注释，通过 PostgreSQL 的 `COMMENT ON TABLE/COLUMN` 设置。每次启动自动同步注释。在数据库工具中打开 `inhouse_ads → Schemas → adn_report → Tables` 即可查看。旧 MySQL 的注释脚本仅作为历史文件保留在 `internal/store/migrations/mysql/`，不要在 PostgreSQL 执行。

| 表 | 用途 |
| --- | --- |
| `users` | 钉钉或本地登录身份、姓名、最近登录时间 |
| `sessions` | 登录会话哈希、用户、过期时间 |
| `oauth_states` | 一次性授权状态与过期时间 |
| `adn_accounts` | ADN 账户名称、唯一标识、创建人员 |
| `uploads` | 上传用户和当时姓名、运营人员、账户、文件和工作表、CSV 路径、列顺序、行数、金额合计、日期范围 |
| `settlement_rows` | 每行日期、结算数、结算单价、结算金额、`extra_fields` 差异字段、`source_values` 原始值、Excel 行号 |

关系为 `users → uploads ← adn_accounts`、`uploads → settlement_rows`。字段差异通过 JSONB 表达，不按渠道新建表，也不动态修改表结构。

标准数字为 `DECIMAL(24,6)`，批次合计为 `DECIMAL(30,6)`。按上传人、账户、创建时间、结算日期建立索引。登录、会话与上传时间使用 `TIMESTAMPTZ`，连接会话时区统一为 UTC，页面按浏览器时区显示；结算日期为 DATE，不进行时区偏移。

入库失败回滚全部数据库记录，并清理当次 CSV；重复上传不会留下新 CSV。数据库与文件系统不支持共同事务，进程在写文件后被强制终止可能留下无数据库引用的 CSV，备份时需要同时备份数据库与 `data/csv/`。第一版没有删除和覆盖导入接口。

## 验证

```sh
make check        # Vue 生产构建、Go 单元测试和 go vet
make integration  # 使用本地配置连接 PostgreSQL，创建随机测试 schema，完成后自动删除
```

`internal/logic/report/uploadlogic_test.go` 通过模型接口替身独立验证 CSV 归档与入库失败补偿，不需要启动 HTTP 服务或连接数据库。

另有 `TestRESTRoutingAndStaticAssets` 验证 go-zero 路由、静态资源、安全响应头及超过 1 MB 的请求不会被框架默认限制提前拦截。

覆盖 Excel 日期系统、精确金额、原始 ID / 特殊字符、缺列 / 重复列 / 错误数值、样例校验、真实 PostgreSQL 入库、重复上传清理、普通用户列表 / 明细 / 下载隔离、管理员访问、CSRF、退出和会话过期，同时检查 6 张表及 39 个字段的注释、重复初始化和不同筛选条件下的参数绑定。

## 主要接口

| 方法与路径 | 功能 |
| --- | --- |
| `GET /auth/login`、`GET /auth/callback` | 钉钉授权与回调 |
| `GET /api/auth/config` | 可用登录方式（不返回密钥） |
| `POST /api/auth/local`、`POST /api/auth/logout` | 本地管理员登录、退出 |
| `GET /api/me` | 当前用户 |
| `GET /api/accounts`、`POST /api/accounts` | ADN 账户目录、新增账户 |
| `POST /api/preview` | multipart：file、sheet，校验与前 10 行预览 |
| `POST /api/uploads` | multipart：file、sheet、operator、account_id，保存结算 |
| `GET /api/uploads?page=1&q=&account_id=` | 批次列表、筛选与统计，每页 20 条 |
| `GET /api/uploads/{id}?page=1` | 明细，每页 100 行 |
| `GET /api/uploads/{id}/csv` | 下载标准化 CSV |
| `GET /api/template` | 下载标准 Excel 模板 |

业务接口均要求登录；写请求需携带 `Origin: <BaseURL>` 和 `X-Requested-With: ADN`。
