# 数据记录与报表系统实现文档

## 一、实现概述

已完成对"眼疾手快"游戏的完整数据记录功能集成，包括：
- ✅ 轮次级别的详细记录（每轮的耗时、成功/失败、准确率、元数据）
- ✅ 会话级别的汇总数据（难度级别、完成轮数、总轮数、状态）
- ✅ 后端API接口（创建轮次、结束轮次、每日统计、训练统计）
- ✅ 前端JavaScript集成（自动记录游戏数据）
- ✅ 增强版每日报表页面

## 二、数据库结构

### 2.1 扩展的 sessions 表

新增字段：
```sql
level TEXT                    -- 难度级别 (easy/medium/hard)
status TEXT DEFAULT 'in_progress'  -- 状态 (in_progress/completed)
completed_rounds INTEGER DEFAULT 0  -- 完成的轮数
total_rounds INTEGER DEFAULT 0      -- 总轮数
```

### 2.2 新增的 session_rounds 表

记录每一轮的详细数据：
```sql
CREATE TABLE session_rounds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,          -- 关联的会话ID
    round_number INTEGER NOT NULL,        -- 轮次编号
    started_at DATETIME NOT NULL,         -- 开始时间
    ended_at DATETIME,                    -- 结束时间
    duration_seconds INTEGER,             -- 耗时（秒）
    success BOOLEAN DEFAULT 0,            -- 成功/失败
    score INTEGER DEFAULT 0,              -- 得分
    accuracy REAL DEFAULT 0,              -- 准确率
    metadata TEXT,                        -- 游戏特定数据（JSON格式）
    FOREIGN KEY(session_id) REFERENCES sessions(id)
);
```

### 2.3 metadata JSON 示例（眼疾手快游戏）

```json
{
  "cups_count": 3,
  "swaps_count": 4,
  "guessed_position": 0,
  "actual_position": 0,
  "level": "easy"
}
```

## 三、API接口

### 3.1 POST /api/session/round/start

**功能**：开始新一轮训练

**请求参数**：
- `session_id`: 会话ID
- `round_number`: 轮次编号

**响应**：
```json
{
  "ok": true,
  "round_id": 123
}
```

### 3.2 POST /api/session/round/finish

**功能**：结束一轮训练

**请求参数**：
- `round_id`: 轮次ID
- `success`: 成功/失败 (1/0)
- `score`: 得分
- `accuracy`: 准确率（0-100）
- `metadata`: JSON格式的游戏特定数据

**响应**：
```json
{
  "ok": true
}
```

### 3.3 GET /api/stats/daily?date=YYYY-MM-DD

**功能**：获取某天的训练统计

**响应**：
```json
{
  "date": "2025-10-21",
  "total_duration": 2482,
  "total_sessions": 25,
  "sessions": [
    {
      "SessionID": 107,
      "TrainingName": "唐诗迷踪",
      "Level": "easy",
      "StartedAt": "2025-10-21T02:23:58.769273+08:00",
      "Duration": 0,
      "CompletedRounds": 2,
      "TotalRounds": 5,
      "SuccessCount": 1,
      "ErrorCount": 1,
      "Accuracy": 50
    }
  ]
}
```

### 3.4 GET /api/stats/training/:id?level=easy&days=30

**功能**：获取某个训练项目的历史轮次数据

**参数**：
- `id`: 训练项目ID
- `level`: 难度级别（可选）
- `days`: 查询天数（默认30）

**响应**：
```json
{
  "training_id": 8,
  "training_name": "唐诗迷踪",
  "level": "easy",
  "days": 7,
  "rounds": [
    {
      "round_number": 1,
      "started_at": "2025-10-21T02:23:58.786461+08:00",
      "duration": 1,
      "success": true,
      "score": 100,
      "accuracy": 100,
      "metadata": "{\"cups_count\":3,\"swaps_count\":4,...}"
    }
  ]
}
```

### 3.5 POST /api/session/update（扩展）

