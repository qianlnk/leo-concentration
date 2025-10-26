#!/usr/bin/env python3
"""
生成专注力训练系统的测试数据
为每个训练项目生成5-10轮真实的训练数据
"""

import requests
import random
import time
import json
from datetime import datetime, timedelta

BASE_URL = "http://localhost:8080"

# 训练项目配置
TRAININGS = {
    6: {  # 数字迷踪
        "name": "数字迷踪",
        "rounds": random.randint(7, 10),
        "duration_range": (10, 30),  # 每轮10-30秒
        "grids": ["3x3", "4x4", "5x5"],
        "metadata_fn": lambda grid, duration, success: {
            "grid_size": grid,
            "total_cells": int(grid[0]) ** 2,
            "time_seconds": duration,
            "success_count": int(grid[0]) ** 2 if success else random.randint(5, int(grid[0]) ** 2 - 1),
            "error_count": 0 if success else random.randint(1, 4)
        }
    },
    10: {  # 火眼金睛
        "name": "火眼金睛",
        "rounds": random.randint(5, 8),
        "duration_range": (15, 40),
        "levels": ["easy", "medium", "hard"],
        "metadata_fn": lambda level, duration, success: {
            "level": level,
            "total_items": {"easy": 20, "medium": 30, "hard": 40}[level],
            "target_count": random.randint(5, 10),
            "target_name": random.choice(["苹果", "香蕉", "西瓜", "草莓"]),
            "target_emoji": random.choice(["🍎", "🍌", "🍉", "🍓"]),
            "time_seconds": duration,
            "success_count": 1 if success else 0,
            "error_count": 0 if success else random.randint(1, 3)
        }
    },
    11: {  # 眼疾手快
        "name": "眼疾手快",
        "rounds": random.randint(6, 9),
        "duration_range": (8, 20),
        "levels": ["easy", "medium", "hard"],
        "metadata_fn": lambda level, duration, success: {
            "cups_count": {"easy": 3, "medium": 4, "hard": 5}[level],
            "swaps_count": {"easy": 5, "medium": 8, "hard": 12}[level],
            "guessed_position": random.randint(0, 4),
            "actual_position": random.randint(0, 4),
            "level": level
        }
    },
    3: {  # 记忆卡片
        "name": "记忆卡片",
        "rounds": random.randint(5, 7),
        "duration_range": (20, 50),
        "pairs": [6, 8, 10, 12],
        "metadata_fn": lambda pairs, duration, success: {
            "pairs_count": pairs,
            "total_cards": pairs * 2,
            "time_seconds": duration,
            "moves_count": random.randint(pairs + 2, pairs * 3),
            "perfect_match": success
        }
    },
    8: {  # 唐诗迷踪
        "name": "唐诗迷踪",
        "rounds": random.randint(5, 8),
        "duration_range": (15, 35),
        "poems": ["静夜思", "春晓", "登鹳雀楼", "悯农"],
        "metadata_fn": lambda poem, duration, success: {
            "poem_name": poem,
            "characters_count": random.randint(20, 28),
            "time_seconds": duration,
            "success_count": random.randint(18, 28) if success else random.randint(10, 18),
            "error_count": 0 if success else random.randint(1, 5)
        }
    }
}

def create_session(training_id):
    """创建训练会话"""
    response = requests.post(f"{BASE_URL}/session/start", data={
        "training_id": training_id
    }, allow_redirects=False)

    # 从重定向URL中提取session_id
    if response.status_code in (302, 303):
        location = response.headers.get('Location', '')
        if 'id=' in location:
            session_id = location.split('id=')[1].split('&')[0]
            return int(session_id)
    return None

def start_round(session_id, round_number):
    """开始一轮训练"""
    response = requests.post(f"{BASE_URL}/api/session/round/start", data={
        "session_id": session_id,
        "round_number": round_number
    })

    if response.status_code == 200:
        data = response.json()
        return data.get('round_id')
    return None

