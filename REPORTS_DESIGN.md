# Leo专注力训练系统 - 数据记录与报表设计

## 一、数据记录需求分析

### 1.1 需要记录的数据

#### 会话级别（Session Level）
- 训练项目
- 开始时间/结束时间
- 总时长
- 难度级别
- 总成功次数/总错误次数
- 完成状态（完成/中途退出）

#### 轮次级别（Round Level）- 每个游戏的每一轮
- 轮次编号
- 开始时间
- 结束时间
- 轮次耗时（秒）
- 成功/失败
- 难度级别
- 特定指标（如：找到物品数、答对题数等）

### 1.2 各游戏特定指标

| 游戏 | 轮次指标 | 会话总计 |
|------|---------|---------|
| 数字迷踪 | 网格大小、完成时间、错误次数 | 平均完成时间、总错误率 |
| 记忆卡片 | 翻牌次数、配对成功次数、用时 | 平均翻牌次数、记忆准确率 |
| 颜色对对碰 | 网格大小、答对数、答错数、用时 | 正确率、平均反应时间 |
| 火眼金睛 | 物品数、找到数、错误点击数、用时 | 观察准确率、平均速度 |
| 眼疾手快 | 杯子数、交换次数、成功/失败、用时 | 成功率、平均难度 |
| 唐诗迷踪 | 诗句长度、完成时间、错误次数 | 正确率、熟练度提升 |
| 平衡超人 | 左脚时长、右脚时长 | 平衡性、耐力提升 |

## 二、数据库设计

### 2.1 扩展 sessions 表

```sql
ALTER TABLE sessions ADD COLUMN level TEXT;              -- 难度级别
ALTER TABLE sessions ADD COLUMN status TEXT;             -- 完成状态
ALTER TABLE sessions ADD COLUMN completed_rounds INTEGER; -- 完成轮数
ALTER TABLE sessions ADD COLUMN total_rounds INTEGER;     -- 总轮数
```

### 2.2 新增 session_rounds 表

```sql
CREATE TABLE session_rounds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    round_number INTEGER NOT NULL,                    -- 轮次编号
    started_at DATETIME NOT NULL,                     -- 开始时间
    ended_at DATETIME,                                -- 结束时间
    duration_seconds INTEGER,                         -- 耗时（秒）
    success BOOLEAN DEFAULT 0,                        -- 成功/失败

    -- 通用指标
    score INTEGER DEFAULT 0,                          -- 得分
    accuracy REAL DEFAULT 0,                          -- 准确率

    -- 特定游戏指标（JSON格式）
    metadata TEXT,                                    -- JSON: 游戏特定数据

    FOREIGN KEY(session_id) REFERENCES sessions(id)
);
```

### 2.3 metadata JSON 示例

```json
{
  "number-trace": {
    "grid_size": 4,
    "clicks": 16,
    "errors": 2
  },
  "memory-cards": {
    "grid_size": 4,
    "flips": 24,
    "pairs_found": 8
  },
  "cup-ball": {
    "cups_count": 3,
    "swaps_count": 4,
    "guessed_position": 2,
    "actual_position": 1
  },
  "eagle-eye": {
    "total_items": 30,
    "target_count": 3,
    "found_count": 3,
    "wrong_clicks": 1
  }
}
```

## 三、报表设计

### 3.1 每日报表（Daily Report）

#### 显示内容
1. **训练总览**
   - 今日训练总时长
   - 完成的训练项目数
   - 总轮次数
   - 整体准确率

2. **各项目明细**
   - 项目名称
   - 难度级别
   - 完成轮数
   - 平均耗时
   - 准确率
   - 趋势（↑提升 ↓下降 →持平）

3. **时间分布图**
   - 横轴：时间段（上午/下午/晚上）
   - 纵轴：训练时长

4. **表现曲线**
   - 横轴：轮次
   - 纵轴：准确率/耗时
   - 展示单个项目的训练曲线

### 3.2 每周报表（Weekly Report）

#### 显示内容
1. **本周总览**
   - 训练天数
   - 总训练时长
   - 日均时长
   - 完成的训练项目

2. **每日趋势图**
   - 横轴：星期一到星期日
   - 纵轴：训练时长
   - 柱状图

3. **项目进步对比**
   - 对比周初和周末的表现
   - 各项目的准确率变化

### 3.3 每月报表（Monthly Report）

#### 显示内容
1. **月度总览**
   - 训练天数/总天数
   - 总训练时长
   - 日均时长
   - 坚持率

2. **项目表现图表**
   - 各项目的训练频率
   - 各难度级别的分布
   - 准确率月度趋势

3. **同级别对比**
   - 按难度级别分组
   - 展示每天的平均分
   - 折线图显示进步趋势

4. **成长亮点**
   - 最大进步项目
   - 最稳定项目
   - 需要加强项目

## 四、图表选择

### 4.1 图表库
使用 **Chart.js** - 轻量级、易用、美观

CDN:
```html
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
```

### 4.2 图表类型

| 数据类型 | 图表类型 | 用途 |
|---------|---------|------|
| 每日训练时长 | 柱状图 | 直观对比 |
| 准确率趋势 | 折线图 | 展示变化 |
| 项目分布 | 饼图 | 占比展示 |
| 轮次表现 | 雷达图 | 多维度对比 |
| 月度进步 | 面积图 | 累积效果 |

## 五、实现计划

### 5.1 数据库迁移
1. 扩展 sessions 表添加新字段
2. 创建 session_rounds 表
3. 迁移历史数据

### 5.2 API 设计

```
POST /api/session/round/start   - 开始新一轮
POST /api/session/round/finish  - 结束一轮
GET  /reports/daily?date=YYYY-MM-DD
GET  /reports/weekly?week=YYYY-WW
GET  /reports/monthly?month=YYYY-MM
GET  /api/stats/training/:id?level=easy&days=30
```

### 5.3 前端改造
1. 各游戏页面添加轮次记录
2. 记录每轮的开始/结束时间
3. 记录游戏特定指标

### 5.4 报表页面
1. 每日报表页面（/reports/daily）
2. 每周报表页面（/reports/weekly）
3. 每月报表页面（/reports/monthly）
4. 项目详情页面（/reports/training/:id）

## 六、数据分析维度

### 6.1 时间维度
- 按时间段：上午/下午/晚上
- 按星期：工作日/周末
- 按月份：季节性变化

### 6.2 难度维度
- 各难度级别的完成率
- 难度提升的时间点
- 最适合的难度级别

### 6.3 项目维度
- 各项目的熟练度
- 项目间的关联性
- 薄弱环节识别

### 6.4 成长维度
- 专注力指数（综合得分）
- 进步速度
- 学习曲线

## 七、家长洞察

### 7.1 自动生成的建议
- "数字迷踪"表现优秀，可以提升难度
- "记忆卡片"准确率下降，建议加强练习
- 上午训练效果最好，建议增加上午时段训练

### 7.2 成长里程碑
- 连续训练7天
- 某项目准确率达到90%
- 完成100个训练轮次
- 所有项目都尝试过

## 八、隐私保护

- 所有数据本地存储
- 不上传到云端
- 可导出数据（JSON/CSV）
- 可清除历史数据

## 九、未来扩展

1. **多用户支持**
   - 添加 children 表
   - 关联训练数据

2. **目标设定**
   - 每日目标
   - 每周目标
   - 自定义目标

3. **奖励系统**
   - 徽章系统
   - 积分系统
   - 成就解锁

4. **数据对比**
   - 同龄对比（匿名）
   - 历史对比
   - 目标对比
