package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// buildEChartsOption generates a simple ECharts option JSON for line/bar/pie charts.
func buildEChartsOption(chartType, title, dataStr string) string {
	chartType = strings.ToLower(strings.TrimSpace(chartType))

	// Parse data: "label1:value1,label2:value2" or "[{\"name\":\"1月\",\"value\":10}]"
	labels, values := parseChartData(dataStr)

	option := map[string]interface{}{
		"title": map[string]interface{}{
			"text": title,
		},
		"tooltip": map[string]interface{}{
			"trigger": "axis",
		},
	}

	switch chartType {
	case "pie":
		pieData := make([]map[string]interface{}, 0, len(labels))
		for i := range labels {
			pieData = append(pieData, map[string]interface{}{"name": labels[i], "value": values[i]})
		}
		option["series"] = []map[string]interface{}{{
			"type": "pie",
			"data": pieData,
		}}
	default:
		option["xAxis"] = map[string]interface{}{
			"type": "category",
			"data": labels,
		}
		option["yAxis"] = map[string]interface{}{
			"type": "value",
		}
		option["series"] = []map[string]interface{}{{
			"type":   chartType,
			"data":   values,
			"smooth": true,
		}}
	}

	b, _ := json.Marshal(option)
	return string(b)
}

func parseChartData(dataStr string) ([]string, []float64) {
	var labels []string
	var values []float64

	dataStr = strings.TrimSpace(dataStr)

	// Try JSON array of objects
	if strings.HasPrefix(dataStr, "[") {
		var items []map[string]interface{}
		if json.Unmarshal([]byte(dataStr), &items) == nil {
			for _, item := range items {
				name, _ := item["name"].(string)
				val := 0.0
				switch v := item["value"].(type) {
				case float64:
					val = v
				case json.Number:
					val, _ = v.Float64()
				}
				labels = append(labels, name)
				values = append(values, val)
			}
			if len(labels) > 0 {
				return labels, values
			}
		}
	}

	// Try "label:value,label:value" format
	pairs := strings.Split(dataStr, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[0])
			val := 0.0
			_, _ = fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &val)
			labels = append(labels, name)
			values = append(values, val)
		}
	}
	if len(labels) > 0 {
		return labels, values
	}

	// Fallback: return placeholder
	labels = []string{"无数据"}
	values = []float64{0}
	return labels, values
}
