# Grafana Makefile 分析文档

## 概述

Grafana 的 Makefile 是一个自文档化的构建系统，支持后端（Go）和前端（TypeScript/React）的完整开发工作流。运行 `make help` 可查看所有可用目标。

---

## 核心变量

### 构建版本与元信息

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `BUILD_NUMBER` | 构建编号 | `local` |
| `BUILD_VERSION` | 构建版本号（从 `package.json` 提取） | 自动提取 |
| `BUILD_COMMIT` | Git commit SHA | `git rev-parse --short HEAD` |
| `BUILD_BRANCH` | Git 分支名 | `git rev-parse --abbrev-ref HEAD` |
| `BUILD_STAMP` | 构建时间戳 | `date +%s` 或 `SOURCE_DATE_EPOCH` |

### Go 构建配置

| 变量 | 说明 |
|------|------|
| `GO` | Go 命令，默认 `go` |
| `GO_VERSION` | 要求的 Go 版本，`1.26.4` |
| `GO_RACE` | 是否启用竞态检测（存在 `.go-race-enabled-locally` 文件或设置环境变量时启用） |
| `GO_BUILD_TAGS` | Go 构建标签（如 `oss`、`enterprise`、`pro`） |
| `GO_BUILD_DEV` | 开发模式构建标志（值为 `1` 或 `dev` 时启用） |
| `GO_LDFLAGS` | 链接器标志，注入版本/commit/分支/时间戳信息 |

### 平台与架构

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `OS` | 目标操作系统 | `GOOS` 或 `go env GOOS` |
| `ARCH` | 目标架构 | `GOARCH` 或 `go env GOARCH` |
| `ARM` | ARM 版本（如 `6`、`7`） | `GOARM` |

### 测试分片

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `SHARD` | 当前分片编号 | `1` |
| `SHARDS` | 总分片数 | `1` |

### 打包变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `TARGZ_PACKAGE_NAME` | tar.gz 包名 | `grafana` |
| `FPM_LICENSE` | 打包许可证 | `AGPLv3` |
| `SLIM` | 是否构建精简 Docker 镜像 | `false` |
| `DOCKER_TAG` | Docker 镜像标签 | `grafana:版本号` |

### 工具链（由 `.citools/Variables.mk` 定义）

| 工具 | 来源 |
|------|------|
| `air` | 热重载工具 `github.com/air-verse/air` |
| `cue` | CUE 语言工具 `cuelang.org/go/cmd/cue` |
| `golangci-lint` | Go 代码检查 `github.com/golangci/golangci-lint/v2` |
| `govulncheck` | Go 漏洞检查 `golang.org/x/vuln/cmd/govulncheck` |
| `lefthook` | Git 钩子管理 `github.com/evilmartians/lefthook` |
| `swagger` | Swagger 生成 `github.com/go-swagger/go-swagger/cmd/swagger` |

---

## 目标分类详解

### 1. 依赖安装（Dependencies）

| 目标 | 说明 |
|------|------|
| `deps-go` | 安装后端 Go 依赖（`go mod download`） |
| `deps-js` | 安装前端 JS 依赖（`yarn install --immutable`） |
| `deps` | 安装所有依赖（当前仅包含 `deps-js`） |
| `node_modules` | 根据 `package.json` 和 `yarn.lock` 安装 node 模块 |

### 2. Swagger / API 规范

| 目标 | 说明 |
|------|------|
| `swagger-oss-gen` | 生成 OSS 版 Swagger API 规范 → `public/api-spec.json` |
| `swagger-enterprise-gen` | 生成 Enterprise 版 Swagger 规范（需企业版启用） |
| `swagger-gen` | 完整 Swagger 生成流程：`gen-go` + 合并规范 + 验证 |
| `swagger-validate` | 验证合并后的 API 规范 |
| `swagger-clean` | 清理生成的 Swagger 文件 |

**关键文件：**
- OSS 规范：`public/api-spec.json`
- Enterprise 规范：`public/api-enterprise-spec.json`
- 合并规范：`public/api-merged.json`
- NgAlert 规范：`pkg/services/ngalert/api/tooling/api.json`

### 3. OpenAPI 3

| 目标 | 说明 |
|------|------|
| `openapi3-gen` | 从 Swagger 2 规范生成 OpenAPI 3 规范 → `public/openapi3.json` |
| `generate-openapi` | 完整 OpenAPI 生成：生成规范 + 运行 API 测试 + 处理前端类型 |

### 4. 国际化（Internationalisation）

| 目标 | 说明 |
|------|------|
| `i18n-extract` | 提取所有 i18n 字符串（OSS + Enterprise + packages + plugins） |
| `i18n-extract-enterprise` | 提取 Enterprise 前端 i18n 字符串（需企业版启用） |

### 5. 代码生成（Building / Code Generation）

