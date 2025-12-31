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
	var traderID string

	flag.StringVar(&dbPath, "db", "./data/data.db", "SQLite database file path")
	flag.StringVar(&traderID, "trader", "", "Trader ID (optional)")
	flag.Parse()

	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		log.Fatalf("invalid database path: %v", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Fatalf("database file does not exist: %s", absPath)
	}

	fmt.Printf("Database: %s\n", absPath)

	s, err := store.New(absPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer s.Close()

	orderStore := s.Order()

	if traderID == "" {
		fmt.Println()
		fmt.Println("WARNING: trader_id not provided; use --trader <trader_id> to filter.")
		fmt.Println("Showing aggregated information for all traders.")
	}

	orders, err := orderStore.GetTraderOrders(traderID, 100)
	if err != nil {
		log.Fatalf("failed to get orders: %v", err)
	}

	fmt.Printf("\nFound %d order records\n\n", len(orders))
	if len(orders) == 0 {
		fmt.Println("No orders found.")
		return
	}

	var (
		totalOrders        = len(orders)
		filledOrders       = 0
		withFilledAt       = 0
		withAvgFillPrice   = 0
		withOrderAction    = 0
		missingFilledAt    = 0
		missingAvgPrice    = 0
		missingOrderAction = 0
	)

	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("%-15s %-10s %-12s %-14s %-12s %s\n", "order_id", "status", "action", "avg_price", "filled_at", "issues")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, order := range orders {
		issues := []string{}

		if order.Status == "FILLED" {
			filledOrders++

			if !order.FilledAt.IsZero() {
				withFilledAt++
			} else {
				missingFilledAt++
				issues = append(issues, "missing_filled_at")
			}

			if order.AvgFillPrice > 0 {
				withAvgFillPrice++
			} else {
				missingAvgPrice++
				issues = append(issues, "avg_fill_price=0")
			}
		}

		if order.OrderAction != "" {
			withOrderAction++
		} else {
			missingOrderAction++
			issues = append(issues, "missing_order_action")
		}

		issueStr := "ok"
		if len(issues) > 0 {
			issueStr = ""
			for i, issue := range issues {
				if i > 0 {
					issueStr += ","
				}
				issueStr += issue
			}
		}

		filledAtStr := "N/A"
		if !order.FilledAt.IsZero() {
			filledAtStr = order.FilledAt.Format("01-02 15:04")
		}

		orderID := order.ExchangeOrderID
		if len(orderID) > 15 {
			orderID = orderID[:15]
		}

		fmt.Printf("%-15s %-10s %-12s %-14.2f %-12s %s\n",
			orderID,
			order.Status,
			order.OrderAction,
			order.AvgFillPrice,
			filledAtStr,
			issueStr,
		)
	}

	fmt.Println("--------------------------------------------------------------------------------")

	fmt.Println("\nSummary:")
	fmt.Printf("  total_orders:           %d\n", totalOrders)
	fmt.Printf("  filled_orders:          %d\n", filledOrders)
	fmt.Printf("  filled_with_filled_at:  %d / %d (%.1f%%)\n", withFilledAt, filledOrders, float64(withFilledAt)/float64(max(filledOrders, 1))*100)
	fmt.Printf("  filled_with_avg_price:  %d / %d (%.1f%%)\n", withAvgFillPrice, filledOrders, float64(withAvgFillPrice)/float64(max(filledOrders, 1))*100)
	fmt.Printf("  with_order_action:      %d / %d (%.1f%%)\n", withOrderAction, totalOrders, float64(withOrderAction)/float64(max(totalOrders, 1))*100)

	if missingFilledAt > 0 || missingAvgPrice > 0 || missingOrderAction > 0 {
		fmt.Println("\nIssues:")
		if missingFilledAt > 0 {
			fmt.Printf("  - %d orders missing filled_at\n", missingFilledAt)
		}
		if missingAvgPrice > 0 {
			fmt.Printf("  - %d filled orders with avg_fill_price=0\n", missingAvgPrice)
		}
		if missingOrderAction > 0 {
			fmt.Printf("  - %d orders missing order_action\n", missingOrderAction)
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
