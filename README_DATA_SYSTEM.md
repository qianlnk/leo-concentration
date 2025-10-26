# 数据记录与可视化系统 - 快速开始

## 🚀 快速演示

运行完整功能演示：
```bash
./demo.sh
```

这将自动：
1. ✅ 创建3个游戏的测试数据
2. ✅ 测试所有API接口
3. ✅ 显示可用的报表页面链接
4. ✅ 验证数据库完整性

## 📊 访问报表

### 1. 增强版每日报表
```
http://localhost:8080/reports/daily/v2
```
功能：
- 📅 选择任意日期查看数据
- 📊 4个汇总卡片（时长、次数、轮次、准确率）
- 📝 详细会话列表（包含难度级别、成功率等）

### 2. 训练项目详情页
```
http://localhost:8080/training/stats/6   # 数字迷踪
http://localhost:8080/training/stats/10  # 火眼金睛
http://localhost:8080/training/stats/11  # 眼疾手快
```
功能：
- 📈 完成时间趋势图（Chart.js）
- 🎯 准确率变化曲线
- 💯 得分趋势柱状图
- 🔽 难度级别和时间范围筛选

## 🎮 已集成数据记录的游戏

| 游戏 | Training ID | 路由 | 状态 |
|------|------------|------|------|
| 眼疾手快 | 11 | /play/cup-ball | ✅ 完成 |
| 数字迷踪 | 6 | /play/number-trace | ✅ 完成 |
| 火眼金睛 | 10 | /play/eagle-eye | ✅ 完成 |
| 记忆卡片 | 3 | /play/memory-cards | ⏳ 待集成 |
| 唐诗迷踪 | 8 | /play/poem-trace | ⏳ 待集成 |

## 🔧 API接口

### 1. 开始轮次
```bash
POST /api/session/round/start
参数: session_id, round_number
返回: {"ok":true,"round_id":123}
```

### 2. 结束轮次
```bash
POST /api/session/round/finish
参数: round_id, success, score, accuracy, metadata
返回: {"ok":true}
```

### 3. 每日统计
```bash
GET /api/stats/daily?date=2025-10-21
返回: {date, total_duration, total_sessions, sessions[]}
```

### 4. 训练统计
```bash
GET /api/stats/training/:id?level=easy&days=30
返回: {training_id, training_name, level, days, rounds[]}
```

## 💾 数据库结构

### sessions 表（扩展）
```sql
level TEXT                          -- 难度级别
status TEXT DEFAULT 'in_progress'   -- 状态
completed_rounds INTEGER            -- 完成轮数
total_rounds INTEGER                -- 总轮数
```

### session_rounds 表（新）
```sql
id                 INTEGER PRIMARY KEY
session_id         INTEGER (外键)
round_number       INTEGER
started_at         DATETIME
ended_at           DATETIME
duration_seconds   INTEGER (自动计算)
success            BOOLEAN
score              INTEGER
accuracy           REAL
metadata           TEXT (JSON格式)
```

## 📖 完整文档

### 主要文档
- [FINAL_SUMMARY.md](FINAL_SUMMARY.md) - **完整总结文档** ⭐
- [DATA_RECORDING_IMPLEMENTATION.md](DATA_RECORDING_IMPLEMENTATION.md) - 实现细节
- [REPORTS_DESIGN.md](REPORTS_DESIGN.md) - 报表设计

### 代码模板
参考 `web/templates/play_cup_ball.html` 查看完整的数据记录集成示例。

## 🎨 关键代码片段

### 前端：开始轮次
```javascript
async function startRound() {
  roundNumber++;
  roundStartTime = Date.now();

  const form = new URLSearchParams();
  form.set('session_id', sessionId);
  form.set('round_number', roundNumber);

  const response = await fetch('/api/session/round/start', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: form
  });

  const data = await response.json();
  currentRoundId = data.round_id;
}
```

### 前端：结束轮次
```javascript
async function finishRound(success, ...) {
  const metadata = JSON.stringify({
    // 游戏特定数据
  });

  const form = new URLSearchParams();
  form.set('round_id', currentRoundId);
  form.set('success', success ? '1' : '0');
  form.set('score', score);
  form.set('accuracy', accuracy);
  form.set('metadata', metadata);

  await fetch('/api/session/round/finish', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: form
  });
}
```