| 目标 | 说明 |
|------|------|
| `gen-cue` | 从 `.cue` 文件生成 Go/TS 代码，同步 dashboard kind CUE 文件 |
| `gen-apps` | 生成 Grafana App SDK 应用代码 + gofmt + fix-cue。可用 `app=<name>` 指定单个应用 |
| `gen-feature-toggles` | 生成功能开关代码（`pkg/services/featuremgmt/`） |
| `gen-go` | 生成 Wire 依赖注入图（OSS 版） |
| `gen-enterprise-go` | 生成 Wire 依赖注入图（Enterprise 版，需企业版启用） |
| `gen-app-manifests-unistore` | 生成统一存储应用清单 |
| `gen-jsonnet` | 生成 Jsonnet 配置 |
| `gen-themes` | 生成主题相关代码 |
| `gen-ts` | 从 Go 结构体生成 TypeScript 定义 |
| `fix-cue` | 格式化和修复 CUE 文件。可用 `app=<name>` 指定单个应用 |
| `update-workspace` | 更新 Go workspace（运行 `gen-go` + `generate-enterprise-imports` + 更新脚本） |
| `generate-enterprise-imports` | 生成 Enterprise 导入文件（需企业版启用） |

### 6. 构建（Building）

| 目标 | 说明 |
|------|------|
| `build-go` | 编译后端，输出到 `bin/<OS>/<ARCH>/grafana` |
| `build-backend` | 构建后端（别名 `build-go`） |
| `build-air` | 构建后端并复制为 `bin/grafana-air`（用于 air 热重载） |
| `build-js` | 构建前端资源（`yarn run build`） |
| `build` | 构建后端 + 前端 |
| `build-plugin-go` | 构建解耦插件，需设置 `PLUGIN_ID` |

**开发模式构建：** 设置 `GO_BUILD_DEV=1` 启用开发模式，会：
- 添加 `-gcflags "all=-N -l"`（禁用优化，便于调试）
- 不使用 `-trimpath`（保留调试路径信息）

### 7. 打包（Packaging）

| 目标 | 说明 |
|------|------|
| `build-catalog-plugins-data` | 下载默认目录插件到 `data/plugins-bundled` |
| `build-targz` | 构建 tar.gz 分发包 |
| `build-deb` | 构建 .deb 包（需要 fpm） |
| `build-rpm` | 构建 .rpm 包（需要 fpm） |
| `build-msi` | 构建 Windows MSI 安装包（需要 Docker + Wine） |

**打包输出路径：** `dist/grafana_<版本>_<编号>_<OS>_<架构>.{tar.gz,deb,rpm,msi}`

### 8. 运行（Run）

| 目标 | 说明 |
|------|------|
| `run` | 构建并运行后端，监听文件变化自动重载（使用 air） |
| `run-go` | 直接运行后端（带竞态检测和 profiling） |
| `run-frontend` | 安装前端依赖并启动开发服务器（`yarn start`） |
| `frontend-service` | 运行前端服务（使用 devenv 脚本） |

### 9. 测试（Testing）

| 目标 | 说明 |
|------|------|
| `test-go` | 运行所有后端测试（单元 + 集成） |
| `test-go-unit` | 运行后端单元测试（`-short` 标志，30 分钟超时） |
| `test-go-unit-pretty` | 使用 tparse 美化输出单元测试结果，需设置 `FILES` 变量 |
| `test-go-integration` | 运行后端集成测试（匹配 `TestIntegration*`） |
| `test-go-integration-postgres` | PostgreSQL 后端集成测试（需先启动 devenv） |
| `test-go-integration-mysql` | MySQL 后端集成测试（需先启动 devenv） |
| `test-go-integration-redis` | Redis 缓存集成测试 |
| `test-go-integration-memcached` | Memcached 缓存集成测试 |
| `test-go-integration-alertmanager` | 远程 Alertmanager 集成测试 |
| `test-go-integration-grafana-alertmanager` | Grafana Alertmanager 集成测试 |
| `test-js` | 运行前端测试（`yarn test`） |
| `test` | 运行所有测试（后端 + 前端） |

**测试分片示例：**
```bash
# 运行第 1 片（共 4 片）
make test-go-unit SHARD=1 SHARDS=4
```

### 10. 代码检查（Linting）

| 目标 | 说明 |
|------|------|
| `golangci-lint` | 使用 golangci-lint 检查 Go 代码 |
| `lint-go` | 运行所有后端代码检查（别名 `golangci-lint`） |
| `lint-go-diff` | 仅检查相对于 `main` 分支变更的 Go 文件 |
| `gofmt` | 格式化所有 Go 文件 |
| `shellcheck` | 检查 shell 脚本（使用 Docker 运行 koalaman/shellcheck） |

**自定义检查范围：**
```bash
# 指定检查的 Go 文件
make lint-go GO_LINT_FILES="./pkg/services/myservice/..."
```

### 11. Docker 镜像构建

| 目标 | 说明 |
|------|------|
| `build-docker` | 从 tar.gz 构建 Alpine Docker 镜像 |
| `build-docker-ubuntu` | 从 tar.gz 构建 Ubuntu Docker 镜像 |
| `build-docker-distroless` | 从 tar.gz 构建 Distroless Docker 镜像 |
| `build-docker-full` | 完整构建 Alpine Docker 镜像（开发用） |
| `build-docker-full-ubuntu` | 完整构建 Ubuntu Docker 镜像（开发用） |
| `build-docker-full-distroless` | 完整构建 Distroless Docker 镜像（开发用） |

