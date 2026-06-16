package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
)

// ChartRenderer generates ECharts options and HTML/chart rendering.
type ChartRenderer struct{}

type ChartRendererService = ChartRenderer

// NewChartRenderer creates a new ChartRenderer.
func NewChartRenderer() *ChartRenderer {
	return &ChartRenderer{}
}

func NewChartRendererService() *ChartRendererService {
	return NewChartRenderer()
}

// ChartData holds label-value pairs for chart rendering.
type ChartData struct {
	Labels []string  `json:"labels"`
	Values []float64 `json:"values"`
}

// GenerateEChartsOption returns a JSON string of the ECharts option.
// Supports bar, line, pie, area, scatter chart types.
func (c *ChartRenderer) GenerateEChartsOption(chartType string, title string, labels []string, values []float64) string {
	if len(labels) == 0 {
		labels = []string{"无数据"}
		values = []float64{0}
	}

	chartType = strings.ToLower(strings.TrimSpace(chartType))
	if chartType == "" {
		chartType = "bar"
	}

	option := map[string]interface{}{
		"title": map[string]interface{}{
			"text": title,
			"left": "center",
			"textStyle": map[string]interface{}{
				"fontSize": 16,
			},
		},
		"tooltip": map[string]interface{}{
			"trigger": "axis",
		},
		"grid": map[string]interface{}{
			"left":         "3%",
			"right":        "4%",
			"bottom":       "3%",
			"containLabel": true,
		},
	}

	switch chartType {
	case "pie":
		pieData := make([]map[string]interface{}, 0, len(labels))
		for i := range labels {
			pieData = append(pieData, map[string]interface{}{
				"name":  labels[i],
				"value": values[i],
			})
		}
		option["tooltip"] = map[string]interface{}{
			"trigger":   "item",
			"formatter": "{a} <br/>{b}: {c} ({d}%)",
		}
		option["legend"] = map[string]interface{}{
			"orient": "vertical",
			"left":   "left",
			"data":   labels,
		}
		option["series"] = []map[string]interface{}{{
			"name":   title,
			"type":   "pie",
			"radius": []string{"0%", "75%"},
			"center": []string{"50%", "55%"},
			"data":   pieData,
			"label": map[string]interface{}{
				"show":      true,
				"formatter": "{b}: {d}%",
			},
			"emphasis": map[string]interface{}{
				"itemStyle": map[string]interface{}{
					"shadowBlur":    10,
					"shadowOffsetX": 0,
					"shadowColor":   "rgba(0, 0, 0, 0.5)",
				},
			},
		}}
	case "area":
		option["xAxis"] = map[string]interface{}{
			"type":        "category",
			"data":        labels,
			"boundaryGap": false,
		}
		option["yAxis"] = map[string]interface{}{
			"type": "value",
		}
		option["series"] = []map[string]interface{}{{
			"name":      title,
			"type":      "line",
			"data":      values,
			"areaStyle": map[string]interface{}{},
			"smooth":    true,
		}}
	case "scatter":
		scatterData := make([][]float64, 0, len(labels))
		for i, v := range values {
			scatterData = append(scatterData, []float64{float64(i), v})
		}
		option["xAxis"] = map[string]interface{}{
			"type": "value",
		}
		option["yAxis"] = map[string]interface{}{
			"type": "value",
		}
		option["series"] = []map[string]interface{}{{
			"name":       title,
			"type":       "scatter",
			"data":       scatterData,
			"symbolSize": 12,
		}}
	case "line":
		option["xAxis"] = map[string]interface{}{
			"type": "category",
			"data": labels,
		}
		option["yAxis"] = map[string]interface{}{
			"type": "value",
		}
		option["series"] = []map[string]interface{}{{
			"name":   title,
			"type":   "line",
			"data":   values,
			"smooth": true,
			"lineStyle": map[string]interface{}{
				"width": 3,
			},
			"symbolSize": 8,
		}}
	default: // bar
		option["xAxis"] = map[string]interface{}{
			"type": "category",
			"data": labels,
		}
		option["yAxis"] = map[string]interface{}{
			"type": "value",
		}
		option["series"] = []map[string]interface{}{{
			"name":     title,
			"type":     "bar",
			"data":     values,
			"barWidth": "60%",
			"itemStyle": map[string]interface{}{
				"borderRadius": []int{4, 4, 0, 0},
			},
		}}
	}

	b, _ := json.MarshalIndent(option, "", "  ")
	return string(b)
}

// GenerateHTMLChart returns a complete HTML snippet that renders an ECharts chart.
func (c *ChartRenderer) GenerateHTMLChart(chartType string, title string, labels []string, values []float64) string {
	optionJSON := c.GenerateEChartsOption(chartType, title, labels, values)
	chartID := "chart_" + strings.ReplaceAll(uuid.New().String(), "-", "_")

	html := fmt.Sprintf(`<div id="%s" style="width: 100%%; height: 400px;"></div>
<script>
(function() {
    var chartDom = document.getElementById('%s');
    if (typeof echarts === 'undefined') {
        chartDom.innerHTML = '<p style="color:#999;text-align:center;padding-top:160px;">Chart rendering requires ECharts library</p>';
        return;
    }
    var myChart = echarts.init(chartDom);
    var option = %s;
    myChart.setOption(option);
    window.addEventListener('resize', function() { myChart.resize(); });
})();
</script>`, chartID, chartID, optionJSON)

	return html
}

// GenerateTextChart returns a simple ASCII/Unicode bar chart for PDF/text rendering.
func (c *ChartRenderer) GenerateTextChart(chartType string, title string, labels []string, values []float64) string {
	if len(labels) == 0 || len(values) == 0 {
		return title + ": 无数据"
	}

	chartType = strings.ToLower(strings.TrimSpace(chartType))
	if chartType == "" {
		chartType = "bar"
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("=", len(title)))
	sb.WriteString("\n\n")

	switch chartType {
	case "pie":
		total := 0.0
		for _, v := range values {
			total += v
		}
		if total == 0 {
			total = 1
		}
		barWidth := 20
		for i, label := range labels {
			pct := values[i] / total * 100
			filled := int(math.Round(pct / 100.0 * float64(barWidth)))
			if filled < 1 && pct > 0 {
				filled = 1
			}
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
			sb.WriteString(fmt.Sprintf(" %s %s %.1f%%\n", padRight(truncate(label, 12), 12), bar, pct))
		}
	default:
		// bar / line / area / scatter
		maxVal := 0.0
		for _, v := range values {
			if v > maxVal {
				maxVal = v
			}
		}
		if maxVal == 0 {
			maxVal = 1
		}
		barWidth := 30
		for i, label := range labels {
			val := values[i]
			filled := int(math.Round(val / maxVal * float64(barWidth)))
			if filled < 1 && val > 0 {
				filled = 1
			}
			bar := strings.Repeat("█", filled)
			sb.WriteString(fmt.Sprintf(" %s │%s %v\n", padRight(truncate(label, 12), 12), padRight(bar, barWidth), formatFloat(val)))
		}
		sb.WriteString(fmt.Sprintf(" %s └%s\n", padRight("", 12), strings.Repeat("─", barWidth)))
	}

	return sb.String()
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

func padRight(s string, length int) string {
	runes := []rune(s)
	if len(runes) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(runes))
}

func formatFloat(v float64) string {
	if v == math.Trunc(v) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.2f", v)
}
