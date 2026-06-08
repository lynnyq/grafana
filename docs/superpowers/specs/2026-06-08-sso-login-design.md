# Grafana SSO 登录集成设计文档

## 概述

在 Grafana 中集成第三方 SSO 登录认证，基于 HTTP API 模式（参考 SSO_LOGIN.md / SSO_LOGIN_TECH_DOC.md），复用 Grafana 已有的 OAuth2/social connector 框架。

## 需求

1. 前端登录页面支持 SSO 和本地用户两种登录方式，Tab 切换，默认 Tab 为 SSO 登录
2. SSO 登录对接第三方 SSO HTTP API（用户名 + RSA 加密密码验证）
3. SSO 用户首次登录自动创建，默认 Viewer 角色
4. 管理员可手动修改 SSO 用户的角色，修改后不受后续 SSO 登录覆盖

## 架构设计

### 整体流程

```
前端 SSO Tab → 用户输入用户名+密码 → RSA加密密码 → POST /api/sso/login
→ 后端调用 SSO HTTP API 验证 → 返回 code:3000 → 自动创建/匹配 Grafana 用户 → 建立 Session
```

### 角色策略

- **新用户**：SSO 首次登录自动创建，默认分配 Viewer 角色
- **已有用户**：SSO 再次登录时不覆盖角色，保留管理员手动设置的值
- **实现方式**：SSO connector 设置 `SkipOrgRoleSync = true`，角色同步由 `user_sync` 在首次创建时通过 `default_role` 配置决定

## 后端改动

### 1. 新增 SSO Connector

**文件**：`pkg/login/social/connectors/sso_api.go`

- 实现 `social.SocialConnector` 接口
- 不走 OAuth2 授权码流程，`AuthCodeURL`/`Exchange` 返回占位值
- `UserInfo` 方法：调用 SSO HTTP API（`/api/v1/authenticate`），用 RSA 公钥加密密码，验证用户身份
- 返回 `BasicUserInfo`，`SkipOrgRoleSync = true`，不在每次登录时同步角色

**SSO API 调用流程**（参考 SSO_LOGIN 文档）：
1. 使用 RSA 公钥加密密码
2. POST 请求到 SSO `/api/v1/authenticate`，body: `{"username": "xxx", "password": "encrypted_password"}`
3. 验证返回 `code: 3000` 表示成功，提取用户信息（username、email、display_name）

### 2. 新增 SSO 登录 API

**文件**：`pkg/api/sso_login.go`

- `POST /api/sso/login`：接收 `{username, password}`
- 调用 SSO connector 的 UserInfo 方法验证用户
- 复用 Grafana `authn` + `user_sync` 机制自动创建/匹配用户
- 建立 Session，返回登录成功响应

### 3. 配置段

**文件**：`conf/defaults.ini`

```ini
[auth.sso]
enabled = false
name = SSO
icon = signin
allow_sign_up = true
auto_login = false
sso_api_url =                    # SSO API 地址，如 https://sso.example.com/api/v1
sso_rsa_public_key =             # RSA 公钥，用于加密密码
default_role = Viewer            # SSO 新用户默认角色
tls_skip_verify_insecure = false
```

### 4. 注册 SSO Provider

- `pkg/login/social/social.go`：添加 `SSOProviderName = "sso"`
- `pkg/services/ssosettings/ssosettingsimpl/service.go`：注册 SSO connector
- `pkg/api/login_oauth.go`：添加 SSO 路由
- `pkg/api/router.go`：注册 `/api/sso/login` 路由

## 前端改动

### 1. 登录页面 Tab 切换

**文件**：`public/app/core/components/Login/LoginPage.tsx`

- 改为 Tab 布局：
  - **Tab 1（默认）**：SSO 登录 — 用户名+密码表单
  - **Tab 2**：本地登录 — 原有 Grafana 用户名+密码表单
- 当 SSO 未启用时，隐藏 SSO Tab，直接显示本地登录

### 2. SSO 登录表单组件

**文件**：`public/app/core/components/Login/SSOLoginForm.tsx`

- 用户名+密码输入框
- 密码在提交前使用 RSA 公钥加密
- 提交到 `POST /api/sso/login`

### 3. 配置传递

- `public/app/core/config.ts`：添加 SSO 相关配置字段（ssoEnabled、ssoName、ssoRsaPublicKey）
- 后端通过 `/api/frontend/settings` 将 SSO 配置传递给前端

## 安全考虑

- 密码使用 RSA 公钥加密后传输，后端不解密，直接转发给 SSO API
- SSO API 返回的 token 不存储，仅用于验证
- RSA 公钥通过后端配置管理，前端通过 `/api/frontend/settings` 获取
- SSO 登录 API 需要限流保护，防止暴力破解

## 文件清单

### 新增文件
| 文件 | 说明 |
|------|------|
| `pkg/login/social/connectors/sso_api.go` | SSO connector 实现 |
| `pkg/api/sso_login.go` | SSO 登录 API handler |
| `public/app/core/components/Login/SSOLoginForm.tsx` | SSO 登录表单组件 |

### 修改文件
| 文件 | 说明 |
|------|------|
| `pkg/login/social/social.go` | 添加 SSOProviderName |
| `conf/defaults.ini` | 添加 [auth.sso] 配置段 |
| `pkg/api/router.go` | 注册 SSO 登录路由 |
| `public/app/core/components/Login/LoginPage.tsx` | Tab 切换布局 |
| `public/app/core/config.ts` | 添加 SSO 配置字段 |
| `pkg/services/ssosettings/ssosettingsimpl/service.go` | 注册 SSO connector |
