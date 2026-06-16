package agent

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/text2sql"
)

// timeRangeSpec holds a parsed time range with descriptive label.
type timeRangeSpec struct {
	Label   string
	Start   time.Time
	End     time.Time
	IsRange bool
}

// parseExtendedTimeRange handles date ranges that the old resolveChineseTimeRange doesn't support.
// Examples: "1-6月", "上半年", "下半年", "最近三个月", "最近半年", "最近一年".
func parseExtendedTimeRange(message string) (timeRangeSpec, bool) {
	now := time.Now()
	loc := now.Location()
	y, currentMonth, _ := now.Date()
	lower := strings.ToLower(message)

	// ----- Half-year: 上半年 / 下半年 / H1 / H2 -----
	if strings.Contains(message, "上半年") || strings.Contains(lower, "h1") || strings.Contains(lower, "first half") {
		return timeRangeSpec{Label: "H1", Start: time.Date(y, 1, 1, 0, 0, 0, 0, loc), End: time.Date(y, 7, 1, 0, 0, 0, 0, loc), IsRange: true}, true
	}
	if strings.Contains(message, "下半年") || strings.Contains(lower, "h2") || strings.Contains(lower, "second half") {
		return timeRangeSpec{Label: "H2", Start: time.Date(y, 7, 1, 0, 0, 0, 0, loc), End: time.Date(y+1, 1, 1, 0, 0, 0, 0, loc), IsRange: true}, true
	}

	// ----- Quarters: Q1/Q2/Q3/Q4 / 一季度 -----
	for _, q := range []struct {
		keyword string
		month   time.Month
		label   string
	}{
		{"q1", 1, "Q1"}, {"一季度", 1, "Q1"},
		{"q2", 4, "Q2"}, {"二季度", 4, "Q2"},
		{"q3", 7, "Q3"}, {"三季度", 7, "Q3"},
		{"q4", 10, "Q4"}, {"四季度", 10, "Q4"},
	} {
		if strings.Contains(lower, q.keyword) {
			return timeRangeSpec{Label: q.label, Start: time.Date(y, q.month, 1, 0, 0, 0, 0, loc), End: time.Date(y, q.month+3, 1, 0, 0, 0, 0, loc), IsRange: true}, true
		}
	}

	// ----- "last N months/weeks/days/years" -----
	reRecently := regexp.MustCompile(`(last|最近|past)\s*(\d{1,2})\s*(months?|weeks?|days?|years?|月|周|天|年)`)
	if m := reRecently.FindStringSubmatch(strings.ToLower(message)); len(m) == 4 {
		n, _ := strconv.Atoi(m[2])
		if n <= 0 {
			n = 1
		}
		switch {
		case strings.Contains(m[3], "月") || strings.Contains(m[3], "month"):
			start := now.AddDate(0, -n, 0)
			return timeRangeSpec{Label: fmt.Sprintf("last %d months", n), Start: start, End: now.AddDate(0, 0, 1), IsRange: true}, true
		case strings.Contains(m[3], "周") || strings.Contains(m[3], "week"):
			start := now.AddDate(0, 0, -n*7)
			return timeRangeSpec{Label: fmt.Sprintf("last %d weeks", n), Start: start, End: now.AddDate(0, 0, 1), IsRange: true}, true
		case strings.Contains(m[3], "天") || strings.Contains(m[3], "day"):
			start := now.AddDate(0, 0, -n)
			return timeRangeSpec{Label: fmt.Sprintf("last %d days", n), Start: start, End: now.AddDate(0, 0, 1), IsRange: true}, true
		case strings.Contains(m[3], "年") || strings.Contains(m[3], "year"):
			start := now.AddDate(-n, 0, 0)
			return timeRangeSpec{Label: fmt.Sprintf("last %d years", n), Start: start, End: now.AddDate(0, 0, 1), IsRange: true}, true
		}
	}

	// ----- "N月-M月" / "N-M月" -----
	reRange := regexp.MustCompile(`(\d{1,2})\s*月?\s*[-~至到]\s*(\d{1,2})\s*月`)
	if m := reRange.FindStringSubmatch(message); len(m) == 3 {
		startMonth, _ := strconv.Atoi(m[1])
		endMonth, _ := strconv.Atoi(m[2])
		if startMonth >= 1 && startMonth <= 12 && endMonth >= 1 && endMonth <= 12 {
			yStart := y
			if time.Month(startMonth) > currentMonth {
				yStart = y - 1
			}
			yEnd := yStart
			if endMonth < startMonth {
				yEnd = yStart + 1
			}
			start := time.Date(yStart, time.Month(startMonth), 1, 0, 0, 0, 0, loc)
			end := time.Date(yEnd, time.Month(endMonth)+1, 1, 0, 0, 0, 0, loc)
			if endMonth == 12 {
				end = time.Date(yEnd+1, 1, 1, 0, 0, 0, 0, loc)
			}
			return timeRangeSpec{Label: m[1] + "-" + m[2] + "月", Start: start, End: end, IsRange: true}, true
		}
	}

	// ----- "this year" / "last year" -----
	if strings.Contains(lower, "this year") || strings.Contains(lower, "今年") {
		return timeRangeSpec{Label: "this year", Start: time.Date(y, 1, 1, 0, 0, 0, 0, loc), End: time.Date(y+1, 1, 1, 0, 0, 0, 0, loc), IsRange: true}, true
	}
	if strings.Contains(lower, "last year") || strings.Contains(lower, "去年") {
		return timeRangeSpec{Label: "last year", Start: time.Date(y-1, 1, 1, 0, 0, 0, 0, loc), End: time.Date(y, 1, 1, 0, 0, 0, 0, loc), IsRange: true}, true
	}

	// ----- "this month" / "last month" -----
	if strings.Contains(lower, "last month") || strings.Contains(lower, "上个月") || strings.Contains(lower, "上月") {
		start := time.Date(y, currentMonth, 1, 0, 0, 0, 0, loc).AddDate(0, -1, 0)
		return timeRangeSpec{Label: "last month", Start: start, End: start.AddDate(0, 1, 0), IsRange: true}, true
	}
	if strings.Contains(lower, "this month") || strings.Contains(lower, "本月") || strings.Contains(lower, "当前") {
		return timeRangeSpec{Label: "this month", Start: time.Date(y, currentMonth, 1, 0, 0, 0, 0, loc), End: time.Date(y, currentMonth+1, 1, 0, 0, 0, 0, loc), IsRange: true}, true
	}

	return timeRangeSpec{}, false
}

// findMatchingMetrics 从语义模型的已配置指标中查找用户问题命中的指标。
// 不做任何业务领域假设——所有指标都来自管理员在语义模型中配置的定义。
func findMatchingMetrics(message string, model text2sql.SemanticModel) []MetricRef {
	if len(model.Metrics) == 0 {
		return nil
	}
	var result []MetricRef
	lower := strings.ToLower(message)
	for _, m := range model.Metrics {
		if strings.Contains(lower, strings.ToLower(m.Label)) || strings.Contains(lower, strings.ToLower(m.Name)) {
			result = append(result, MetricRef{Entity: m.Entity, Field: m.Field, Agg: m.Aggregation, Alias: m.Name})
		}
	}
	return result
}

// MetricRef is a lightweight reference to a metric for fast path.
type MetricRef struct {
	Entity string
	Field  string
	Agg    string
	Alias  string
}
