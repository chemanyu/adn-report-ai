# ADN 结算 → 账户 → 项目 → 曝光：SQL 交接说明

给对接 agent。查询走 metadata-mcp 的 **HTTP execute_sql 接口**（一个 POST，不需要 MCP 协议），
接口细节见上一级目录 `execute-sql-http-api.md`。本文只讲：查哪个库、发什么 SQL、拿到什么、下一步用哪个字段。
下面所有 SQL 已于 2026-09-16 通过该接口跑通，`adn_sql_client.py` 里有同样的模板和一个 `locate()` 端到端函数。

## 1. 怎么调

```
POST http://172.16.3.25:8081/api/v1/execute_sql
Authorization: Bearer openid.<钉钉 openid>
Content-Type: application/json

{"database_id": "adt", "sql": "SELECT ...", "question": "可选，只落审计"}
```
成功 200，body 顶层就是 `{columns, rows:[{列名:值}], row_count, truncated, execution_ms}`。
失败看 HTTP 码加 `code` 或 `error` 字段（401 凭证、403 `permission_denied` 附 `denied_tables`、400 `sql_rejected` / `syntax_error` / `query_failed`）。

Python：`export MS_OPENID=xxx && python3 adn_sql_client.py adt "SELECT ..."`；代码里 `execute_sql(database_id, sql)` 返回 list[dict]。

## 2. 写 SQL 的硬规则（每条都踩过）

| 规则 | 违反时的返回 |
|---|---|
| 只能单条 SELECT | 400 `sql_rejected` |
| 所有库都过 PostgreSQL 解析器：**禁反引号**，保留字列写 `p.system`，日期加减写 `INTERVAL '7' DAY` | 400 `syntax_error` |
| `inhouse_ads` 表名必须全限定 `adn_report.settlement_rows` | 400 `query_failed: relation does not exist` |
| 不开放 `information_schema` / `SHOW` | 403 `permission_denied` |
| 结果上限 5000 行，没写 LIMIT 会自动补，`truncated=true` 表示被截 | 数据不完整，可能超的先 GROUP BY |
| Doris 日期写 `date = '2026-09-10'` 或半开区间；注释只能 `/* */` | 空结果 / 语句被截 |
| 字符串参数自己转义单引号 | 注入或语法错 |

## 3. 库、表、字段

| database_id | 表 | 一行是什么 | 用到的字段 |
|---|---|---|---|
| `inhouse_ads`（PG） | `adn_report.settlement_rows` | 运营上传的一条结算任务 | `settlement_date` `agency` `advertiser` `task_name` `settlement_count` `unit_price` `amount` `extra_fields` `upload_id` |
| `adt`（MySQL 5.6） | `account` | 一个账户 | `account_id` `account_name` `account_type`(1 代理商 / 2 广告主) `pid`(父账户) |
| `adn`（MySQL 8） | `adn_project` | 一个 ADN 投放项目 | `id` `name` `account_id`(广告主) `agent_id`(代理商) `status` `run_status` `del_flag` `deep_link` `create_time`(秒级 int) |
| `doris_data_center`（Doris） | `center_ssp_all` | 平台侧日汇总，ADN 流量 = `accept_id = 11243` | `date` `view` `click` `real_cost`(微元) `prime_cost` `dp_succ` |
| `inhouse_ads`（待建） | `adn_report.consume_rows` | 项目级日消耗 | 见 §4 第 ⑥ 步 |

## 4. 链路与 SQL

键的传递：`settlement_rows(agency, advertiser)` → `account(account_name + account_type + pid)` → `account_id`
→ `adn_project(account_id)` → `project_id` → 消耗表 `(项目ID, 日期)`。任务级可选：`task_name` 里的键 ↔ `adn_project.deep_link`。

