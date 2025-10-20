# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个儿童专注力训练系统，使用 Go 语言开发的 Web 应用。系统提供多种训练项目（平衡超人、数字迷踪、记忆卡片、唐诗迷踪、全身唤醒等），用于追踪和记录儿童的训练会话与进度。

## 核心架构

### 单文件应用结构
- 所有后端逻辑集中在 `main.go` 中（约530行代码）
- 使用 Go 标准库 `net/http` 处理路由和 HTTP 请求
- 数据库采用 SQLite（`modernc.org/sqlite`），存储在 `data/leo_concentration.db`
- 使用 `embed.FS` 将模板和静态文件编译进二进制文件

### 数据库模式
- **trainings 表**: 训练项目定义（id, name, slug, description, category, suggested_minutes, active）
- **sessions 表**: 训练会话记录（id, training_id, started_at, ended_at, duration_seconds, success_count, error_count, notes）
- slug 字段用于路由到专门的训练页面（如 `/play/balance-hero`）

### Web 资源组织
- `web/templates/*.html`: Go HTML 模板文件
- `web/static/*`: CSS/JS 静态资源
- `web/app/`: PWA 相关文件（manifest.json, sw.js）

### 路由结构
- `/`: 首页，显示所有训练项目和今日训练总时长
- `/trainings`: 训练项目管理（GET 查看，POST 创建新项目）
- `/training?id=X`: 单个训练项目详情
- `/session/start`: POST 启动新训练会话，重定向到对应的 play 页面或通用会话页
- `/session?id=X`: 查看会话详情
- `/session/update`: POST 更新会话（成功/错误次数、备注）
- `/session/finish`: POST 结束会话，计算时长
- `/api/session/update`: JSON API，支持增量更新备注
- `/play/{slug}`: 各种训练的交互页面（balance-hero, number-trace, memory-cards, poem-trace, warmup-jumping）
- `/reports/daily`: 每日训练报告（近30天统计）
- `/static/*`: 静态资源服务

## 开发命令

### 运行应用
```bash
go run main.go
```
服务器将在 `http://localhost:8080` 启动

### 构建二进制
```bash
go build -o leo-concentration main.go
```

### 依赖管理
```bash
# 下载依赖
go mod download

# 更新依赖
go mod tidy
```

### 数据库位置
数据库文件自动创建在 `data/leo_concentration.db`，首次运行时会自动执行 migrate 和 seed

## 代码约定

### 函数命名模式
- HTTP 处理器函数使用 `handle` 前缀（如 `handleIndex`, `handleSessionStart`）
- 数据库操作函数直接命名（如 `migrate`, `seedTrainings`）
- 辅助函数直接功能命名（如 `parseInt`, `renderTemplate`）

### 错误处理
- HTTP 错误使用 `http.Error(w, message, statusCode)`
- 数据库错误使用 `log.Fatalf` (启动时) 或返回 HTTP 500 (运行时)
- 使用 `log.Printf` 记录非致命错误

### 模板渲染
- 使用 `renderTemplate(w, templateName, data)` 统一渲染
- 模板数据使用 `map[string]any` 传递
- 所有模板在启动时通过 `template.ParseFS` 预解析

### 会话工作流
1. 用户在首页或训练页点击"开始训练"
2. POST `/session/start` 创建会话记录，获得 session_id
3. 根据 training.slug 重定向到对应的 `/play/{slug}?id={session_id}` 页面
4. 训练过程中可通过 `/api/session/update` 更新计数和备注
5. 完成后 POST `/session/finish` 记录结束时间和总时长

## 未来扩展计划（参考 todolist.md）

- **AI 引导**: 引入"奇奇博士"角色，提供训练引导和正向反馈
- **语音识别**: 集成数字听写训练，使用 Vosk 或 whisper.cpp 进行离线识别
- **视频分析**: 通过 MediaPipe 或 gocv 实现动作识别和评分
- **数据模型扩展**: 增加 children、plans、events、media 表，支持结构化事件记录
- **详细评分**: 从简单的成功/错误计数升级为多维度评分系统
- **可视化报告**: 添加训练曲线图、难度进阶跟踪、详细的会话分析

## 注意事项

- 代码库较小，适合快速迭代和原型开发
- 当前没有用户认证系统，所有数据公开访问
- 数据库迁移是增量式的，使用 PRAGMA table_info 检查列是否存在
- seedTrainings 是幂等的，不会重复插入已存在的训练项目
- 静态文件和模板都嵌入到编译后的二进制文件中，部署时只需单个可执行文件和 data 目录