## 🧪 测试

### 手动测试
```bash
# 创建会话
curl -X POST 'http://localhost:8080/session/start' -d 'training_id=6'

# 开始轮次
curl -X POST 'http://localhost:8080/api/session/round/start' \
  -d "session_id=112&round_number=1"

# 结束轮次
curl -X POST 'http://localhost:8080/api/session/round/finish' \
  -d "round_id=13&success=1&score=100&accuracy=100&metadata={}"

# 查询数据
curl 'http://localhost:8080/api/stats/daily?date=2025-10-21'
```

### 查看数据库
```bash
# 查询所有轮次
sqlite3 data/leo_concentration.db \
  "SELECT * FROM session_rounds ORDER BY id DESC LIMIT 10;"

# 查询统计
sqlite3 data/leo_concentration.db \
  "SELECT t.name, COUNT(sr.id), AVG(sr.accuracy)
   FROM session_rounds sr
   JOIN sessions s ON sr.session_id = s.id
   JOIN trainings t ON s.training_id = t.id
   GROUP BY s.training_id;"
```

## 🎯 为新游戏添加数据记录

### 步骤1: 添加变量
```javascript
let currentRoundId = 0;
let roundStartTime = 0;
let roundNumber = 0;
```

### 步骤2: 在游戏开始时调用
```javascript
async function startGame() {
  // ... 原有代码 ...

  // 开始轮次记录
  roundNumber++;
  const form = new URLSearchParams();
  form.set('session_id', sessionId);
  form.set('round_number', roundNumber);

  const response = await fetch('/api/session/round/start', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: form
  });

  const data = await response.json();
  currentRoundId = data.round_id;
}
```

### 步骤3: 在游戏结束时调用
```javascript
async function finishGame() {
  // 构建元数据
  const metadata = JSON.stringify({
    // 游戏特定数据
    level: level,
    score: finalScore,
    // ... 其他数据
  });

  // 记录轮次完成
  const form = new URLSearchParams();
  form.set('round_id', currentRoundId);
  form.set('success', success ? '1' : '0');
  form.set('score', finalScore);
  form.set('accuracy', accuracy);
  form.set('metadata', metadata);

  await fetch('/api/session/round/finish', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: form
  });
}
```

## 📊 数据洞察示例

### 进步趋势
数字迷踪游戏完成时间变化：
```
轮次1: 18秒 → 轮次5: 10秒 (进步44%)
准确率: 84% → 100% (提升16%)
```

### 难度对比
火眼金睛不同难度表现：
```
简单(30项): 15秒, 75%准确率
中等(50项): 28秒, 66%准确率
困难(80项): 45秒, 60%准确率
```

## 🎨 UI特点

- 🎨 渐变卡片设计
- 📱 响应式布局
- 🎯 难度级别徽章
- 📈 Chart.js交互式图表
- 🔽 灵活的筛选器

## 🏆 技术亮点

1. **完整的数据追踪** - 从单次点击到整体会话
2. **灵活的元数据** - JSON格式存储游戏特定数据
3. **自动计算时长** - 后端自动计算duration，防止篡改
4. **专业可视化** - Chart.js图表，趋势清晰可见
5. **易于扩展** - 清晰的代码结构，模块化设计

## 🔮 未来计划

- [ ] 集成更多游戏（记忆卡片、唐诗迷踪等）
- [ ] 每周/每月报表
- [ ] AI分析和个性化建议
- [ ] 数据导出功能（CSV/JSON/PDF）
- [ ] 多用户支持

## 💬 问题反馈

如有问题或建议，请查看：
- [FINAL_SUMMARY.md](FINAL_SUMMARY.md) - 完整功能说明
- [DATA_RECORDING_IMPLEMENTATION.md](DATA_RECORDING_IMPLEMENTATION.md) - 技术细节

---

**版本**: v1.0
**更新时间**: 2025-10-21
**状态**: ✅ 生产就绪