### ① 名字 → 代理商ID、广告主ID　`database_id = "adt"`
广告主按 type=2 找，再 JOIN 它的 `pid` 要求父账户 type=1 且名字等于代理商。只按名字会撞（「知乎」4 个 ID、「优优」3 个）。
```sql
SELECT c.account_id AS advertiser_id, c.account_name AS advertiser,
       p.account_id AS agency_id,     p.account_name AS agency
FROM account c JOIN account p ON p.account_id = c.pid
WHERE c.account_name = '微博-汽水-1户' AND c.account_type = 2
  AND p.account_name = '微博汽水'     AND p.account_type = 1
```
返回 `[{"advertiser_id":1073772042,"advertiser":"微博-汽水-1户","agency_id":1073772041,"agency":"微博汽水"}]`。
必须恰好 1 行：0 行进人工映射，多行报错。批量解析把 `c.account_name IN ('a','b',...)` 一次查出，应用层再按 agency 过滤。

已知 3 组需要别名：
| 结算里的写法 | 库里实际 |
|---|---|
| 支付宝 / 短剧-短剧 | 代理商「支付宝-短剧」1073772093，广告主「短剧」1073772094 |
| 程投 / 程投-微博 | 广告主「微博-程投」1073772705 |
| 微博CPA / 知乎 | 代理商「微博-CPA」1073772655，广告主「知乎」1073772657 |

### ② 广告主ID → 项目列表　`database_id = "adn"`
```sql
SELECT p.id AS project_id, p.name AS project_name, p.account_id AS advertiser_id, p.agent_id AS agency_id,
       p.status, p.run_status, p.del_flag, FROM_UNIXTIME(p.create_time) AS create_time, p.deep_link
FROM adn_project p
WHERE p.account_id = 1073772042 AND p.del_flag = 0
ORDER BY p.id
```
一个广告主几个到几十个项目。项目名不唯一（「补单」「快手iOS」重复出现），不要按名字反查。

### ③ 任务键 → 项目　`database_id = "adn"`（可选，做任务级）
从 `task_name` 提取键：`qsyy_9999_0Xios`、`hpcp_9999_00Xios`、`ios_perform_screen_NNNNN`、`shuqi_dawenyu_NNN`、`th_NNNN`，或开头四位数字（淘宝闪购）。
**只匹配 `deep_link`，不要用 `target_url`**：target_url 统一回落到 08ios，会把所有项目串到一起。
```sql
SELECT p.id AS project_id, p.name AS project_name
FROM adn_project p
WHERE p.account_id = 1073772042
  AND (p.deep_link LIKE '%qsyy_9999_09ios%' OR p.name LIKE 'qsyy_9999_09ios-%')
```
返回 1 行 → 直接写 `project_id`；多行 → 一任务多项目，写映射表；0 行 → 停在账户级（汇川、飞猪的任务名没有键，属正常）。

### ④ 项目ID 列表 → 项目主数据　`database_id = "adn"`
```sql
SELECT p.id AS project_id, p.name AS project_name, p.account_id AS advertiser_id, p.agent_id AS agency_id, p.status, p.del_flag
FROM adn_project p WHERE p.id IN (50003301, 50000696)
```

### ⑤ 结算明细与看板粒度聚合　`database_id = "inhouse_ads"`
```sql
/* 明细 */
SELECT s.id, s.settlement_date::date AS settlement_date, s.agency, s.advertiser, s.task_name,
       s.settlement_count, s.unit_price, s.amount, s.extra_fields, s.upload_id
FROM adn_report.settlement_rows s
WHERE s.settlement_date >= '2026-08-23' AND s.settlement_date < '2026-08-24'
ORDER BY s.agency, s.advertiser, s.id;

/* 代理商-广告主-日期 聚合（app_37 粒度） */
SELECT s.agency, s.advertiser, s.settlement_date::date AS settlement_date,
       COUNT(*) AS task_rows, SUM(s.settlement_count) AS settlement_count, SUM(s.amount) AS amount
FROM adn_report.settlement_rows s
WHERE s.settlement_date >= '2026-08-01' AND s.settlement_date < '2026-09-01'
GROUP BY s.agency, s.advertiser, s.settlement_date::date
ORDER BY s.settlement_date, s.agency, s.advertiser;

/* 去重的 代理商-广告主，用于批量做 ① */
SELECT s.agency, s.advertiser, COUNT(*) AS n, MIN(s.settlement_date)::date AS d_from, MAX(s.settlement_date)::date AS d_to
FROM adn_report.settlement_rows s GROUP BY s.agency, s.advertiser;
```
金额以 `amount` 为准，不要用 count × price 推算（飞猪只有金额，淘宝闪购没有单价）。数值列返回的是字符串，自己转数。

