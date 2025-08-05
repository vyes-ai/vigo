# Vigo

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.24-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-green)](LICENSE)

Vigo 是一个高性能、简洁易用的 Go Web 框架，专为构建现代 RESTful API 而设计。它提供了强大的路由系统、智能参数解析、灵活的中间件机制，并内置了对 GORM 的深度集成。

## 🚀 特性

- **高性能路由系统** - 基于前缀树的快速路由匹配，支持路径参数和通配符
- **智能参数解析** - 自动从 Path、Query、Header、JSON、Form 等多种来源解析参数
- **GORM 深度集成** - 内置 CRUD 操作，自动生成 RESTful API
- **丰富的中间件** - 提供 CORS、限流、缓存、SSE 等常用中间件
- **错误处理机制** - 统一的错误处理和响应格式
- **类型安全** - 基于结构体的参数验证和类型转换
- **生产就绪** - 支持 TLS、连接限制、优雅关闭等企业级特性

## 📦 安装

```bash
go mod init your-project
go get github.com/vyes-ai/vigo
```

## 🏁 快速开始

```go
package main

import (
    "github.com/vyes-ai/vigo"
    "github.com/vyes-ai/vigo/logv"
)

func main() {
    // 创建应用
    app, err := vigo.New()
    if err != nil {
        logv.Fatal().Err(err).Msg("Failed to create app")
    }

    // 注册路由
    router := app.Router()
    router.Get("/hello", hello)
    router.Get("/user/:id", getUser)

    // 启动服务
    logv.Info().Msg("Starting server on :8000")
    if err := app.Run(); err != nil {
        logv.Fatal().Err(err).Msg("Server failed")
    }
}

func hello(x *vigo.X) (any, error) {
    return map[string]string{"message": "Hello, Vigo!"}, nil
}

type getUserOpts struct {
    ID string `json:"id" parse:"path"`
}

func getUser(x *vigo.X) (any, error) {
    args := &getUserOpts{}
    if err := x.Parse(args); err != nil {
        return nil, err
    }
    
    return map[string]any{
        "id":   args.ID,
        "name": "用户" + args.ID,
    }, nil
}
```

## 📝 技术栈约束

- **框架**: vigo (github.com/vyes-ai/vigo)
- **语言**: Golang 1.24+
- **ORM**: GORM
- **数据库**: 支持 MySQL、PostgreSQL、SQLite 等 GORM 支持的数据库

## 🛣️ 路由注册

### 基础语法

```go
// 在 {resource}/init.go 或 api/init.go 中定义
var Router = vigo.NewRouter()

// 在各处理文件中注册路由
var _ = Router.Get("/", handler)                              // 列表
var _ = Router.Get("/:id", middlewareA, handler)              // 获取单个，支持中间件
var _ = Router.Post("/", "创建用户", handler)                  // 带描述信息
var _ = Router.Post("/:id/action", argOpts{}, handler)        // 带参数结构体描述
var _ = Router.Patch("/:id", handler1, handler2)              // 支持多个处理函数
var _ = Router.Delete("/:id", handler)                        // 删除

// 路由扩展 - 集成子路由
var _ = Router.Extend("/:user_id/address", address.Router)    // 嵌套资源
```

### 路径参数规则

- 路径参数使用 `:param_name` 格式
- 不同级之间的参数名必须不同
- 同级路由的参数名必须一致
- 示例：`/api/user/:user_id/role/:role_id`

### 项目结构示例

```
project/
├── init.go                  // 根路由 var Router = vigo.NewRouter()
├── api/
│   ├── init.go              // var Router = vigo.NewRouter()
│   ├── user/
│   │   ├── init.go          // var Router = vigo.NewRouter() 
│   │   ├── get.go           // var _ = Router.Get("/:user_id", getUser)
│   │   ├── list.go          // var _ = Router.Get("/", listUsers)
│   │   ├── create.go        // var _ = Router.Post("/", createUser)
│   │   └── address/
│   │       ├── init.go      // var Router = vigo.NewRouter()
│   │       └── get.go       // var _ = Router.Get("/:address_id", getAddress)
│   └── role/
│       ├── init.go
│       └── ...
├── models/
│   └── user.go
└── cfg/
|    └── db.go                // 数据库配置
└── cli/main.go               // 可执行程序入口
```

