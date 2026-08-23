# task186-namemerge 植物学名异名归并证据服务

面向植物标本馆分类学家的**命名法规归并证据服务**：登记历史学名、发表证据与模式标本，
按《国际植物命名法规》(ICN) 的优先权、合法性、同名冲突与拼写变体规则计算名称关系，
构造"同物异名簇"，在指定分类观点下输出**当前接受名及法规证据**，并把确认后的清单发布为
不可变快照供馆藏系统消费；观点修订只发布新清单，旧清单仍可查询其原有归并证据。

## 标准命令

```bash
# 构建 / 静态检查 / 测试（固定环境）
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...

# 冒烟自检：真实创建名称/证据/关系/观点/清单，关闭并重开数据库验证重启恢复
go run ./cmd/namemerge --smoke-test

# 启动 HTTP 服务
go run ./cmd/namemerge --addr :8080 --db ./namemerge.db
```

## 业务闭环

1. 登记名称（学名、作者、发表年份区间）、发表证据（指纹幂等去重）、模式标本（指纹幂等去重）；
2. 绑定名称 → 发表 → 模式（同一名称+发表重复绑定可"补齐模式"）；
3. 提议并证明名称关系（同型异名须共享模式标本；组合变更须共享种加词且属名不同；
   归并边成环被拒绝）；
4. 创建法规规则版本与分类观点，求值产生名称簇：优先权最早且具备模式者 → **接受名**，
   其余簇成员 → **异名**；缺模式 / 日期不可排序 / 同一模式指向多个接受名 → **冲突待裁决**；
5. 裁决冲突（接受 / 驳回 / 暂缓）后发布**清单快照**（绑定规则版本、指纹冻结、不可编辑）；
6. 修订产生新观点并发布新清单，旧观点与新清单可做差异对比。

## API 一览（前缀 /api，全部 JSON）

| 能力 | 入口 |
| --- | --- |
| 登记/列表/详情/更新名称 | `POST/GET /api/names`、`GET/PUT /api/names/{id}` |
| 登记/列表发表证据 | `POST/GET /api/publications` |
| 登记/列表模式标本 | `POST/GET /api/specimens` |
| 绑定证据、证据详情 | `POST/GET /api/names/{id}/evidence` |
| 提议/列表/证明关系 | `POST/GET /api/relations`、`POST /api/relations/{id}/prove` |
| 规则版本 | `POST/GET /api/rules`、`GET /api/rules/current` |
| 观点创建/列表/求值/簇/冲突 | `POST/GET /api/views`、`POST /api/views/{id}/evaluate`、`GET /api/views/{id}/clusters`、`GET /api/views/{id}/conflicts` |
| 裁决 | `POST /api/views/{id}/rulings` |
| 发布清单、查询清单 | `POST /api/views/{id}/publish`、`GET /api/views/{id}/checklist` |
| 清单列表/差异 | `GET /api/checklists`、`GET /api/checklists/{id}/diff?vs=<id>` |
| 统计 / 自检 | `GET /api/stats`、`GET /api/health` |

## 状态机

- 名称：`pending_review → legitimate / illegitimate / synonym_candidate / accepted`
- 发表证据：`pending_check → valid / date_conflict / missing_type`
- 名称关系：`proposed → proven / conflicting / rejected`
- 分类观点：`draft → publishable / pending_ruling → published → superseded`

## 持久化与重启恢复

SQLite（`modernc.org/sqlite` 纯 Go 驱动，CGO 无关）持久化全部实体：
`names`、`publications`、`specimens`、`name_links`、`relations`、
`rule_versions`、`views`、`rulings`、`checklists`、`checklist_items`。
`--smoke-test` 关闭并重开同一数据库，验证名称、关系、清单快照与指纹完整恢复；
发布清单绑定规则版本，重启后从未判定关系继续归并。
