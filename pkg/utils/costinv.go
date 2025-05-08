// pkg/utils/costinv.go
package utils

import (
	"bytes"
	"fmt"
	"time"
)

// CostInvItem represents one line in your cost/inventory feed.
// Populate this slice from your DB or wherever you track SKU, cost, and quantity.
type CostInvItem struct {
	SKU  string  // your item identifier
	Cost float64 // unit cost, e.g. 12.34
	Qty  int     // available quantity
}

// BuildFlatCostInv returns the feed bytes and a timestamped filename.
// Format: one header row, then lines like "SenderID|SKU|Cost|Qty".
func BuildFlatCostInv(items []CostInvItem, senderID string) (data []byte, filename string, err error) {
	ts := time.Now().Format("20060102_150405")
	filename = fmt.Sprintf("COSTINV_%s.txt", ts)

	var buf bytes.Buffer
	// Header row (adjust column names to Amazon’s expected names)
	buf.WriteString("SenderID|SKU|Cost|Quantity\n")

	for _, itm := range items {
		line := fmt.Sprintf("%s|%s|%.2f|%d\n",
			senderID,
			itm.SKU,
			itm.Cost,
			itm.Qty,
		)
		buf.WriteString(line)
	}

	return buf.Bytes(), filename, nil
}
