# Leo专注力训练系统

一个专为儿童设计的专注力训练Web应用，提供多种互动训练项目。

## 功能特点

### 训练项目

1. **平衡超人** - 单脚站立平衡训练，提升身体控制与专注力
2. **数字迷踪** - 按顺序点击数字，训练注意力与顺序记忆
3. **记忆卡片** - 翻牌配对游戏，训练记忆力
4. **唐诗迷踪** - 按顺序点击诗句，训练语文认知
5. **全身唤醒** - 开合跳运动，激活身体能量
6. **颜色对对碰** - 识别颜色字的字义，训练抗干扰能力
7. **火眼金睛** - 在众多物品中快速找出目标，训练观察力 ⭐新增

### 核心功能

- ✅ 多种训练模式，覆盖认知、记忆、身体协调等方面
- ✅ 实时记录训练时长、成功次数、错误次数
- ✅ 每日训练报告，查看近30天训练统计
- ✅ 难度分级，适应不同年龄段儿童
- ✅ 简洁友好的界面设计
- ✅ 数据本地存储，保护隐私

## 技术架构

- **后端**: Go语言 + 标准库 net/http
- **数据库**: SQLite (modernc.org/sqlite)
- **前端**: 原生HTML/CSS/JavaScript
- **部署**: 单一可执行文件 + 嵌入式资源

## Windows系统安装指南

### 方法一：使用预编译版本（推荐）

1. 下载 `leo-concentration-windows.zip` 发布包
2. 解压到任意目录（如：`C:\leo-concentration\`）
3. 双击 `start.bat` 启动程序
4. 浏览器会自动打开 `http://localhost:8080`
5. 开始使用训练系统！

### 方法二：从源码构建

#### 前置要求

- Windows 10/11 操作系统
- Go 1.19+ 编译器 ([下载地址](https://golang.org/dl/))

#### 构建步骤

1. **下载源码**
   ```cmd
   git clone https://github.com/qianlnk/leo-concentration.git
   cd leo-concentration
   ```

2. **运行构建脚本**
   ```cmd
   build-windows.bat
   ```

3. **查看构建结果**
   - 构建完成后，`release\` 目录包含所有必需文件
   - 双击 `release\start.bat` 启动系统

## 使用说明

### 启动系统

**Windows系统**:
- 双击 `start.bat`
- 或在命令行运行: `leo-concentration.exe`

**其他系统**:
```bash
# macOS/Linux
./leo-concentration

# 或使用 Go 运行
go run main.go
```

### 访问系统

启动后在浏览器中访问: `http://localhost:8080`

### 训练流程

1. 在首页选择训练项目
2. 点击"开始训练"按钮
3. 根据提示完成训练任务
4. 系统自动记录训练数据
5. 点击"结束训练"保存记录

### 查看报告

访问 `/reports/daily` 查看每日训练统计报告

## 数据存储

所有训练数据存储在 `data/leo_concentration.db` SQLite数据库中

- 训练项目定义
- 训练会话记录
- 成功/错误次数统计
- 训练时长记录

## 目录结构

```
leo-concentration/
├── main.go              # 主程序文件
├── go.mod               # Go模块依赖
├── build-windows.bat    # Windows构建脚本
├── README.md            # 本文档
├── CLAUDE.md            # 项目说明文档
├── data/                # 数据库目录
│   └── leo_concentration.db
└── web/                 # Web资源（嵌入到二进制）
    ├── templates/       # HTML模板
    └── static/          # CSS/JS静态文件
```

## 常见问题

### Q: 如何更改服务器端口？

A: 编辑 `main.go` 文件中的 `addr := ":8080"` 行，改为其他端口，然后重新构建。

### Q: 数据库文件在哪里？

A: 在程序运行目录的 `data/leo_concentration.db`

### Q: 如何备份训练数据？

A: 复制 `data/leo_concentration.db` 文件即可

### Q: 可以多人使用吗？

A: 当前版本所有数据共享，适合家庭单用户使用。未来版本会增加多用户支持。

### Q: 为什么浏览器没有自动打开？

A: 手动在浏览器中访问 `http://localhost:8080` 即可

## 系统要求

- **操作系统**: Windows 10/11, macOS 10.15+, Linux
- **浏览器**: Chrome 90+, Edge 90+, Firefox 88+, Safari 14+
- **内存**: 最少 100MB
- **磁盘**: 最少 50MB

## 未来规划

- [ ] AI引导助手（奇奇博士角色）
- [ ] 语音识别训练
- [ ] 视频动作识别
- [ ] 多用户/多儿童管理
- [ ] 训练计划制定
- [ ] 更丰富的数据可视化
- [ ] 移动端PWA支持
- [ ] 训练成就系统

## 开源协议

本项目采用 MIT 协议开源。

## 贡献

欢迎提交 Issue 和 Pull Request！

## 联系方式

如有问题或建议，请提交 GitHub Issue。

---

**祝小朋友们训练愉快！🎉**