**Docker 构建模式：**
- 默认生产模式：`NODE_ENV=production`，`yarn build`
- 开发模式：设置 `GO_BUILD_DEV=1` 或 `NODE_ENV=dev`，使用 `yarn dev`

**精简镜像：** 设置 `SLIM=true` 构建精简版 Docker 镜像

### 12. 开发环境服务（Services）

| 目标 | 说明 |
|------|------|
| `devenv` | 启动开发环境服务（需指定 `sources` 参数） |
| `devenv-down` | 停止开发环境服务 |
| `devenv-postgres` | PostgreSQL 测试环境 |
| `devenv-mysql` | MySQL 测试环境 |

**使用示例：**
```bash
# 启动 PostgreSQL 和 LDAP
make devenv sources=postgres,auth/openldap

# 启动 PostgreSQL 测试环境
make devenv sources=postgres_tests
```

### 13. 辅助工具（Helpers）

| 目标 | 说明 |
|------|------|
| `protobuf` | 编译 protobuf 定义（使用 buf 工具） |
| `clean` | 清理构建产物（`node_modules`、`public/build`） |
| `lefthook-install` | 安装 Git pre-commit 钩子 |
| `lefthook-uninstall` | 卸载 Git 钩子 |
| `enable-go-race` | 启用本地 Go 竞态检测（创建 `.go-race-enabled-locally` 文件） |
| `go-race-is-enabled` | 检查竞态检测是否启用 |
| `check-licenses` | 检查依赖许可证合规性 |
| `.policy.yml` | 生成 policy-bot 配置文件 |
| `help` | 显示帮助信息 |

---

## 关键构建流程

### 完整构建流程

```
make all  →  deps  →  build
                ↓           ↓
           deps-js    build-go + build-js
```

### 后端构建链

```
build-go
  ├── pkg/services/preference/themes_generated.go (gen-themes)
  └── go build (GO_BUILD_ENV + GO_BUILD_ARGS)
       ├── -buildvcs=false
       ├── -trimpath (生产模式)
       ├── -race (竞态检测启用时)
       ├── -tags (构建标签)
       ├── -gcflags (开发模式: all=-N -l)
       ├── -ldflags (版本信息注入)
       └── 输出: bin/<OS>/<ARCH>/grafana
```

### Swagger 生成链

```
swagger-gen
  ├── gen-go (Wire 依赖注入)
  ├── swagger-oss-gen → public/api-spec.json
  ├── swagger-enterprise-gen → public/api-enterprise-spec.json (可选)
  ├── ngalert spec → pkg/services/ngalert/api/tooling/api.json
  ├── 合并 → public/api-merged.json
  └── swagger-validate
```

### 打包构建链

```
build-targz
  ├── data/plugins-bundled (下载目录插件)
  ├── bin/<OS>/<ARCH>/grafana (后端二进制)
  └── public/build (前端资源)

build-deb / build-rpm → 依赖 build-targz
build-docker / build-docker-ubuntu / build-docker-distroless → 依赖 build-targz
build-msi → 依赖 build-targz
```

---

## 常用开发命令速查

| 场景 | 命令 |
|------|------|
| 启动后端（热重载） | `make run` |
| 启动前端开发服务器 | `make run-frontend` |
| 构建后端 | `make build-backend` |
| 构建前端 | `make build-js` |
| 运行后端单元测试 | `make test-go-unit` |
| 运行指定包的单元测试 | `go test -run TestName ./pkg/services/myservice/` |
| 运行前端测试 | `yarn test path/to/file` |
| Go 代码检查 | `make lint-go` |
| 生成 Wire DI | `make gen-go` |
| 生成 CUE 代码 | `make gen-cue` |
| 生成功能开关 | `make gen-feature-toggles` |
| 启动 PostgreSQL 开发环境 | `make devenv sources=postgres` |
| 启动 PostgreSQL 测试环境 | `make devenv sources=postgres_tests` |
| 编译 protobuf | `make protobuf` |
| 清理构建产物 | `make clean` |

---

## 注意事项

1. **Wire DI**：修改后端服务初始化后必须运行 `make gen-go` 重新生成
2. **CUE schemas**：修改 `kinds/` 下的 CUE 文件后需运行 `make gen-cue`
3. **Feature toggles**：修改 `pkg/services/featuremgmt/` 后需运行 `make gen-feature-toggles`
4. **Go workspace**：添加 Go 模块后需运行 `make update-workspace`
5. **Enterprise 条件编译**：多个目标通过检测 `pkg/extensions/ext.go` 是否存在来判断企业版是否启用
6. **竞态检测**：运行 `make enable-go-race` 启用本地竞态检测，之后所有 Go 测试和构建都会带上 `-race` 标志
7. **测试分片**：CI 环境使用 `SHARD`/`SHARDS` 变量并行化测试，本地默认不分片
8. **构建标签**：默认 `oss`，企业版使用 `enterprise`，专业版使用 `pro`