新增参数支持：
- `level`: 难度级别
- `status`: 会话状态
- `completed_rounds`: 完成轮数
- `total_rounds`: 总轮数

## 四、前端集成（眼疾手快游戏）

### 4.1 核心变量

```javascript
let currentRoundId = 0;      // 当前轮次的数据库ID
let roundStartTime = 0;      // 轮次开始时间戳
```

### 4.2 开始轮次时记录

```javascript
async function startRound() {
  roundStartTime = Date.now();

  // 调用API创建轮次记录
  const form = new URLSearchParams();
  form.set('session_id', sessionId);
  form.set('round_number', currentRound);
  const response = await fetch('/api/session/round/start', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: form
  });
  const data = await response.json();
  currentRoundId = data.round_id;
}
```

### 4.3 结束轮次时记录

```javascript
async function finishRound(success, guessedPosition, actualPosition) {
  const config = levelConfigs[level];
  const accuracy = success ? 100 : 0;

  // 构建metadata JSON
  const metadata = JSON.stringify({
    cups_count: config.cups,
    swaps_count: config.swaps,
    guessed_position: guessedPosition,
    actual_position: actualPosition,
    level: level
  });

  const form = new URLSearchParams();
  form.set('round_id', currentRoundId);
  form.set('success', success ? '1' : '0');
  form.set('score', success ? '100' : '0');
  form.set('accuracy', accuracy);
  form.set('metadata', metadata);

  await fetch('/api/session/round/finish', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: form
  });
}
```

### 4.4 游戏结束时更新会话

```javascript
async function finishGame() {
  const form = new URLSearchParams();
  form.set('id', sessionId);
  form.set('success', successCount);
  form.set('errors', errorCount);
  form.set('notes', note);
  form.set('append', '1');
  form.set('level', level);
  form.set('completed_rounds', totalRounds);
  form.set('total_rounds', totalRounds);
  form.set('status', 'completed');

  await fetch('/api/session/update', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: form
  });
}
```

## 五、报表页面

### 5.1 原有报表页面

路径：`/reports/daily`

功能：显示最近30天的训练时长和次数汇总，带简单的折线图。

### 5.2 增强版报表页面

路径：`/reports/daily/v2`

功能：
- 📊 日期选择器，查看任意日期的数据
- 📈 四个汇总卡片：训练时长、训练次数、总轮次、平均准确率
- 📝 详细的会话列表，显示：
  - 训练项目名称
  - 难度级别徽章
  - 开始时间
  - 训练时长
  - 完成轮数
  - 成功/错误次数
  - 准确率
- 🎨 美观的UI设计，使用渐变卡片和徽章

## 六、完整的数据流

```
1. 用户开始游戏
   ↓
2. POST /session/start 创建会话
   ↓
3. 开始第一轮
   ↓
4. POST /api/session/round/start 创建轮次记录
   ↓
5. 用户完成游戏操作
   ↓
6. POST /api/session/round/finish 记录轮次结果
   ↓
7. 重复步骤 3-6 直到所有轮次完成
   ↓
8. POST /api/session/update 更新会话状态
   ↓
9. POST /session/finish 结束会话
   ↓
10. GET /reports/daily/v2 查看报表
```

## 七、测试验证

### 7.1 完整流程测试

已通过以下测试：
```bash
# 1. 创建会话
curl -X POST 'http://localhost:8080/session/start' -d 'training_id=8'
# 会话ID: 107

# 2. 开始轮次
curl -X POST 'http://localhost:8080/api/session/round/start' \
  -d "session_id=107&round_number=1"
# 响应: {"ok":true,"round_id":1}

# 3. 结束轮次
curl -X POST 'http://localhost:8080/api/session/round/finish' \
  -d "round_id=1&success=1&score=100&accuracy=100&metadata={...}"
# 响应: {"ok":true}

# 4. 查询统计
curl 'http://localhost:8080/api/stats/daily?date=2025-10-21'
# 返回完整的每日统计数据

curl 'http://localhost:8080/api/stats/training/8?level=easy&days=7'
# 返回训练项目的轮次历史
```

