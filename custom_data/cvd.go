package custom_data

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

// CVDData represents a single candle's CVD data
type CVDData struct {
	OpenTime       time.Time
	Close          float64
	Volume         float64
	TakerBuyVolume float64
	Delta          float64
	CVD            float64
}

// GetCumulativeVolumeDelta returns the Cumulative Volume Delta data for a given symbol.
// It fetches the last 100 15m klines, calculates CVD, and returns a formatted string table of the last 5 candles.
func GetCumulativeVolumeDelta(symbol string) string {
	// Use a public client (no keys needed for Klines)
	client := futures.NewClient("", "")

	// Fetch Klines (15m interval, limit 100)
	klines, err := client.NewKlinesService().
		Symbol(symbol).
		Interval("15m").
		Limit(100).
		Do(context.Background())

	if err != nil {
		log.Printf("Error fetching klines for CVD %s: %v", symbol, err)
		return fmt.Sprintf("Error fetching CVD data: %v", err)
	}

	var cvdList []CVDData
	var currentCVD float64 = 0

	for _, k := range klines {
		// Parse necessary fields
		// Binance Kline:
		// [0] Open time
		// [1] Open
		// [2] High
		// [3] Low
		// [4] Close
		// [5] Volume
		// ...
		// [9] Taker buy base asset volume

		closePrice, _ := strconv.ParseFloat(k.Close, 64)
		volume, _ := strconv.ParseFloat(k.Volume, 64)
		takerBuyVolume, _ := strconv.ParseFloat(k.TakerBuyBaseAssetVolume, 64)
		openTime := time.Unix(k.OpenTime/1000, 0)

		// Calculate Delta
		// Taker Sell Volume = Volume - Taker Buy Volume
		// Delta = Taker Buy Volume - Taker Sell Volume
		//       = Taker Buy Volume - (Volume - Taker Buy Volume)
		//       = 2 * Taker Buy Volume - Volume
		delta := 2*takerBuyVolume - volume

		// Calculate CVD (Cumulative Sum)
		currentCVD += delta

		cvdList = append(cvdList, CVDData{
			OpenTime:       openTime,
			Close:          closePrice,
			Volume:         volume,
			TakerBuyVolume: takerBuyVolume,
			Delta:          delta,
			CVD:            currentCVD,
		})
	}

	// Format the output (Last 5 candles)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CVD Data for %s (15m):\n", symbol))
	sb.WriteString("| Time | Close | Delta | CVD |\n")
	sb.WriteString("|---|---|---|---|\n")

	startIdx := len(cvdList) - 5
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(cvdList); i++ {
		data := cvdList[i]
		sb.WriteString(fmt.Sprintf("| %s | %.2f | %.2f | %.2f |\n",
			data.OpenTime.Format("15:04"),
			data.Close,
			data.Delta,
			data.CVD,
		))
	}

	// Add a brief analysis
	last := cvdList[len(cvdList)-1]
	prev := cvdList[len(cvdList)-2]

	trend := "Neutral"
	if last.CVD > prev.CVD {
		trend = "Rising (Buying Pressure)"
	} else if last.CVD < prev.CVD {
		trend = "Falling (Selling Pressure)"
	}

	sb.WriteString(fmt.Sprintf("\nTrend: %s", trend))

	return sb.String()
}
