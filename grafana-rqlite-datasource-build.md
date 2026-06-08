# grafana-rqlite-datasource 插件编译打包文档

本文档介绍如何为 grafana-rqlite-datasource 插件编译和打包生产环境的前后端代码。

## 插件信息

- **插件 ID**: `grafana-rqlite-datasource`
- **可执行文件名**: `gpx_grafana-rqlite-datasource`
- **类型**: 数据源插件（datasource）
- **前端目录**: [public/app/plugins/datasource/grafana-rqlite-datasource/](file:///Users/yangqian/go/src/github.com/lynnyq/grafana/public/app/plugins/datasource/grafana-rqlite-datasource)
- **后端目录**: [pkg/tsdb/grafana-rqlite-datasource/](file:///Users/yangqian/go/src/github.com/lynnyq/grafana/pkg/tsdb/grafana-rqlite-datasource)

## 前置条件

### 环境要求

- Node.js >= 22 < 25（参考项目根目录 [.nvmrc](file:///Users/yangqian/go/src/github.com/lynnyq/grafana/.nvmrc)）
- Go 1.25+（参考项目根目录 [go.mod](file:///Users/yangqian/go/src/github.com/lynnyq/grafana/go.mod)）
- Yarn 4.11.0+（项目使用 Yarn 作为包管理器）
- Mage（用于 Go 构建）

### 安装依赖

```bash
# 在项目根目录执行
yarn install --immutable
```

## 编译打包步骤

### 方式一：完整构建（推荐）

#### 1. 构建后端

```bash
# 在项目根目录执行
make build-plugin-go PLUGIN_ID=grafana-rqlite-datasource
```

此命令会：
- 编译 Go 后端代码
- 输出二进制文件到前端插件的 `dist/` 目录

#### 2. 构建前端

```bash
# 方式 A：使用项目根目录的 yarn 脚本
yarn plugin:build

# 方式 B：仅构建 rqlite 插件（使用 nx）
nx run grafana-rqlite-datasource:build

# 方式 C：在插件目录直接构建
cd public/app/plugins/datasource/grafana-rqlite-datasource
yarn build
```

### 方式二：分步构建

#### 后端构建详解

后端构建使用 Makefile 中的 `build-plugin-go` 目标，该目标会：
1. 调用 [pkg/tsdb/Magefile.go](file:///Users/yangqian/go/src/github.com/lynnyq/grafana/pkg/tsdb/Magefile.go) 中的构建脚本
2. 使用 `grafana-plugin-sdk-go/build` 包进行构建
3. 将编译后的二进制文件输出到 `public/app/plugins/datasource/grafana-rqlite-datasource/dist/`

#### 前端构建详解

前端构建使用 webpack，配置文件为 [webpack.config.ts](file:///Users/yangqian/go/src/github.com/lynnyq/grafana/public/app/plugins/datasource/grafana-rqlite-datasource/webpack.config.ts)，它继承自 `@grafana/plugin-configs/webpack.config.ts`。

前端脚本说明（在 [package.json](file:///Users/yangqian/go/src/github.com/lynnyq/grafana/public/app/plugins/datasource/grafana-rqlite-datasource/package.json) 中定义）：
- `yarn build`: 生产环境构建
- `yarn build:commit`: 生产环境构建，并包含 git commit 哈希
- `yarn dev`: 开发模式构建，支持热重载

## 构建输出

构建完成后，插件文件将位于：

```
public/app/plugins/datasource/grafana-rqlite-datasource/
├── dist/
│   ├── module.js                    # 前端编译后的主文件
│   ├── module.js.map                # 前端 source map
│   ├── gpx_grafana-rqlite-datasource_darwin_amd64  # macOS 后端二进制
│   ├── gpx_grafana-rqlite-datasource_linux_amd64   # Linux 后端二进制
│   ├── gpx_grafana-rqlite-datasource_windows_amd64.exe  # Windows 后端二进制
│   └── ...                          # 其他平台的二进制文件
├── plugin.json                      # 插件配置文件
└── img/                             # 插件图片资源
```

## 生成 tar 包

构建完成后，您可以将插件打包成 tar 归档文件，方便分发和部署。

### 1. 打包完整插件（包含所有平台）

```bash
# 进入插件目录
cd public/app/plugins/datasource/grafana-rqlite-datasource

# 创建 tar 包
tar -czf grafana-rqlite-datasource-$(date +%Y%m%d).tar.gz \
  --exclude="*.md" \
  --exclude="*.ts" \
  --exclude="*.tsx" \
  --exclude="*.go" \
  --exclude="node_modules" \
  --exclude="*.test.*" \
  --exclude="jest*" \
  --exclude="tsconfig.json" \
  --exclude="webpack.config.ts" \
  --exclude="project.json" \
  --exclude=".eslint*" \
  plugin.json img/ dist/
```

### 2. 打包特定平台的插件

如果您只想打包特定平台的版本，可以使用以下命令：

```bash
# 进入插件目录
cd public/app/plugins/datasource/grafana-rqlite-datasource

# 定义版本号（可选）
PLUGIN_VERSION=$(grep -Eo '"version": "[^"]+"' package.json | cut -d'"' -f4)
if [ -z "$PLUGIN_VERSION" ]; then
  PLUGIN_VERSION="1.0.0"
fi

# 打包 Linux 版本
tar -czf grafana-rqlite-datasource-linux-${PLUGIN_VERSION}.tar.gz \
  --exclude="gpx_grafana-rqlite-datasource_darwin*" \
  --exclude="gpx_grafana-rqlite-datasource_windows*" \
  --exclude="*.md" \
  --exclude="*.ts" \
  --exclude="*.tsx" \
  --exclude="*.go" \
  --exclude="node_modules" \
  --exclude="*.test.*" \
  --exclude="jest*" \
  --exclude="tsconfig.json" \
  --exclude="webpack.config.ts" \
  --exclude="project.json" \
  --exclude=".eslint*" \
  plugin.json img/ dist/

# 打包 macOS 版本
tar -czf grafana-rqlite-datasource-darwin-${PLUGIN_VERSION}.tar.gz \
  --exclude="gpx_grafana-rqlite-datasource_linux*" \
  --exclude="gpx_grafana-rqlite-datasource_windows*" \
  --exclude="*.md" \
  --exclude="*.ts" \
  --exclude="*.tsx" \
  --exclude="*.go" \
  --exclude="node_modules" \
  --exclude="*.test.*" \
  --exclude="jest*" \
  --exclude="tsconfig.json" \
  --exclude="webpack.config.ts" \
  --exclude="project.json" \
  --exclude=".eslint*" \
  plugin.json img/ dist/

# 打包 Windows 版本
tar -czf grafana-rqlite-datasource-windows-${PLUGIN_VERSION}.tar.gz \
  --exclude="gpx_grafana-rqlite-datasource_linux*" \
  --exclude="gpx_grafana-rqlite-datasource_darwin*" \
  --exclude="*.md" \
  --exclude="*.ts" \
  --exclude="*.tsx" \
  --exclude="*.go" \
  --exclude="node_modules" \
  --exclude="*.test.*" \
  --exclude="jest*" \
  --exclude="tsconfig.json" \
  --exclude="webpack.config.ts" \
  --exclude="project.json" \
  --exclude=".eslint*" \
  plugin.json img/ dist/
```

### 3. 使用脚本一键打包

您也可以创建一个简单的打包脚本，在项目根目录创建 `package-rqlite-plugin.sh`：

```bash
#!/bin/bash

# 脚本位置：项目根目录

set -e

# 插件信息
PLUGIN_ID="grafana-rqlite-datasource"
PLUGIN_DIR="public/app/plugins/datasource/${PLUGIN_ID}"
OUTPUT_DIR="dist/plugins"

# 获取版本号
PLUGIN_VERSION=$(grep -Eo '"version": "[^"]+"' "${PLUGIN_DIR}/package.json" | cut -d'"' -f4)
if [ -z "$PLUGIN_VERSION" ]; then
  PLUGIN_VERSION="1.0.0"
fi

# 创建输出目录
mkdir -p "${OUTPUT_DIR}"

echo "开始打包 ${PLUGIN_ID} v${PLUGIN_VERSION}..."

# 进入插件目录
cd "${PLUGIN_DIR}"

# 打包所有平台
echo "打包所有平台版本..."
tar -czf "../../../../${OUTPUT_DIR}/${PLUGIN_ID}-${PLUGIN_VERSION}-all.tar.gz" \
  --exclude="*.md" \
  --exclude="*.ts" \
  --exclude="*.tsx" \
  --exclude="*.go" \
  --exclude="node_modules" \
  --exclude="*.test.*" \
  --exclude="jest*" \
  --exclude="tsconfig.json" \
  --exclude="webpack.config.ts" \
  --exclude="project.json" \
  --exclude=".eslint*" \
  plugin.json img/ dist/

# 打包 Linux 版本
echo "打包 Linux 版本..."
tar -czf "../../../../${OUTPUT_DIR}/${PLUGIN_ID}-${PLUGIN_VERSION}-linux.tar.gz" \
  --exclude="gpx_${PLUGIN_ID}_darwin*" \
  --exclude="gpx_${PLUGIN_ID}_windows*" \
  --exclude="*.md" \
  --exclude="*.ts" \
  --exclude="*.tsx" \
  --exclude="*.go" \
  --exclude="node_modules" \
  --exclude="*.test.*" \
  --exclude="jest*" \
  --exclude="tsconfig.json" \
  --exclude="webpack.config.ts" \
  --exclude="project.json" \
  --exclude=".eslint*" \
  plugin.json img/ dist/

# 打包 macOS 版本
echo "打包 macOS 版本..."
tar -czf "../../../../${OUTPUT_DIR}/${PLUGIN_ID}-${PLUGIN_VERSION}-darwin.tar.gz" \
  --exclude="gpx_${PLUGIN_ID}_linux*" \
  --exclude="gpx_${PLUGIN_ID}_windows*" \
  --exclude="*.md" \
  --exclude="*.ts" \
  --exclude="*.tsx" \
  --exclude="*.go" \
  --exclude="node_modules" \
  --exclude="*.test.*" \
  --exclude="jest*" \
  --exclude="tsconfig.json" \
  --exclude="webpack.config.ts" \
  --exclude="project.json" \
  --exclude=".eslint*" \
  plugin.json img/ dist/

# 打包 Windows 版本
echo "打包 Windows 版本..."
tar -czf "../../../../${OUTPUT_DIR}/${PLUGIN_ID}-${PLUGIN_VERSION}-windows.tar.gz" \
  --exclude="gpx_${PLUGIN_ID}_linux*" \
  --exclude="gpx_${PLUGIN_ID}_darwin*" \
  --exclude="*.md" \
  --exclude="*.ts" \
  --exclude="*.tsx" \
  --exclude="*.go" \
  --exclude="node_modules" \
  --exclude="*.test.*" \
  --exclude="jest*" \
  --exclude="tsconfig.json" \
  --exclude="webpack.config.ts" \
  --exclude="project.json" \
  --exclude=".eslint*" \
  plugin.json img/ dist/

cd ../../../../

echo "打包完成！文件位于 ${OUTPUT_DIR}/"
ls -lh "${OUTPUT_DIR}/"
```

使用方法：
```bash
# 赋予执行权限
chmod +x package-rqlite-plugin.sh

# 运行脚本
./package-rqlite-plugin.sh
```

### 4. 部署 tar 包

将生成的 tar 包部署到 Grafana：

```bash
# 在 Grafana 服务器上
cd /var/lib/grafana/plugins

# 解压插件包
tar -xzf grafana-rqlite-datasource-*.tar.gz

# 重启 Grafana 服务
sudo systemctl restart grafana-server
```

## 开发模式

如果需要在开发时实时更新，可以使用以下命令：

```bash
# 后端（需要先构建一次）
make build-plugin-go PLUGIN_ID=grafana-rqlite-datasource

# 前端开发模式（热重载）
cd public/app/plugins/datasource/grafana-rqlite-datasource
yarn dev

# 或者在项目根目录使用
yarn plugin:build:dev
```

## 验证构建

### 1. 检查输出文件

确保 `dist/` 目录中存在以下文件：
- 前端：`module.js`
- 后端：对应平台的可执行文件

### 2. 在 Grafana 中测试

将构建好的插件部署到 Grafana 并测试：
1. 启动 Grafana 后端
2. 启动 Grafana 前端开发服务器
3. 在 Grafana 中添加 rqlite 数据源并测试连接

## 常见问题

### 问题 1：构建后端时提示 PLUGIN_ID 未设置

**解决方法**：确保在执行 `make build-plugin-go` 时正确指定了 PLUGIN_ID 参数。

### 问题 2：找不到 mage 命令

**解决方法**：安装 mage 构建工具：
```bash
go install github.com/magefile/mage@latest
```

### 问题 3：前端构建报错

**解决方法**：
- 确保已在项目根目录执行了 `yarn install --immutable`
- 检查 Node.js 版本是否符合要求
- 清理缓存并重新构建：
  ```bash
  yarn cache clean
  rm -rf node_modules
  yarn install --immutable
  ```

## 参考

- [Grafana Plugin SDK for Go](https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go)
- [Grafana 插件开发文档](https://grafana.com/developers/plugin-tools)
- [项目根目录 Makefile](file:///Users/yangqian/go/src/github.com/lynnyq/grafana/Makefile)