### ⑥ 项目级 展示 / 点击 / CTR / 消耗 / 转化　`database_id = "inhouse_ads"`（**表待建**）
今天这份数据只在《adn消耗_最终版.xlsx》里（列：日期、代理商ID、代理商、广告主ID、广告主、项目ID、项目、展示、点击、消耗、转化、转化单价、CPC、CPM），要先落成
`adn_report.consume_rows(stat_date, agency_id, agency, advertiser_id, advertiser, project_id, project_name, impressions, clicks, spend, conversions)`，主键 `(stat_date, project_id)`。落库后：
```sql
SELECT c.stat_date, c.project_id, c.project_name,
       SUM(c.impressions) AS impressions, SUM(c.clicks) AS clicks, SUM(c.spend) AS spend, SUM(c.conversions) AS conversions
FROM adn_report.consume_rows c
WHERE c.stat_date >= '2026-08-23' AND c.stat_date < '2026-08-24' AND c.advertiser_id = 1073772042
GROUP BY c.stat_date, c.project_id, c.project_name
ORDER BY c.stat_date, c.project_id
```
账户级 = 去掉 `project_id` 分组；CTR = clicks / impressions；媒体成本 = spend × 0.95（app_37 口径）。

### ⑦ 平台整体日汇总，只做对账　`database_id = "doris_data_center"`
`center_ssp_all` 拆不到账户或项目（`dist_advid` 是广告位 ID），只用来核总量和拿真实媒体成本。
```sql
SELECT date, SUM(view) AS impressions, SUM(click) AS clicks,
       SUM(real_cost) / 1000000 AS real_cost_yuan, SUM(dp_succ) AS deeplink_succ
FROM center_ssp_all
WHERE date >= '2026-09-01' AND date < '2026-09-16' AND accept_id = 11243
GROUP BY date ORDER BY date
```
实测与消耗 Excel 逐日：曝光偏差 ≤0.3%，点击 ≤1%，`real_cost` ≈ 消耗 × 0.88。

## 5. 端到端示例（真实数据）

结算行 id=495：`2026-08-23 | 微博汽水 | 微博-汽水-1户 | 汽水音乐定向26年8月-qsyy_9999_09ios | 23037 × 0.17 = 3916.29`

| 步 | 库 | 输入 | 输出 |
|---|---|---|---|
| ① | adt | 微博汽水 / 微博-汽水-1户 | agency_id 1073772041，advertiser_id 1073772042 |
| ② | adn | account_id = 1073772042 | 23 个项目 |
| ③ | adn | 键 `qsyy_9999_09ios` | 唯一项目 50003301「视频流-09」 |
| ⑥ | 消耗表 | (50003301, 2026-08-23) | 展示 116,000，点击 31,220，CTR 26.9%，消耗 2,855.36，转化 20,160 |

`python3 adn_sql_client.py` 默认就跑这个例子。

## 6. 导入时把 ID 冗余进 settlement_rows（建议）

`settlement_rows` 加 `agency_id`、`advertiser_id`（必填，导入时按 ① 解析）、`project_id`（可空，③ 唯一命中才写）；
一任务多项目另建 `settlement_task_map(advertiser_id, task_key, project_id)`；解析失败的行进待处理表，运营指定一次后自动命中。
之后看板 SQL 全按 ID JOIN。
