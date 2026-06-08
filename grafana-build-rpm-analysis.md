# Grafana `make build-rpm` 详细解析

## 概述

Grafana 使用 **FPM (Effing Package Manager)** 工具来构建 RPM 包，而不是传统的 `.spec` 文件方式。这种方式更加灵活、高效，特别适合跨平台打包场景。

## 完整构建流程

```
make build-rpm
    ↓
[1] 先构建 tar.gz (build-targz)
    ├→ 编译 Go 后端 (bin/linux/amd64/grafana)
    ├→ 构建前端资源 (public/build)
    ├→ 下载默认插件 (data/plugins-bundled)
    └→ 组装 tar.gz 文件
    ↓
[2] 调用 build-rpm.sh 脚本
    ├→ 解压 tar.gz 到临时目录
    ├→ 创建 RPM 目录结构
    ├→ 复制配置文件和脚本
    └→ 使用 fpm 打包为 RPM
    ↓
[3] 输出 RPM 文件
```

## 详细步骤分析

### 步骤 1：Makefile 依赖关系

```makefile
# Makefile:441
build-rpm: $(RPM_FILE)

$(RPM_FILE): $(TARGZ_FILE)
    @echo "building rpm"
    TARGZ_PACKAGE_NAME="$(TARGZ_PACKAGE_NAME)" \
    BUILD_VERSION="$(BUILD_VERSION)" \
    BUILD_NUMBER="$(BUILD_NUMBER)" \
    OS="$(OS)" \
    ARCH="$(ARCH)" \
    FPM_LICENSE="$(FPM_LICENSE)" \
    $(if $(ARM),GOARM="$(ARM)") \
    bash scripts/build-rpm.sh
```

**关键点**：RPM 构建依赖于 `$(TARGZ_FILE)`，即先构建 tar.gz。

### 步骤 2：构建 tar.gz ([scripts/build-targz.sh](file:///Users/yangqian/go/src/github.com/lynnyq/grafana/scripts/build-targz.sh))

**前置依赖**：
- Go 后端编译：`bin/linux/amd64/grafana`
- 前端构建：`public/build`
- 默认插件：`data/plugins-bundled`

**打包内容**：
```bash
# 目录结构：grafana-${VERSION}/
├── bin/                    # Go 二进制文件
│   └── linux/
│       └── amd64/
│           └── grafana
├── conf/                   # 配置文件
├── data/
│   └── plugins-bundled/    # 预装插件
├── docs/
│   └── sources/            # 文档
├── public/                 # 前端资源
├── packaging/               # 打包脚本
│   ├── deb/
│   ├── rpm/
│   ├── docker/
│   └── wrappers/
├── VERSION                 # 版本文件
├── LICENSE
├── NOTICE.md
├── README.md
└── Dockerfile
```

**输出**：`dist/grafana_${BUILD_VERSION}_${BUILD_NUMBER}_linux_amd64.tar.gz`

### 步骤 3：RPM 打包 ([scripts/build-rpm.sh](file:///Users/yangqian/go/src/github.com/lynnyq/grafana/scripts/build-rpm.sh))

#### 3.1 解压 tar.gz

```bash
TARGZ="${REPO_ROOT}/dist/${TARGZ_PACKAGE_NAME}_${BUILD_VERSION}_${BUILD_NUMBER}_${OS}_${ARCH_LABEL}.tar.gz"
tar --exclude=storybook --strip-components=1 -xf "${TARGZ}" -C "${SRC}"
```

#### 3.2 创建 RPM 目录结构

```bash
mkdir -p \
  "${PKG}/usr/sbin" \           # 可执行文件
  "${PKG}/usr/share" \           # 程序文件
  "${PKG}/etc/sysconfig" \       # 环境配置
  "${PKG}/etc/grafana" \         # 配置文件
  "${PKG}/usr/lib/systemd/system"  # systemd 服务
```

#### 3.3 复制包装脚本

```bash
cp "${SRC}/packaging/wrappers/grafana" "${PKG}/usr/sbin/"
cp "${SRC}/packaging/wrappers/grafana-server" "${PKG}/usr/sbin/"
cp "${SRC}/packaging/wrappers/grafana-cli" "${PKG}/usr/sbin/"
chmod 0755 "${PKG}/usr/sbin/grafana" "${PKG}/usr/sbin/grafana-server" "${PKG}/usr/sbin/grafana-cli"
```

#### 3.4 复制完整程序

```bash
cp -r "${SRC}" "${PKG}/usr/share/grafana"
```

#### 3.5 复制 RPM 特定配置文件

```bash
cp "${SRC}/packaging/rpm/sysconfig/grafana-server" "${PKG}/etc/sysconfig/grafana-server"
cp "${SRC}/packaging/rpm/systemd/grafana-server.service" "${PKG}/usr/lib/systemd/system/grafana-server.service"
```

#### 3.6 使用 FPM 打包