def finish_round(round_id, duration, success, score, accuracy, metadata):
    """完成一轮训练"""
    response = requests.post(f"{BASE_URL}/api/session/round/finish", data={
        "round_id": round_id,
        "duration_seconds": duration,
        "success": "1" if success else "0",
        "score": score,
        "accuracy": accuracy,
        "metadata": json.dumps(metadata)
    })

    return response.status_code == 200

def generate_data_for_training(training_id, config):
    """为一个训练项目生成数据"""
    print(f"\n🎯 开始生成 {config['name']} 的数据...")

    # 创建会话
    session_id = create_session(training_id)
    if not session_id:
        print(f"❌ 创建会话失败")
        return

    print(f"✅ 会话创建成功 (session_id={session_id})")

    # 生成多轮数据
    rounds_count = config['rounds']
    success_count = 0

    for round_num in range(1, rounds_count + 1):
        # 开始轮次
        round_id = start_round(session_id, round_num)
        if not round_id:
            print(f"❌ 第{round_num}轮开始失败")
            continue

        # 模拟游戏进行时间
        time.sleep(0.1)  # 短暂延迟模拟真实游戏

        # 生成随机数据
        duration = random.randint(*config['duration_range'])
        success = random.random() > 0.3  # 70%成功率
        if success:
            success_count += 1

        # 根据成功与否计算分数和准确率
        if success:
            score = random.randint(90, 100)
            accuracy = random.randint(95, 100)
        else:
            score = random.randint(50, 85)
            accuracy = random.randint(60, 90)

        # 生成metadata
        if training_id == 6:  # 数字迷踪
            grid = random.choice(config['grids'])
            metadata = config['metadata_fn'](grid, duration, success)
        elif training_id in [10, 11]:  # 火眼金睛、眼疾手快
            level = random.choice(config['levels'])
            metadata = config['metadata_fn'](level, duration, success)
        elif training_id == 3:  # 记忆卡片
            pairs = random.choice(config['pairs'])
            metadata = config['metadata_fn'](pairs, duration, success)
        elif training_id == 8:  # 唐诗迷踪
            poem = random.choice(config['poems'])
            metadata = config['metadata_fn'](poem, duration, success)
        else:
            metadata = {}

        # 完成轮次
        if finish_round(round_id, duration, success, score, accuracy, metadata):
            status = "✅" if success else "❌"
            print(f"  {status} 第{round_num}轮: {duration}秒, 得分{score}, 准确率{accuracy}%")
        else:
            print(f"  ❌ 第{round_num}轮记录失败")

    print(f"✅ {config['name']} 完成: {success_count}/{rounds_count} 轮成功")

def main():
    print("=" * 60)
    print("🎮 专注力训练系统 - 测试数据生成")
    print("=" * 60)

    # 检查服务器是否运行
    try:
        response = requests.get(BASE_URL, timeout=2)
        if response.status_code != 200:
            print("❌ 服务器未运行，请先启动应用")
            return
    except Exception as e:
        print(f"❌ 无法连接到服务器: {e}")
        return

    print(f"✅ 服务器连接正常")

    # 为每个训练项目生成数据
    for training_id, config in TRAININGS.items():
        try:
            generate_data_for_training(training_id, config)
            time.sleep(0.5)  # 各训练之间短暂延迟
        except Exception as e:
            print(f"❌ {config['name']} 生成失败: {e}")

    print("\n" + "=" * 60)
    print("✅ 所有数据生成完成！")
    print("=" * 60)
    print("\n📊 现在可以访问以下页面查看数据：")
    print(f"  - 首页: {BASE_URL}/")
    print(f"  - 每日报表: {BASE_URL}/reports/daily/v2")
    print(f"  - 数字迷踪统计: {BASE_URL}/training/stats/6")
    print(f"  - 火眼金睛统计: {BASE_URL}/training/stats/10")
    print(f"  - 眼疾手快统计: {BASE_URL}/training/stats/11")

if __name__ == "__main__":
    main()