## 🔧 参数解析

使用结构体 + `parse` 标签进行参数解析：

```go
type requestOpts struct {
    // 路径参数 - 必选参数用非指针类型
    ID       string  `json:"id" parse:"path"`                    // 路径参数名与字段名一致
    UserID   string  `json:"user_id" parse:"path@user_id"`       // 使用别名指定路径参数名
    
    // 请求头参数
    Token    string  `json:"token" parse:"header@Authorization"`
    
    // 查询参数
    Page     int     `json:"page" parse:"query" default:"1"`     // 必选参数，有默认值
    Size     int     `json:"size" parse:"query"`                 // 必选参数，无默认值时缺失会报错
    Keyword  *string `json:"keyword" parse:"query"`              // 可选参数用指针类型
    
    // JSON 数据 (Content-Type: application/json)
    Name     string  `json:"name" parse:"json"`
    Email    string  `json:"email" parse:"json"`
    
    // 表单数据 (Content-Type: application/x-www-form-urlencoded)
    Title    string  `json:"title" parse:"form"`
    Content  string  `json:"content" parse:"form"`
    
    // 文件上传 (Content-Type: multipart/form-data)
    Avatar   *multipart.FileHeader   `json:"avatar" parse:"form"`    // 单个文件
    Files    []*multipart.FileHeader `json:"files" parse:"form"`     // 多个文件
}

func handler(x *vigo.X) (any, error) {
    args := &requestOpts{}
    if err := x.Parse(args); err != nil {
        return nil, err // 框架自动处理 400 错误响应
    }
    
    // 使用解析后的参数
    if args.Keyword != nil {
        // 处理可选参数
    }
    
    return result, nil
}
```

### 参数解析规则

- **必选参数**: 非指针类型，缺失时返回错误
- **可选参数**: 指针类型，缺失时为 nil
- **默认值**: 通过 `default` 标签设置，仅对必选参数生效
- **参数别名**: 使用 `@` 指定别名，如 `parse:"path@user_id"`

## ⚡ 处理函数

### 标准签名

所有处理函数和中间件必须使用统一签名：

```go
func handlerName(x *vigo.X) (any, error) {
    // 实现逻辑
    return result, nil // result 可以是任意可序列化对象或 nil
}
```

### 响应格式规范

```go
// 成功获取单个资源/创建/更新
return userModel, nil

// 删除成功
return map[string]string{"message": "删除成功"}, nil

// 列表查询 - 必须包含 total 和 items
return map[string]any{
    "total": count,
    "items": users,
}, nil

// 错误处理
return nil, vigo.NewError("错误信息").WithCode(404)
```


### CRUD 操作示例

