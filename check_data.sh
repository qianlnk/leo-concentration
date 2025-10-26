#!/bin/bash

echo "════════════════════════════════════════════════"
echo "数据记录状态检查"
echo "════════════════════════════════════════════════"
echo ""

echo "📊 最近10个会话的轮次记录："
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
sqlite3 data/leo_concentration.db << 'EOF'
.mode column
.headers on
SELECT
  s.id as '会话ID',
  t.name as '游戏',
  COUNT(sr.id) as '轮次数',
  CASE
    WHEN COUNT(sr.id) > 0 THEN '✓ 有数据'
    ELSE '✗ 无数据'
  END as '状态'
FROM sessions s
LEFT JOIN trainings t ON s.training_id = t.id
LEFT JOIN session_rounds sr ON s.id = sr.session_id
WHERE s.id >= (SELECT MAX(id) - 9 FROM sessions)
GROUP BY s.id
ORDER BY s.id DESC;
EOF

echo ""
echo "📈 已集成数据记录的游戏："
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  ✅ 眼疾手快 (ID: 11)"
echo "  ✅ 数字迷踪 (ID: 6)"
echo "  ✅ 火眼金睛 (ID: 10)"
echo ""

echo "⏳ 待集成的游戏："
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
sqlite3 data/leo_concentration.db "SELECT '  ❌ ' || name || ' (ID: ' || id || ')' FROM trainings WHERE id NOT IN (6, 10, 11) AND slug IS NOT NULL ORDER BY id;"
echo ""

echo "💡 如何查看统计数据："
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  1. 首页点击 '📊 每日报表'"
echo "  2. 首页训练卡片点击 '📈 查看统计'"
echo "  3. 直接访问:"
echo "     http://localhost:8080/reports/daily/v2"
echo "     http://localhost:8080/training/stats/6  (数字迷踪)"
echo "     http://localhost:8080/training/stats/10 (火眼金睛)"
echo "     http://localhost:8080/training/stats/11 (眼疾手快)"
echo ""

echo "🔍 总轮次统计："
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
TOTAL_ROUNDS=$(sqlite3 data/leo_concentration.db "SELECT COUNT(*) FROM session_rounds;")
echo "  总共记录了 $TOTAL_ROUNDS 轮游戏数据"
echo ""

echo "════════════════════════════════════════════════"
