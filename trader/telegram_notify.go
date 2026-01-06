package trader

import (
	"context"
	"fmt"
	"nofx/logger"
	"nofx/notify"
	"nofx/store"
	"strings"
	"time"
)

func sendTelegramTradeNotification(st *store.Store, traderID string, exchangeID string, exchangeType string, action string, symbol string, side string, quantity float64, price float64, fee float64, realizedPnL float64, filledAt time.Time) {
	if st == nil {
		return
	}

	traderInfo, err := st.Trader().GetByID(traderID)
	if err != nil || traderInfo == nil {
		return
	}

	tg, err := st.Telegram().Get(traderInfo.UserID)
	if err != nil || tg == nil || !tg.Enabled {
		return
	}

	actionLower := strings.ToLower(strings.TrimSpace(action))
	isOpen := strings.HasPrefix(actionLower, "open_")
	isClose := strings.HasPrefix(actionLower, "close_")

	if isOpen && !tg.NotifyOnOpen {
		return
	}
	if isClose && !tg.NotifyOnClose {
		return
	}

	notional := price * quantity

	header := fmt.Sprintf("NOFX %s %s", strings.ToUpper(actionLower), symbol)
	lines := []string{
		header,
		fmt.Sprintf("Trader: %s", traderInfo.Name),
		fmt.Sprintf("Exchange: %s", exchangeType),
		fmt.Sprintf("Order: %s %s", strings.ToUpper(side), symbol),
		fmt.Sprintf("Price: %.8f", price),
		fmt.Sprintf("Qty: %.8f", quantity),
		fmt.Sprintf("Notional: %.2f", notional),
	}
	if isClose {
		lines = append(lines, fmt.Sprintf("PnL: %+0.2f", realizedPnL))
	}
	if fee != 0 {
		lines = append(lines, fmt.Sprintf("Fee: %.8f", fee))
	}
	if !filledAt.IsZero() {
		lines = append(lines, fmt.Sprintf("Time: %s", filledAt.Format(time.RFC3339)))
	}

	msg := strings.Join(lines, "\n")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := notify.SendTelegramMessage(ctx, tg.BotToken, tg.ChatID, msg); err != nil {
			logger.Infof(
				"telegram notification failed (trade, trader=%s, trader_id=%s, user_id=%s, exchange=%s, action=%s, symbol=%s): %v",
				traderInfo.Name,
				traderID,
				traderInfo.UserID,
				exchangeType,
				actionLower,
				symbol,
				err,
			)
		}
	}()
}

func sendTelegramErrorNotification(st *store.Store, userID string, traderName string, exchangeType string, action string, symbol string, err error) {
	if st == nil || err == nil {
		return
	}

	tg, tgErr := st.Telegram().Get(userID)
	if tgErr != nil || tg == nil || !tg.Enabled || !tg.NotifyOnError {
		return
	}

	var header string
	if strings.TrimSpace(action) != "" && strings.TrimSpace(symbol) != "" {
		header = fmt.Sprintf("NOFX ERROR %s %s", strings.ToUpper(strings.TrimSpace(action)), strings.TrimSpace(symbol))
	} else {
		header = "NOFX ERROR"
	}

	lines := []string{
		header,
		fmt.Sprintf("Trader: %s", traderName),
		fmt.Sprintf("Exchange: %s", exchangeType),
	}
	if strings.TrimSpace(action) != "" {
		lines = append(lines, fmt.Sprintf("Action: %s", action))
	}
	if strings.TrimSpace(symbol) != "" {
		lines = append(lines, fmt.Sprintf("Symbol: %s", symbol))
	}
	lines = append(lines, fmt.Sprintf("Error: %s", err.Error()))

	msg := strings.Join(lines, "\n")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := notify.SendTelegramMessage(ctx, tg.BotToken, tg.ChatID, msg); err != nil {
			logger.Infof("telegram notification failed (error, user_id=%s, trader=%s): %v", userID, traderName, err)
		}
	}()
}