```bash
fpm \
  --input-type=dir \                    # 输入类型：目录
  --chdir="${PKG}" \                    # 工作目录
  --output-type=rpm \                   # 输出类型：RPM
  --vendor="Grafana Labs" \
  --url=https://grafana.com \
  --maintainer=contact@grafana.com \
  --version="${RPM_VERSION}" \
  --package="${REPO_ROOT}/dist/${FILENAME}" \
  --config-files=/etc/sysconfig/grafana-server \
  --config-files=/usr/lib/systemd/system/grafana-server.service \
  --after-install="${SRC}/packaging/rpm/control/postinst" \
  --depends=/sbin/service \
  --architecture="${PKG_ARCH}" \
  --description=Grafana \
  --license="${FPM_LICENSE:-AGPLv3}" \
  --name="${TARGZ_PACKAGE_NAME}" \
  --rpm-posttrans="${SRC}/packaging/rpm/control/posttrans" \
  --rpm-digest=sha256 \
  --rpm-compression xzmt \
  --rpm-user root \
  --rpm-group root \
  .
```

## FPM 工具介绍

### 什么是 FPM？

**FPM (Effing Package Manager)** 是一个用 Ruby 编写的打包工具，由 Mike Lankamp 开发。它可以将多种格式的输入（目录、gem、rpm、deb 等）转换为目标包格式（rpm、deb、solaris 等）。

### FPM 核心特性

| 特性 | 说明 |
|------|------|
| **多格式支持** | 输入：dir、gem、rpm、deb、tar、npm 等<br>输出：rpm、deb、solaris、puppet 等 |
| **简单易用** | 命令行参数驱动，无需复杂配置 |
| **元数据控制** | 支持设置 vendor、license、description、depends 等 |
| **钩子脚本** | 支持 pre-install、post-install、pre-uninstall、post-uninstall 脚本 |
| **跨平台** | 可在 Linux 构建 Debian 包，在 macOS 构建 RPM 包 |

### FPM 常用参数

```bash
# 基本用法
fpm -s <源类型> -t <目标类型> [选项] <源路径>

# 常用参数
-s dir                  # 源类型：目录
-t rpm                 # 目标类型：RPM
-n <name>              # 包名称
-v <version>           # 版本号
--vendor <vendor>      # 供应商
--description <desc>   # 描述
--depends <dep>        # 依赖
--provides <feature>   # 提供功能
--config-files <path>  # 配置文件
--before-install <scr> # 安装前脚本
--after-install <scr>   # 安装后脚本
--before-removal <scr> # 卸载前脚本
--after-removal <scr>  # 卸载后脚本
--rpm-posttrans <scr>   # RPM 事务后脚本
--rpm-compression xz   # RPM 压缩算法
--rpm-digest sha256    # RPM 摘要算法
```

## 安装 FPM

```bash
# CentOS/RHEL
sudo yum install -y ruby ruby-devel rubygems rpm-build gcc make
gem install fpm

# Debian/Ubuntu
sudo apt-get install -y ruby ruby-dev rubygems build-essential
gem install fpm

# macOS
brew install gomplate fpm
```

## Grafana RPM 包结构

### 文件布局

```
/etc/grafana/                    # 配置目录
├── grafana.ini                  # 主配置文件
├── ldap.toml                    # LDAP 配置
└── provisioning/               # 自动配置
    ├── dashboards/
    ├── datasources/
    ├── plugins/
    ├── access-control/
    └── alerting/

/etc/sysconfig/grafana-server    # 环境配置

/etc/default/grafana            # 默认配置

/var/lib/grafana/               # 数据目录
├── plugins/                    # 插件
└── ...

/var/log/grafana/              # 日志目录

/usr/share/grafana/            # 程序目录
├── bin/
│   └── grafana
├── public/
├── conf/
├── ...

/usr/sbin/
├── grafana                     # 包装脚本
├── grafana-server              # 服务器启动脚本
└── grafana-cli                # CLI 工具
```

### RPM 包元数据

```bash
# 查看 RPM 包信息
rpm -qip grafana-13.1.0-1.x86_64.rpm

# 包信息示例
Name        : grafana
Version     : 13.1.0
Release     : 1
Vendor      : Grafana Labs
URL         : https://grafana.com
License     : AGPLv3
Summary     : Grafana
Arch        : x86_64
Size        : 185432567
Buildhost   : build.local
Prefix      : /
```

## Grafana RPM 生命周期脚本

### postinst（安装后脚本）

```bash
# 首次安装 ($1 == 1)
1. 创建 grafana 用户和组
2. 创建数据目录 (/var/log/grafana, /var/lib/grafana)
3. 移动预装插件到数据目录
4. 设置目录权限
5. 复制配置文件（如果不存在）
6. 创建配置目录并复制示例配置
7. 设置配置文件权限

# 升级 ($1 >= 2)
1. 如果 RESTART_ON_UPGRADE=true，停止再启动服务
```

### posttrans（事务后脚本）

```bash
# 配置文件恢复
1. 检查 /etc/grafana/grafana.ini 是否存在
2. 如果不存在但有 grafana.ini.rpmsave，从备份恢复
3. 恢复配置文件权限
```

## FPM vs 传统 Spec 打包对比

### 传统 Spec 打包流程

