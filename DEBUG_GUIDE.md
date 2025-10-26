# 🔍 数据记录调试指南

## 问题：玩完游戏后没有数据记录

### ✅ 快速检查步骤

#### 步骤1：打开浏览器开发者工具

1. 访问游戏页面：http://localhost:8080
2. 按 **F12** 打开开发者工具
3. 切换到 **Console（控制台）** 标签

#### 步骤2：开始游戏并观察日志

1. 点击"眼疾手快"的"开始训练"
2. 点击"开始游戏"按钮
3. 在控制台中查找以下日志：

**期望看到的日志**：
```
🔵 开始记录轮次 {sessionId: 117, currentRound: 1}
✅ 轮次记录开始成功 {roundId: 32, response: {ok: true, round_id: 32}}
```

**如果看到以下日志**：
```
❌ 开始轮次失败: [错误信息]
```
说明API调用失败，需要检查网络或后端。

#### 步骤3：完成一轮游戏

1. 观察球的位置
2. 杯子交换后点击你认为有球的杯子
3. 在控制台中查找：

**期望看到的日志**：
```
🔵 准备结束轮次 {currentRoundId: 32, success: true}
🔵 发送结束轮次请求 {roundId: 32, success: true, metadata: "..."}
✅ 轮次记录结束成功 {ok: true}
```

**如果看到以下日志**：
```
⚠️ 没有currentRoundId，跳过结束轮次记录
```
说明startRound没有成功获取roundId。

#### 步骤4：检查Network（网络）标签

1. 切换到开发者工具的 **Network（网络）** 标签
2. 刷新页面并重新开始游戏
3. 查找以下请求：

**应该看到的请求**：
```
POST /api/session/round/start
  Status: 200
  Response: {"ok":true,"round_id":32}

POST /api/session/round/finish
  Status: 200
  Response: {"ok":true}
```

**如果看到 404 或 500 错误**：
- 404：路由不存在，检查服务器是否正确编译
- 500：服务器错误，检查后端日志

---

## 🧪 使用测试页面

我专门创建了一个测试页面，可以直接测试API：

### 访问测试页面
```
http://localhost:8080/test/cup-ball
```

### 测试步骤
1. 点击 "1. 创建会话" - 应该显示 Session ID
2. 点击 "2. 开始轮次" - 应该显示 Round ID
3. 点击 "3. 结束轮次" - 应该显示成功消息

**如果任何步骤失败**，日志区域会显示具体错误信息。

---

## 🔎 常见问题诊断

### 问题1：控制台没有任何日志
**可能原因**：
- 浏览器缓存了旧代码
- 页面没有正确加载

**解决方法**：
1. 硬刷新页面（Ctrl+F5 或 Cmd+Shift+R）
2. 清除浏览器缓存
3. 重新访问页面

---

### 问题2：看到 "⚠️ 没有currentRoundId"
**可能原因**：
- startRound函数没有被调用
- startRound API调用失败
- API返回格式不正确

**解决方法**：
1. 检查控制台是否有 "🔵 开始记录轮次" 日志
2. 如果没有，说明startRound根本没被调用
3. 如果有但没有 "✅ 轮次记录开始成功"，检查Network标签

---

### 问题3：Network标签看到404错误
**可能原因**：
- 服务器没有正确编译
- 路由配置错误

**解决方法**：
```bash
# 重新编译
cd /Users/xiezhenjia/go/src/github.com/qianlnk/leo-concentration
go build -o leo-concentration main.go

# 重启服务器
pkill -f "leo-concentration"
./leo-concentration
```

---

### 问题4：Network标签看到500错误
**可能原因**：
- 数据库连接问题
- API参数错误
- 后端代码bug

**解决方法**：
```bash
# 查看服务器日志
tail -f /tmp/leo.log

# 或者
tail -f nohup.out
```

---

## 📊 验证数据是否真的记录了

### 方法1：查询数据库
```bash
# 查询最近的轮次记录
sqlite3 data/leo_concentration.db "SELECT id, session_id, round_number, success, score FROM session_rounds ORDER BY id DESC LIMIT 10;"
```

### 方法2：使用检查脚本
```bash
./check_data.sh
```

### 方法3：访问统计页面
```
http://localhost:8080/reports/daily/v2
```

---

## 🎯 针对不同游戏的检查

### 眼疾手快 (已集成) ✅
- 路由：`/play/cup-ball?id=X`
- 应该有console.log日志
- 每完成一轮就会记录

### 数字迷踪 (已集成) ✅
- 路由：`/play/number-trace?id=X`
- 每完成一个级别（3x3/4x4/5x5）记录一轮

### 火眼金睛 (已集成) ✅
- 路由：`/play/eagle-eye?id=X`
- 每完成一次游戏记录一轮

### 其他游戏 (未集成) ❌
如果玩的是：
- 记忆卡片
- 唐诗迷踪
- 颜色对对碰
- 平衡超人
- 全身唤醒

**这些游戏还没有集成数据记录功能**，所以不会有任何数据。

---

## 🛠️ 我已经添加的调试功能

### 1. Console日志
在游戏页面中，每个关键步骤都会输出日志：
- 🔵 蓝色：操作开始
- ✅ 绿色：操作成功
- ❌ 红色：操作失败
- ⚠️ 黄色：警告信息

### 2. 测试页面
- URL：`/test/cup-ball`
- 提供独立的API测试环境
- 显示详细的调用日志

### 3. 检查脚本
- 文件：`check_data.sh`
- 快速查看数据记录状态
- 显示集成情况

---

## 📝 完整测试流程示例

### 1. 打开浏览器
```
http://localhost:8080
```

### 2. 打开开发者工具（F12）

### 3. 开始玩眼疾手快
- 点击"眼疾手快"卡片的"开始训练"
- 点击"开始游戏"按钮

### 4. 观察Console（控制台）
应该看到：
```
🔵 开始记录轮次 {sessionId: 118, currentRound: 1}
✅ 轮次记录开始成功 {roundId: 33, response: {…}}
```

### 5. 完成一轮游戏
点击一个杯子后，应该看到：
```
🔵 准备结束轮次 {currentRoundId: 33, success: true}
🔵 发送结束轮次请求 {roundId: 33, success: true, ...}
✅ 轮次记录结束成功 {ok: true}
```

### 6. 查看统计
- 点击首页的 "📊 每日报表"
- 或访问 `http://localhost:8080/reports/daily/v2`
- 应该能看到刚才的训练记录

---

## ❓ 还是没有数据？

如果按照以上步骤检查后**仍然没有数据**，请提供以下信息：

1. **玩的是哪个游戏？**
   - 游戏名称
   - 游戏URL

2. **Console中的日志**
   - 截图或复制所有日志

3. **Network标签中的请求**
   - 是否有 `/api/session/round/start` 请求？
   - 请求状态码是什么？
   - 响应内容是什么？

4. **数据库查询结果**
   ```bash
   sqlite3 data/leo_concentration.db "SELECT COUNT(*) FROM session_rounds;"
   ```

有了这些信息，我可以帮你精确定位问题！

---

## 💡 提示

- ✅ 确保玩的是已集成的游戏（眼疾手快、数字迷踪、火眼金睛）
- ✅ 确保完成了至少一轮游戏
- ✅ 确保打开了开发者工具的Console标签
- ✅ 确保硬刷新了页面（Ctrl+F5）
- ✅ 确保服务器正在运行（访问首页正常）

现在就去试试，看看Console里有什么日志吧！🔍
