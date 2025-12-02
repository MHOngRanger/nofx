package custom_data

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

// LiquidationOrder represents a simplified liquidation order
type LiquidationOrder struct {
	Time      time.Time
	Price     float64
	OrigQty   float64
	ValueUSD  float64
	Side      string // "SELL" (Long Rekt) or "BUY" (Short Rekt)
	TypeLabel string // "Long Liq" or "Short Liq"
}

// GetLiquidationData returns the Liquidation Data for a given symbol.
// It fetches the last 100 force orders, aggregates them, and returns a summary string.
func GetLiquidationData(symbol string) string {
	// Use a public client (no keys needed for ForceOrders)
	client := futures.NewClient("", "")

	// Fetch Force Orders (Limit 100)
	orders, err := client.NewListLiquidationOrdersService().
		Symbol(symbol).
		Limit(100).
		Do(context.Background())

	if err != nil {
		log.Printf("Error fetching liquidation data for %s: %v", symbol, err)
		return fmt.Sprintf("Error fetching liquidation data: %v", err)
	}

	if len(orders) == 0 {
		return fmt.Sprintf("No recent liquidation data found for %s", symbol)
	}

	var liqList []LiquidationOrder
	var totalLongLiqVol, totalShortLiqVol float64
	var longLiqCount, shortLiqCount int

	for _, o := range orders {
		price, _ := strconv.ParseFloat(o.Price, 64)
		origQty, _ := strconv.ParseFloat(o.OrigQuantity, 64)
		timeVal := time.Unix(o.Time/1000, 0)
		valueUSD := price * origQty

		side := string(o.Side)
		typeLabel := ""

		// SELL side means a Long position was liquidated (selling to close)
		// BUY side means a Short position was liquidated (buying to close)
		if side == "SELL" {
			typeLabel = "Long Liq"
			totalLongLiqVol += valueUSD
			longLiqCount++
		} else {
			typeLabel = "Short Liq"
			totalShortLiqVol += valueUSD
			shortLiqCount++
		}

		liqList = append(liqList, LiquidationOrder{
			Time:      timeVal,
			Price:     price,
			OrigQty:   origQty,
			ValueUSD:  valueUSD,
			Side:      side,
			TypeLabel: typeLabel,
		})
	}

	// Sort by Value USD descending to find largest liquidations
	sort.Slice(liqList, func(i, j int) bool {
		return liqList[i].ValueUSD > liqList[j].ValueUSD
	})

	// Format Output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Liquidation Data for %s (Last %d orders):\n", symbol, len(orders)))

	// Summary Stats
	sb.WriteString("### Summary\n")
	sb.WriteString(fmt.Sprintf("- **Long Liquidations**: %d orders, Total $%.2f\n", longLiqCount, totalLongLiqVol))
	sb.WriteString(fmt.Sprintf("- **Short Liquidations**: %d orders, Total $%.2f\n", shortLiqCount, totalShortLiqVol))

	ratio := 0.0
	if totalShortLiqVol > 0 {
		ratio = totalLongLiqVol / totalShortLiqVol
	}
	sb.WriteString(fmt.Sprintf("- **Long/Short Ratio (Vol)**: %.2f\n\n", ratio))

	// Top 3 Largest Liquidations
	sb.WriteString("### Top 3 Largest Liquidations\n")
	sb.WriteString("| Time | Type | Price | Value (USD) |\n")
	sb.WriteString("|---|---|---|---|\n")

	limit := 3
	if len(liqList) < 3 {
		limit = len(liqList)
	}

	for i := 0; i < limit; i++ {
		o := liqList[i]
		sb.WriteString(fmt.Sprintf("| %s | %s | %.2f | $%.2f |\n",
			o.Time.Format("15:04:05"),
			o.TypeLabel,
			o.Price,
			o.ValueUSD,
		))
	}

	return sb.String()
}