```go
import (
    "errors"
    "gorm.io/gorm"
    "your-project/cfg"
    "your-project/models"
)

// 获取单个资源
func getUser(x *vigo.X) (any, error) {
    args := &getUserOpts{}
    if err := x.Parse(args); err != nil {
        return nil, err
    }

    var user models.User
    if err := cfg.DB().Where("id = ?", args.ID).First(&user).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, vigo.NewError("用户不存在").WithCode(404)
        }
        return nil, vigo.NewError("数据库查询错误").WithError(err)
    }
    return user, nil
}

// 列表查询（支持分页）
func listUsers(x *vigo.X) (any, error) {
    args := &listOpts{}
    if err := x.Parse(args); err != nil {
        return nil, err
    }

    var users []models.User
    var total int64
    
    query := cfg.DB().Model(&models.User{})
    // 添加查询条件
    if args.Keyword != nil {
        query = query.Where("name LIKE ?", "%"+*args.Keyword+"%")
    }
    
    // 计算总数
    if err := query.Count(&total).Error; err != nil {
        return nil, vigo.NewError("数据库计数错误").WithError(err)
    }
    
    // 分页查询
    offset := (args.Page - 1) * args.PageSize
    if err := query.Offset(offset).Limit(args.PageSize).Find(&users).Error; err != nil {
        return nil, vigo.NewError("数据库查询错误").WithError(err)
    }
    
    return map[string]any{"total": total, "items": users}, nil
}

// 创建资源
func createUser(x *vigo.X) (any, error) {
    args := &createUserOpts{}
    if err := x.Parse(args); err != nil {
        return nil, err
    }

    newUser := models.User{Name: args.Name, Email: args.Email}
    if err := cfg.DB().Create(&newUser).Error; err != nil {
        return nil, vigo.NewError("创建失败").WithError(err)
    }
    return newUser, nil
}

// 更新资源
func updateUser(x *vigo.X) (any, error) {
    args := &updateUserOpts{}
    if err := x.Parse(args); err != nil {
        return nil, err
    }

    var existingUser models.User
    if err := cfg.DB().First(&existingUser, args.ID).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, vigo.NewError("用户不存在").WithCode(404)
        }
        return nil, vigo.NewError("数据库查询错误").WithError(err)
    }

    // 只更新传入的字段
    updates := models.User{Name: args.Name, Email: args.Email}
    if err := cfg.DB().Model(&existingUser).Select("name", "email").Updates(updates).Error; err != nil {
        return nil, vigo.NewError("更新失败").WithError(err)
    }
    return existingUser, nil
}

// 删除资源
func deleteUser(x *vigo.X) (any, error) {
    args := &deleteUserOpts{}
    if err := x.Parse(args); err != nil {
        return nil, err
    }

    result := cfg.DB().Where("id = ?", args.ID).Delete(&models.User{})
    if result.Error != nil {
        return nil, vigo.NewError("删除失败").WithError(result.Error)
    }
    if result.RowsAffected == 0 {
        return nil, vigo.NewError("用户不存在或已被删除").WithCode(404)
    }
    return map[string]string{"message": "删除成功"}, nil
}
```

## 🧰 自动 CRUD

Vigo 提供了自动 CRUD 功能，可以快速生成标准的 RESTful API：

```go
import "github.com/vyes-ai/vigo/contrib/crud"

// 自动生成完整 CRUD API
crud.All(router, cfg.DB, &models.User{})

// 或者单独注册
router.Get("/:id", crud.Get(cfg.DB, &models.User{}))
router.Get("/", crud.List(cfg.DB, &models.User{}))
router.Post("/", crud.Create(cfg.DB, &models.User{}))
router.Patch("/:id", crud.Update(cfg.DB, &models.User{}))
router.Delete("/:id", crud.Delete(cfg.DB, &models.User{}))
```

## 🔗 中间件

### CORS 中间件

```go
import "github.com/vyes-ai/vigo/contrib/cors"

// 允许所有跨域请求
router.UseBefore(cors.AllowAny)

// 指定允许的域名
router.UseBefore(cors.CorsAllow("https://example.com", "https://app.example.com"))
```

### 缓存中间件

```go
import (
    "time"
    "github.com/vyes-ai/vigo/contrib/cache"
)

// 缓存 GET 请求 10 秒
cacheMiddleware := cache.NewCacheMiddleware(yourHandler, 10*time.Second)
router.Get("/data", cacheMiddleware.Handler)
```

### 自定义中间件

```go
func authMiddleware(x *vigo.X) (any, error) {
    token := x.Request.Header.Get("Authorization")
    if token == "" {
        return nil, vigo.NewError("缺少认证token").WithCode(401)
    }
    
    // 验证 token 逻辑
    userID, err := validateToken(token)
    if err != nil {
        return nil, vigo.NewError("无效token").WithCode(401)
    }
    
    // 将用户信息存储到请求上下文
    x.Set("user_id", userID)
    return nil, nil // 中间件返回 nil 继续执行后续处理函数
}

// 使用中间件
router.Get("/profile", authMiddleware, getProfile)
router.UseBefore(authMiddleware) // 对整个路由组应用
```