```
编写 .spec 文件
    ↓
rpmbuild -ba xxx.spec
    ↓
[1] %prep    - 解压源码
    ↓
[2] %build   - 编译构建
    ↓
[3] %install - 安装到 BUILDROOT
    ↓
[4] %files   - 定义包文件列表
    ↓
[5] %changelog - 变更日志
    ↓
[6] 生成 RPM
```

### 关键差异对比

| 方面 | FPM 方式 | 传统 Spec 方式 |
|------|---------|---------------|
| **配置方式** | 命令行参数 + Shell 脚本 | `.spec` 文件定义 |
| **构建依赖** | 仅需 FPM 工具 | 需要完整 RPM build 环境 |
| **灵活性** | 高 - 可用 Shell 脚本做任何预处理 | 低 - 受 spec 语法限制 |
| **复杂度** | 低 - 几行命令即可打包 | 高 - 需要编写完整的 spec 文件 |
| **跨平台** | 支持（可在 macOS 构建 RPM） | 仅限 Linux |
| **依赖管理** | `--depends` 参数 | Requires 字段 |
| **钩子脚本** | `--before-install` 等参数 | %pre/%post/%preun/%postun |
| **文件列表** | 自动扫描目录 | 手动在 %files 中定义 |
| **二次开发** | 简单 - 直接修改 Shell 脚本 | 复杂 - 需要理解 spec 语法 |
| **企业级特性** | 基础 | 支持 SELinux、triggers 等高级特性 |

### FPM 方式优势

1. **简单直观**：不需要学习复杂的 spec 语法
2. **易于维护**：Shell 脚本比 spec 文件更易读
3. **灵活预处理**：可以在打包前做任何文件操作
4. **快速迭代**：修改后立即重新打包
5. **跨平台构建**：同一个脚本可构建 RPM、DEB、Solaris 等

### FPM 方式劣势

1. **元数据限制**：无法使用高级 RPM 特性（如 SELinux contexts）
2. **规范遵循**：可能不完全符合某些 Linux 发行版的打包规范
3. **调试困难**：出现问题时排查不如 spec 直观
4. **依赖解析**：不如 rpmbuild 智能

### 传统 Spec 方式优势

1. **规范性**：符合 Linux 发行版标准
2. **高级特性**：支持 SELinux、triggers、dependencies 等
3. **自动依赖**：能自动分析二进制依赖
4. **验证工具**：有 rpmdevtools 等工具辅助

### 传统 Spec 方式劣势

1. **学习曲线**：语法复杂，学习成本高
2. **维护困难**：spec 文件难以理解和修改
3. **调试复杂**：问题排查需要丰富经验
4. **灵活性差**：难以实现复杂预处理逻辑

## 实际应用场景

### 何时使用 FPM

- ✅ 快速打包内部工具
- ✅ 跨平台打包（Linux → RPM/DEB）
- ✅ 微服务容器镜像中的包构建
- ✅ CI/CD 流水线中的自动化打包
- ✅ 需要灵活预处理逻辑的场景

### 何时使用传统 Spec

- ✅ 需要提交到 Linux 发行版官方仓库
- ✅ 需要严格遵循发行版打包规范
- ✅ 需要 SELinux 上下文等高级特性
- ✅ 企业内部有成熟的 spec 打包流程

## 实际使用示例

### 使用 FPM 打包一个简单应用

```bash
#!/bin/bash
# 1. 构建应用
go build -o myapp ./cmd/myapp

# 2. 准备目录
mkdir -p pkg/usr/local/bin

# 3. 复制文件
cp myapp pkg/usr/local/bin/

# 4. 打包 RPM
fpm \
  -s dir \
  -t rpm \
  -n myapp \
  -v 1.0.0 \
  --vendor "My Company" \
  --description "My Application" \
  --depends "openssl" \
  --after-install scripts/postinst.sh \
  --before-remove scripts/prerm.sh \
  pkg
```

### 传统 Spec 方式

```spec
Name:           myapp
Version:        1.0.0
Release:        1%{?dist}
Summary:        My Application
License:        MIT
Vendor:         My Company

%description
My Application is a tool for...

%prep
# 解压源码（此处为预编译二进制）

%build
# 编译步骤

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}%{_bindir}
install -p myapp %{buildroot}%{_bindir}/

%files
%defattr(-,root,root,-)
%{_bindir}/myapp

%post
# 安装后脚本

%postun
# 卸载后脚本
```

## 总结

Grafana 选择 **FPM** 作为打包工具的主要原因：

1. **开发效率优先**：Grafana 团队需要快速迭代，FPM 提供了更快的打包速度
2. **跨平台支持**：FPM 可以在不同平台构建不同格式的包
3. **灵活性**：Shell 脚本方式更易于集成到现有的构建系统
4. **维护成本低**：相比复杂的 spec 文件，Shell 脚本更易读易维护
5. **CI/CD 友好**：易于在自动化流水线中集成

对于企业内部使用或追求规范性，FPM 是一个很好的选择；对于需要提交到官方仓库或严格遵循发行版规范，传统 spec 方式仍然是金标准。
