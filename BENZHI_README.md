# task186-namemerge 评测说明

植物学名异名归并证据服务：按命名法规（优先权/合法性/同名冲突/拼写变体）归并学名，
构造同物异名簇，在分类观点下输出接受名及法规证据，发布不可变清单快照。

## 构建与运行

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/namemerge --smoke-test          # 冒烟自检（含重启恢复验证）
go run ./cmd/namemerge --addr :8080 --db ./namemerge.db
```

## --smoke-test 契约（Docker 双架构唯一判据）

1. 登记 3 个名称、3 份发表证据、1 个共享模式标本；
2. A(1753) 与 B(1790) 共享模式 → 提议并证明同型异名关系；C 缺模式；
3. 求值：A=accepted、B=synonym、C=deferred（missing_type 冲突）；
4. 发布被 pending_ruling 拒绝 → 补齐 C 的模式 → 重新求值 → 发布成功（3 条目）；
5. 关闭数据库，重开同一文件：3 名称、清单指纹一致 → 退出码 0。

## Docker 双架构

```bash
bash build_benzhi_docker.sh <镜像名> linux/amd64
bash build_benzhi_docker.sh <镜像名> linux/arm64
docker run --rm <镜像名> --smoke-test        # 仅传 flag，不追加路径参数
```

## API（前缀 /api）

`POST/GET /api/names`、`GET/PUT /api/names/{id}`、`POST/GET /api/publications`、
`POST/GET /api/specimens`、`POST/GET /api/names/{id}/evidence`、
`POST/GET /api/relations`、`POST /api/relations/{id}/prove`、
`POST/GET /api/rules`、`GET /api/rules/current`、`POST/GET /api/views`、
`POST /api/views/{id}/evaluate`、`GET /api/views/{id}/clusters`、
`GET /api/views/{id}/conflicts`、`POST /api/views/{id}/rulings`、
`POST /api/views/{id}/publish`、`GET /api/views/{id}/checklist`、
`GET /api/checklists`、`GET /api/checklists/{id}/diff?vs=<id>`、
`GET /api/stats`、`GET /api/health`。

## 组件版本

- Go 1.26.3（`GOTOOLCHAIN=local`）
- SQLite 3.46.1（`modernc.org/sqlite` v1.52.0，纯 Go 驱动）
- 见 `component-versions.json`