### 7.2 数据库验证

```sql
-- 查询会话数据
SELECT id, training_id, level, status, completed_rounds, total_rounds
FROM sessions WHERE id=107;

-- 查询轮次数据
SELECT id, round_number, success, score, accuracy, metadata
FROM session_rounds WHERE session_id=107;
```

结果：
```
107|8|easy|completed|2|5|1|1

1|1|1|100|100.0|{"cups_count":3,"swaps_count":4,...}
2|2|0|0|0.0|{"cups_count":3,"swaps_count":4,...}
```

## 八、后续任务

### 8.1 其他游戏集成

将数据记录功能集成到其他游戏：
- [ ] 数字迷踪
- [ ] 记忆卡片
- [ ] 火眼金睛
- [ ] 颜色对对碰
- [ ] 唐诗迷踪
- [ ] 平衡超人
- [ ] 全身唤醒

### 8.2 报表功能增强

- [ ] 添加Chart.js图表库
- [ ] 实现训练曲线图（按时间展示准确率变化）
- [ ] 实现难度进阶跟踪
- [ ] 每周报表页面
- [ ] 每月报表页面
- [ ] 训练项目详情页（展示单个项目的历史数据）

### 8.3 数据导出

- [ ] CSV导出功能
- [ ] JSON导出功能
- [ ] 数据备份功能

### 8.4 分析功能

- [ ] 最佳训练时段分析
- [ ] 进步趋势识别
- [ ] 薄弱项目识别
- [ ] 成长里程碑提醒

## 九、技术要点

### 9.1 日期格式处理

SQLite的`date()`函数无法正确解析Go的时间格式，因此使用`substr()`函数：
```sql
WHERE substr(s.started_at, 1, 10) = '2025-10-21'
```

### 9.2 动态SQL构建

使用动态SQL更新，避免覆盖未传递的字段：
```go
query := `UPDATE sessions SET success_count=?, error_count=?, notes=?`
args := []any{success, errors, notes}

if level != "" {
  query += `, level=?`
  args = append(args, level)
}
```

### 9.3 异步数据记录

使用`async/await`确保数据记录不阻塞游戏流程：
```javascript
async function finishRound(success, ...) {
  try {
    await fetch('/api/session/round/finish', {...});
  } catch (err) {
    console.error('Failed to finish round:', err);
    // 记录失败不影响游戏继续
  }
}
```

## 十、文件清单

### 修改的文件
1. `main.go` - 后端逻辑
   - 扩展数据库migration
   - 添加4个新API接口
   - 更新session更新接口

2. `web/templates/play_cup_ball.html` - 眼疾手快游戏
   - 添加轮次记录功能
   - 集成数据记录API调用

### 新增的文件
1. `web/templates/reports_daily_v2.html` - 增强版每日报表页面

2. `REPORTS_DESIGN.md` - 报表系统设计文档

3. `DATA_RECORDING_IMPLEMENTATION.md` - 本文档

## 十一、访问地址

- 眼疾手快游戏：`http://localhost:8080/play/cup-ball?id={session_id}`
- 每日报表（原版）：`http://localhost:8080/reports/daily`
- 每日报表（新版）：`http://localhost:8080/reports/daily/v2`
- 每日统计API：`http://localhost:8080/api/stats/daily?date=YYYY-MM-DD`
- 训练统计API：`http://localhost:8080/api/stats/training/:id?level=easy&days=30`

## 十二、总结

已成功实现：
✅ 完整的数据记录基础架构
✅ 轮次级别的详细数据追踪
✅ 会话级别的汇总信息
✅ RESTful API接口
✅ 前端自动记录集成
✅ 增强版报表页面
✅ 完整的测试验证

该系统为其他训练项目的数据记录提供了完整的参考模板，后续可以快速复制到其他游戏中。
