package panel

import (
	"regexp"
	"strings"
	"time"
)

var (
	logTimestamp = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})`)
	matchedEvent = regexp.MustCompile(`Event matched: ([^,\s]+)`)
	sentTarget   = regexp.MustCompile(`Successfully sent notification to (.+)$`)
	failedTarget = regexp.MustCompile(`Failed to send notification to (.+?)(?::|$)`)
)

// DeliverySummary is deliberately payload-free. It is derived from normal
// application logs and is only used to make the operational dashboard useful.
type DeliverySummary struct {
	Total       int
	Failed      int
	SuccessRate int
	MaxDaily    int
	Days        []DeliveryDay
	Events      []MetricItem
	Recent      []DeliveryRow
}

type DeliveryDay struct {
	Label   string
	Success int
	Failed  int
}

type MetricItem struct {
	Label string
	Count int
}

type DeliveryRow struct {
	Time    string
	Target  string
	Success bool
}

func summarizeDeliveries(lines []string, now time.Time) DeliverySummary {
	start := now.AddDate(0, 0, -6)
	days := make([]DeliveryDay, 7)
	dayIndex := make(map[string]int, len(days))
	for i := range days {
		day := start.AddDate(0, 0, i)
		days[i].Label = day.Format("01/02")
		dayIndex[day.Format("2006-01-02")] = i
	}

	summary := DeliverySummary{Days: days}
	events := map[string]int{}
	for _, line := range lines {
		if match := matchedEvent.FindStringSubmatch(line); len(match) == 2 {
			events[match[1]]++
		}

		ok, target := deliveryFromLine(line)
		if target == "" {
			continue
		}
		ts, parsed := parseLogTime(line)
		if !parsed {
			continue
		}
		idx, exists := dayIndex[ts.Format("2006-01-02")]
		if !exists {
			continue
		}
		if ok {
			summary.Days[idx].Success++
		} else {
			summary.Days[idx].Failed++
		}
		if count := summary.Days[idx].Success + summary.Days[idx].Failed; count > summary.MaxDaily {
			summary.MaxDaily = count
		}
		summary.Total++
		if !ok {
			summary.Failed++
		}
		summary.Recent = append(summary.Recent, DeliveryRow{
			Time: ts.Format("01/02 15:04"), Target: target, Success: ok,
		})
	}
	if summary.Total > 0 {
		summary.SuccessRate = (summary.Total - summary.Failed) * 100 / summary.Total
	}
	for label, count := range events {
		summary.Events = append(summary.Events, MetricItem{Label: label, Count: count})
	}
	summary.Events = sortedMetricItems(summary.Events)
	if len(summary.Recent) > 8 {
		summary.Recent = summary.Recent[len(summary.Recent)-8:]
	}
	for left, right := 0, len(summary.Recent)-1; left < right; left, right = left+1, right-1 {
		summary.Recent[left], summary.Recent[right] = summary.Recent[right], summary.Recent[left]
	}
	return summary
}

func metricPercent(value, total int) int {
	if total <= 0 || value <= 0 {
		return 0
	}
	return value * 100 / total
}

func deliveryFromLine(line string) (bool, string) {
	if match := sentTarget.FindStringSubmatch(line); len(match) == 2 {
		return true, strings.TrimSpace(match[1])
	}
	if match := failedTarget.FindStringSubmatch(line); len(match) == 2 {
		return false, strings.TrimSpace(match[1])
	}
	return false, ""
}

func parseLogTime(line string) (time.Time, bool) {
	match := logTimestamp.FindStringSubmatch(line)
	if len(match) != 2 {
		return time.Time{}, false
	}
	ts, err := time.ParseInLocation("2006/01/02 15:04:05", match[1], time.Local)
	return ts, err == nil
}

func sortedMetricItems(items []MetricItem) []MetricItem {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && (items[j].Count > items[j-1].Count || (items[j].Count == items[j-1].Count && items[j].Label < items[j-1].Label)); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
	if len(items) > 6 {
		return items[:6]
	}
	return items
}