## ❌ 错误处理

### 标准错误

```go
// 预定义错误
return nil, vigo.ErrNotFound                                    // 404
return nil, vigo.ErrNotAuthorized                              // 401
return nil, vigo.ErrForbidden                                  // 403
return nil, vigo.ErrInternalServer                             // 500

// 自定义错误
return nil, vigo.NewError("参数无效")                            // 默认 400
return nil, vigo.NewError("资源不存在").WithCode(404)              // 指定状态码
return nil, vigo.NewError("查询失败").WithError(dbErr)            // 包装底层错误
return nil, vigo.NewError("用户 %s 不存在").WithArgs(username)     // 格式化消息
```


## 🔧 高级配置

### 服务器配置

```go
app, err := vigo.New(
    func(c *vigo.RestConf) {
        c.Host = "0.0.0.0"
        c.Port = 8080
    },
)
```

### TLS 配置

```go
import "crypto/tls"

app, err := vigo.New(
    func(c *vigo.RestConf) {
        c.TlsCfg = &tls.Config{
            // TLS 配置
        }
    },
)
```

### 多域名支持

```go
// 主域名路由
mainRouter := app.Router()
mainRouter.Get("/", mainHandler)

// 子域名路由
apiRouter := app.Domain("api.example.com")
apiRouter.Get("/v1/users", listUsers)

// 通配符域名
subRouter := app.Domain("*.example.com")
subRouter.Get("/health", healthCheck)
```

## 📊 Server-Sent Events (SSE)

```go
func sseHandler(x *vigo.X) (any, error) {
    writer := x.SSEEvent()
    
    // 发送事件
    writer("message", "Hello from SSE")
    writer("update", map[string]any{"count": 42})
    
    return nil, nil
}

router.Get("/events", sseHandler)
```

## 🎯 完整示例

```go
// api/user/init.go
package user

import "github.com/vyes-ai/vigo"

var Router = vigo.NewRouter()

// api/user/get.go
package user

import (
    "errors"
    "gorm.io/gorm"
    "github.com/vyes-ai/vigo"
    "your-project/models"
    "your-project/cfg"
)

type getUserOpts struct {
    ID string `json:"id" parse:"path@user_id"`
}

var _ = Router.Get("/:user_id", "获取用户信息", getUserOpts{}, getUser)

func getUser(x *vigo.X) (any, error) {
    args := &getUserOpts{}
    if err := x.Parse(args); err != nil {
        return nil, err
    }

    var user models.User
    if err := cfg.DB().Where("id = ?", args.ID).First(&user).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, vigo.NewError("用户不存在").WithCode(404)
        }
        return nil, vigo.NewError("数据库查询失败").WithError(err)
    }
    
    return user, nil
}

// api/init.go
package api

import "github.com/vyes-ai/vigo"
import "your-project/api/user"

var Router = vigo.NewRouter()

var _ = Router.Extend("/user", user.Router)

// main.go
package main

import (
    "github.com/vyes-ai/vigo"
    "github.com/vyes-ai/vigo/contrib/cors"
    "your-project/api"
)

func main() {
    app, err := vigo.New()
    if err != nil {
        panic(err)
    }

    // 配置 CORS
    app.Router().UseBefore(cors.AllowAny)
    
    // 注册 API 路由
    app.Router().Extend("/api", api.Router)
    
    // 启用 AI 接口文档
    app.EnableAI()
    
    // 启动服务
    if err := app.Run(); err != nil {
        panic(err)
    }
}
```

## 📖 API 文档

启用 AI 功能后，框架会自动生成 API 文档：

```go
app.EnableAI() // 启用后访问 /api.json 获取 API 列表
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

本项目基于 [Apache License 2.0](LICENSE) 许可证开源。。

---

**Vigo** - 让 Go Web 开发更简单、更高效！
