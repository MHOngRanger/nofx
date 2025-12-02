package custom_data

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/adshao/go-binance/v2/futures"
)

// DepthBin represents an aggregated price level
type DepthBin struct {
	PriceBin float64
	ValueUSD float64
	Side     string // "Bid" or "Ask"
}

// GetMarketDepth returns the Market Depth data for a given symbol.
// It fetches depth data, aggregates it into price bins, and returns a summary of support/resistance.
func GetMarketDepth(symbol string) string {
	// Use a public client
	client := futures.NewClient("", "")

	// Fetch Depth (Limit 1000)
	depth, err := client.NewDepthService().
		Symbol(symbol).
		Limit(1000).
		Do(context.Background())

	if err != nil {
		log.Printf("Error fetching depth data for %s: %v", symbol, err)
		return fmt.Sprintf("Error fetching depth data: %v", err)
	}

	// Determine current price (approximate from best bid/ask)
	if len(depth.Bids) == 0 || len(depth.Asks) == 0 {
		return "Insufficient depth data"
	}

	bestBid, _ := strconv.ParseFloat(depth.Bids[0].Price, 64)
	bestAsk, _ := strconv.ParseFloat(depth.Asks[0].Price, 64)
	midPrice := (bestBid + bestAsk) / 2

	// Dynamic Bin Size: ~0.05% of price, rounded to nice number
	rawBinSize := midPrice * 0.0005
	binSize := calculateNiceBinSize(rawBinSize)

	// Aggregate Bids
	var bids []PriceLevel
	for _, b := range depth.Bids {
		bids = append(bids, PriceLevel{Price: b.Price, Quantity: b.Quantity})
	}
	bidBins := aggregateDepth(bids, binSize, "Bid")

	// Aggregate Asks
	var asks []PriceLevel
	for _, a := range depth.Asks {
		asks = append(asks, PriceLevel{Price: a.Price, Quantity: a.Quantity})
	}
	askBins := aggregateDepth(asks, binSize, "Ask")

	// Filter bins within +/- 2% range
	lowerBound := midPrice * 0.98
	upperBound := midPrice * 1.02

	filteredBids := filterAndSortBins(bidBins, lowerBound, upperBound, true) // Sort Bids descending by value
	filteredAsks := filterAndSortBins(askBins, lowerBound, upperBound, true) // Sort Asks descending by value

	// Format Output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Market Depth for %s (Bin Size: %.2f):\n", symbol, binSize))
	sb.WriteString(fmt.Sprintf("Current Price: %.2f\n\n", midPrice))

	sb.WriteString("### Top Support Levels (Bids)\n")
	sb.WriteString("| Price Bin | Value (USD) | Strength |\n")
	sb.WriteString("|---|---|---|\n")
	writeTopBins(&sb, filteredBids, 5)

	sb.WriteString("\n### Top Resistance Levels (Asks)\n")
	sb.WriteString("| Price Bin | Value (USD) | Strength |\n")
	sb.WriteString("|---|---|---|\n")
	writeTopBins(&sb, filteredAsks, 5)

	return sb.String()
}

func calculateNiceBinSize(raw float64) float64 {
	// Simple logic to find a "nice" number (1, 2, 5, 10, 20, 50, 100...)
	pow10 := math.Pow(10, math.Floor(math.Log10(raw)))
	base := raw / pow10

	var niceBase float64
	if base < 2 {
		niceBase = 1
	} else if base < 5 {
		niceBase = 2
	} else {
		niceBase = 5
	}

	return niceBase * pow10
}

// PriceLevel is a local struct to unify Bid and Ask types
type PriceLevel struct {
	Price    string
	Quantity string
}

func aggregateDepth(entries []PriceLevel, binSize float64, side string) map[float64]*DepthBin {
	bins := make(map[float64]*DepthBin)

	for _, entry := range entries {
		price, _ := strconv.ParseFloat(entry.Price, 64)
		qty, _ := strconv.ParseFloat(entry.Quantity, 64)
		value := price * qty

		// Binning: Floor to nearest binSize
		binPrice := math.Floor(price/binSize) * binSize

		if _, exists := bins[binPrice]; !exists {
			bins[binPrice] = &DepthBin{PriceBin: binPrice, ValueUSD: 0, Side: side}
		}
		bins[binPrice].ValueUSD += value
	}
	return bins
}

func filterAndSortBins(binsMap map[float64]*DepthBin, minPrice, maxPrice float64, desc bool) []DepthBin {
	var result []DepthBin
	for _, bin := range binsMap {
		if bin.PriceBin >= minPrice && bin.PriceBin <= maxPrice {
			result = append(result, *bin)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if desc {
			return result[i].ValueUSD > result[j].ValueUSD
		}
		return result[i].ValueUSD < result[j].ValueUSD
	})

	return result
}

func writeTopBins(sb *strings.Builder, bins []DepthBin, limit int) {
	if len(bins) < limit {
		limit = len(bins)
	}

	// Find max value for simple "Strength" visualization
	maxVal := 0.0
	if len(bins) > 0 {
		maxVal = bins[0].ValueUSD
	}

	for i := 0; i < limit; i++ {
		bin := bins[i]
		strength := ""
		if maxVal > 0 {
			stars := int((bin.ValueUSD / maxVal) * 5)
			if stars < 1 {
				stars = 1
			}
			strength = strings.Repeat("★", stars)
		}

		sb.WriteString(fmt.Sprintf("| %.2f | $%.0f | %s |\n",
			bin.PriceBin,
			bin.ValueUSD,
			strength,
		))
	}
}
