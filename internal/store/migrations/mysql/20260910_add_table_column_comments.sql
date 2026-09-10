-- 为第一版已有表补充中文表注释和字段注释。
-- 连接到目标数据库后执行，保留原字段类型、默认值、主外键及索引，不修改业务数据。
-- 可重复执行。新建数据库直接使用 schema.sql，无需执行本文件。
SET NAMES utf8mb4;

ALTER TABLE `users`
 COMMENT = '登录用户表：记录钉钉用户或本地管理员身份',
 MODIFY COLUMN id BIGINT NOT NULL AUTO_INCREMENT COMMENT '用户主键，自增 ID',
 MODIFY COLUMN identity_key VARCHAR(191) COLLATE utf8mb4_bin NOT NULL COMMENT '唯一登录身份：钉钉为 ding:unionId，本地管理员为 local:用户名，区分大小写',
 MODIFY COLUMN name VARCHAR(191) NOT NULL COMMENT '当前用户姓名：钉钉昵称或本地管理员名称，登录时更新',
 MODIFY COLUMN created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '首次登录创建用户的时间，UTC',
 MODIFY COLUMN last_login_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '最近一次成功登录时间，UTC';

ALTER TABLE `sessions`
 COMMENT = '登录会话表：保存会话令牌哈希及有效期，退出登录时删除',
 MODIFY COLUMN token_hash CHAR(64) CHARACTER SET ascii NOT NULL COMMENT '浏览器登录会话令牌的 SHA-256 哈希，64 位十六进制，不保存原始令牌',
 MODIFY COLUMN user_id BIGINT NOT NULL COMMENT '登录用户 ID，关联 users.id，用于确定访问权限',
 MODIFY COLUMN expires_at DATETIME NOT NULL COMMENT '会话到期时间，UTC，登录后有效期为 7 天';

ALTER TABLE `oauth_states`
 COMMENT = '钉钉 OAuth 授权状态表：校验授权回调，状态只能使用一次',
 MODIFY COLUMN token_hash CHAR(64) CHARACTER SET ascii NOT NULL COMMENT 'OAuth state 随机值的 SHA-256 哈希，同时与发起授权的浏览器 Cookie 校验',
 MODIFY COLUMN expires_at DATETIME NOT NULL COMMENT '授权状态到期时间，UTC，创建后有效期为 10 分钟';

ALTER TABLE `adn_accounts`
 COMMENT = 'ADN 账户目录表：记录结算文件关联的业务账户，供登录成员选择',
 MODIFY COLUMN id BIGINT NOT NULL AUTO_INCREMENT COMMENT 'ADN 账户主键，自增 ID',
 MODIFY COLUMN name VARCHAR(191) NOT NULL COMMENT 'ADN 账户显示名称，由创建人员填写',
 MODIFY COLUMN account_code VARCHAR(191) COLLATE utf8mb4_bin NOT NULL COMMENT '实际 ADN 账号或账户 ID，全平台唯一且区分大小写',
 MODIFY COLUMN created_by BIGINT NOT NULL COMMENT '创建该账户的登录用户 ID，关联 users.id',
 MODIFY COLUMN created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '账户创建时间，UTC';

ALTER TABLE `uploads`
 COMMENT = '结算上传批次表：每条记录对应一次成功导入的 Excel 工作表及其 CSV',
 MODIFY COLUMN id BIGINT NOT NULL AUTO_INCREMENT COMMENT '上传批次主键，自增 ID',
 MODIFY COLUMN user_id BIGINT NOT NULL COMMENT '实际上传用户 ID，关联 users.id，普通用户的数据访问权限以此字段为准',
 MODIFY COLUMN uploader_name VARCHAR(191) NOT NULL COMMENT '上传时的登录用户姓名快照，不随用户后续改名而变化',
 MODIFY COLUMN operator_name VARCHAR(191) NOT NULL COMMENT '本次结算的运营负责人姓名，默认登录姓名，可手工修改，不影响访问权限',
 MODIFY COLUMN account_id BIGINT NOT NULL COMMENT '本次结算关联的 ADN 账户 ID，关联 adn_accounts.id',
 MODIFY COLUMN original_name VARCHAR(255) NOT NULL COMMENT '上传的原始 Excel 文件名，含扩展名，不含本地目录',
 MODIFY COLUMN sheet_name VARCHAR(191) NOT NULL COMMENT '本次导入的 Excel 工作表名称，一次仅导入一个工作表',
 MODIFY COLUMN sha256 CHAR(64) CHARACTER SET ascii NOT NULL COMMENT '整个原始 XLSX 文件的 SHA-256 哈希，结合用户、账户和工作表防止重复上传',
 MODIFY COLUMN csv_path VARCHAR(255) NOT NULL COMMENT '归档 CSV 的随机文件名，相对于配置 DataDir 下的 csv 目录',
 MODIFY COLUMN columns_json JSON NOT NULL COMMENT '原始表头的 JSON 字符串数组，按 Excel 列顺序保存，用于动态展示和 CSV 输出',
 MODIFY COLUMN row_count INT NOT NULL COMMENT '成功入库的有效数据行数，不含表头及空行',
 MODIFY COLUMN total_amount DECIMAL(30,6) NOT NULL COMMENT '本批次所有明细结算金额之和，单位元，精确到 6 位小数',
 MODIFY COLUMN date_from DATE NOT NULL COMMENT '本批次明细中最早的结算日期，业务日期，不作时区转换',
 MODIFY COLUMN date_to DATE NOT NULL COMMENT '本批次明细中最晚的结算日期，业务日期，不作时区转换',
 MODIFY COLUMN created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '上传批次记录创建时间，UTC';

ALTER TABLE `settlement_rows`
 COMMENT = '结算明细表：固定结算字段加动态扩展字段，每条对应 Excel 的一个有效数据行',
 MODIFY COLUMN id BIGINT NOT NULL AUTO_INCREMENT COMMENT '结算明细主键，自增 ID',
 MODIFY COLUMN upload_id BIGINT NOT NULL COMMENT '所属上传批次 ID，关联 uploads.id，通过批次关联上传人和 ADN 账户',
 MODIFY COLUMN source_row INT NOT NULL COMMENT '原始 Excel 行号，从 1 开始，表头为第 1 行，保留跳过空行后的实际行号',
 MODIFY COLUMN settlement_date DATE NOT NULL COMMENT '标准列「日期」：结算业务日期，统一为 YYYY-MM-DD，不作时区转换',
 MODIFY COLUMN settlement_count DECIMAL(24,6) NOT NULL COMMENT '标准列「结算数」：结算数量，最多 18 位整数及 6 位小数，允许负数调整',
 MODIFY COLUMN unit_price DECIMAL(24,6) NOT NULL COMMENT '标准列「结算单价」：单笔结算价格，单位元，最多 18 位整数及 6 位小数',
 MODIFY COLUMN amount DECIMAL(24,6) NOT NULL COMMENT '标准列「结算金额」：该行结算金额，单位元，最多 18 位整数及 6 位小数，不自动按数量乘单价推算',
 MODIFY COLUMN extra_fields JSON NOT NULL COMMENT '四个标准列以外的 JSON 对象，键为原列名、值为原始单元格字符串，如渠道名称、广告位 ID',
 MODIFY COLUMN source_values JSON NOT NULL COMMENT '整行原始单元格值的 JSON 字符串数组，与 uploads.columns_json 按下标对应，保留标准化前的值';
