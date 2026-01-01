package main

import (
	"flag"
	"fmt"
	"log"
	"nofx/store"
	"os"
	"path/filepath"
	"time"
)

func main() {
	var dbPath string
	var dryRun bool

	flag.StringVar(&dbPath, "db", "./data/data.db", "数据库文件路径")
	flag.BoolVar(&dryRun, "dry-run", false, "只检查不修复（预览模式）")
	flag.Parse()

	// 确保数据库文件存在
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		log.Fatalf("❌ 无效的数据库路径: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Fatalf("❌ 数据库文件不存在: %s", absPath)
	}

	fmt.Printf("📂 数据库路径: %s\n", absPath)

	// 打开数据库
	s, err := store.New(absPath)
	if err != nil {
		log.Fatalf("❌ 无法打开数据库: %v", err)
	}
	defer s.Close()

	db := s.DB()

	fmt.Println("\n🔍 检查需要修复的订单...")

	// 1. 修复缺少 filled_at 的 FILLED 订单（使用 updated_at 或 created_at）
	var needFixFilledAt int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM trader_orders
		WHERE status = 'FILLED' AND (filled_at IS NULL OR filled_at = '')
	`).Scan(&needFixFilledAt)
	if err != nil {
		log.Fatalf("❌ 查询失败: %v", err)
	}

	fmt.Printf("   📋 缺少成交时间的订单: %d 条\n", needFixFilledAt)

	// 2. 修复 avg_fill_price = 0 的 FILLED 订单（使用 price 字段）
	var needFixAvgPrice int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM trader_orders
		WHERE status = 'FILLED' AND (avg_fill_price = 0 OR avg_fill_price IS NULL) AND price > 0
	`).Scan(&needFixAvgPrice)
	if err != nil {
		log.Fatalf("❌ 查询失败: %v", err)
	}

	fmt.Printf("   💰 成交价为0的订单: %d 条\n", needFixAvgPrice)

	if needFixFilledAt == 0 && needFixAvgPrice == 0 {
		fmt.Println("\n✅ 没有需要修复的订单！")
		return
	}

	if dryRun {
		fmt.Println("\n⚠️  预览模式（--dry-run），不会修改数据")
		fmt.Println("   运行 'go run ./scripts/fix_order_data' 来执行实际修复")
		return
	}

	fmt.Println("\n🔧 开始修复...")

	// 修复缺少 filled_at 的订单
	if needFixFilledAt > 0 {
		result, err := db.Exec(`
			UPDATE trader_orders
			SET filled_at = COALESCE(updated_at, created_at)
			WHERE status = 'FILLED' AND (filled_at IS NULL OR filled_at = '')
		`)
		if err != nil {
			log.Fatalf("❌ 修复成交时间失败: %v", err)
		}
		rows, _ := result.RowsAffected()
		fmt.Printf("   ✅ 修复了 %d 条订单的成交时间\n", rows)
	}

	// 修复 avg_fill_price = 0 的订单
	if needFixAvgPrice > 0 {
		result, err := db.Exec(`
			UPDATE trader_orders
			SET avg_fill_price = price,
				filled_quantity = quantity
			WHERE status = 'FILLED'
			  AND (avg_fill_price = 0 OR avg_fill_price IS NULL)
			  AND price > 0
		`)
		if err != nil {
			log.Fatalf("❌ 修复成交价失败: %v", err)
		}
		rows, _ := result.RowsAffected()
		fmt.Printf("   ✅ 修复了 %d 条订单的成交价\n", rows)
	}

	// 验证修复结果
	fmt.Println("\n🔍 验证修复结果...")
	time.Sleep(100 * time.Millisecond)

	var stillMissingFilledAt int
	db.QueryRow(`
		SELECT COUNT(*)
		FROM trader_orders
		WHERE status = 'FILLED' AND (filled_at IS NULL OR filled_at = '')
	`).Scan(&stillMissingFilledAt)

	var stillMissingAvgPrice int
	db.QueryRow(`
		SELECT COUNT(*)
		FROM trader_orders
		WHERE status = 'FILLED' AND (avg_fill_price = 0 OR avg_fill_price IS NULL)
	`).Scan(&stillMissingAvgPrice)

	fmt.Printf("   📋 仍缺少成交时间: %d 条\n", stillMissingFilledAt)
	fmt.Printf("   💰 仍缺少成交价: %d 条\n", stillMissingAvgPrice)

	if stillMissingFilledAt == 0 && stillMissingAvgPrice == 0 {
		fmt.Println("\n✅ 修复完成！所有订单数据已完整")
		fmt.Println("\n💡 现在刷新图表页面，应该能看到 B/S 标记了")
	} else {
		fmt.Println("\n⚠️  仍有部分订单无法修复，可能需要手动检查")
	}
}
