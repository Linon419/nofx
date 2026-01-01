package main

import (
	"flag"
	"fmt"
	"log"
	"nofx/store"
	"os"
	"path/filepath"
)

func main() {
	var dbPath string
	var dryRun bool

	flag.StringVar(&dbPath, "db", "./data/data.db", "数据库文件路径")
	flag.BoolVar(&dryRun, "dry-run", false, "只检查不删除（预览模式）")
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

	orderStore := s.Order()

	// 1. 检查重复订单数量
	fmt.Println("\n🔍 检查重复数据...")
	dupOrders, err := orderStore.GetDuplicateOrdersCount()
	if err != nil {
		log.Fatalf("❌ 检查重复订单失败: %v", err)
	}
	fmt.Printf("  📋 重复订单: %d 条\n", dupOrders)

	dupFills, err := orderStore.GetDuplicateFillsCount()
	if err != nil {
		log.Fatalf("❌ 检查重复成交失败: %v", err)
	}
	fmt.Printf("  📊 重复成交: %d 条\n", dupFills)

	if dupOrders == 0 && dupFills == 0 {
		fmt.Println("\n✅ 数据库没有重复记录，无需清理")
		return
	}

	if dryRun {
		fmt.Println("\n⚠️  预览模式（--dry-run），不会删除数据")
		fmt.Println("   运行 'go run ./scripts/cleanup_duplicates' 来执行实际清理")
		return
	}

	// 2. 清理重复订单
	if dupOrders > 0 {
		fmt.Println("\n🧹 清理重复订单...")
		deleted, err := orderStore.CleanupDuplicateOrders()
		if err != nil {
			log.Fatalf("❌ 清理失败: %v", err)
		}
		fmt.Printf("  ✅ 删除了 %d 条重复订单\n", deleted)
	}

	// 3. 清理重复成交
	if dupFills > 0 {
		fmt.Println("\n🧹 清理重复成交...")
		deleted, err := orderStore.CleanupDuplicateFills()
		if err != nil {
			log.Fatalf("❌ 清理失败: %v", err)
		}
		fmt.Printf("  ✅ 删除了 %d 条重复成交\n", deleted)
	}

	// 4. 验证清理结果
	fmt.Println("\n🔍 验证清理结果...")
	dupOrdersAfter, _ := orderStore.GetDuplicateOrdersCount()
	dupFillsAfter, _ := orderStore.GetDuplicateFillsCount()
	fmt.Printf("  📋 剩余重复订单: %d 条\n", dupOrdersAfter)
	fmt.Printf("  📊 剩余重复成交: %d 条\n", dupFillsAfter)

	if dupOrdersAfter == 0 && dupFillsAfter == 0 {
		fmt.Println("\n✅ 清理完成！数据库已去重")
	} else {
		fmt.Println("\n⚠️  仍有重复数据，可能需要手动检查")
	}
}
