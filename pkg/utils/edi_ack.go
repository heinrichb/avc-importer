// pkg/utils/edi_ack.go
package utils

import (
	"fmt"
	"regexp"
	"time"
)

var (
	// Capture:
	// 1: ISA sender ID
	// 2: ISA receiver ID
	// 3: Date (YYMMDD)
	// 4: Time (HHMM)
	// 5: Interchange control number (e.g. 900000009)
	isaRe = regexp.MustCompile(
		`ISA\*00\*.+\*00\*.+\*ZZ\*([^\*]+)\*ZZ\*([^\*]+)\*` +
			`([0-9]{6})\*([0-9]{4})\*U\*00400\*([0-9]+)\*`)

	// Capture:
	// 1: GS sender ID
	// 2: GS receiver ID
	// 3: GS date (YYYYMMDD)
	// 4: GS time (HHMM)
	// 5: GS control number
	gsRe = regexp.MustCompile(
		`GS\*PO\*([^\*]+)\*([^\*]+)\*([0-9]{8})\*([0-9]{4})\*([0-9]+)\*X`)

	// Capture the ST control number from ST*850*0001
	stRe = regexp.MustCompile(`ST\*850\*([0-9]+)`)
)

/*
Generate997 builds a minimal 997 Functional Acknowledgment.
It echoes back the ISA/GS/ST control numbers from the inbound X12 and
uses senderID as your GS sender ID.

Parameters:
  - in:       raw contents of the 850 file
  - senderID: your Amazon‑assigned GS ID (configured in edi.senderId)

Returns:
  - a string containing the 997 EDI document
  - an error if any required segment cannot be parsed
*/
func Generate997(in, senderID string) (string, error) {
	// parse ISA
	m := isaRe.FindStringSubmatch(in)
	if len(m) < 6 {
		return "", fmt.Errorf("invalid ISA segment: expected 5 captures, got %d", len(m)-1)
	}
	origInterchangeCtrl := m[5]

	// parse GS
	m2 := gsRe.FindStringSubmatch(in)
	if len(m2) < 6 {
		return "", fmt.Errorf("invalid GS segment: expected 5 captures, got %d", len(m2)-1)
	}
	origDate := m2[3]
	origTime := m2[4]
	origGroupCtrl := m2[5]

	// parse ST
	m3 := stRe.FindStringSubmatch(in)
	if len(m3) < 2 {
		return "", fmt.Errorf("invalid ST segment: could not find control number")
	}
	origSetCtrl := m3[1]

	// build timestamps for new ISA
	now := time.Now()
	isaDate := now.Format("060102") // YYMMDD
	isaTime := now.Format("1504")   // HHMM

	// ISA header: swap roles if needed; use senderID
	isa := fmt.Sprintf(
		"ISA*00*          *00*          *ZZ*AMAZON         *ZZ*%s*%s*%s*U*00400*%s*0*T*>~\n",
		senderID, isaDate, isaTime, origInterchangeCtrl,
	)
	// GS header
	gs := fmt.Sprintf(
		"GS*FA*AMAZON*%s*%s*%s*%s*X*004010~\n",
		senderID, origDate, origTime, origGroupCtrl,
	)
	// ST, AK1, AK9, SE, GE, IEA
	st := fmt.Sprintf("ST*997*%s~\n", origSetCtrl)
	ak1 := fmt.Sprintf("AK1*PO*%s~\n", origGroupCtrl)
	ak9 := "AK9*A*1*1*1~\n"
	se := fmt.Sprintf("SE*6*%s~\n", origSetCtrl)
	ge := fmt.Sprintf("GE*1*%s~\n", origGroupCtrl)
	iea := fmt.Sprintf("IEA*1*%s~", origInterchangeCtrl)

	return isa + gs + st + ak1 + ak9 + se + ge + iea, nil
}
