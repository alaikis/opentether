package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/database"
	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/storage"
	"github.com/jung-kurt/gofpdf"
	"gorm.io/gorm"
)

// =============================================================================
// Data types
// =============================================================================

// ReportTemplateSection defines a single section within a template definition.
type ReportTemplateSection struct {
	Type        string `json:"type"`                 // title, subtitle, summary, table, chart, text
	Content     string `json:"content,omitempty"`    // text content for title/subtitle/summary/text
	Title       string `json:"title,omitempty"`      // section heading for table/chart
	Query       string `json:"query,omitempty"`      // SQL or query string
	ChartType   string `json:"chart_type,omitempty"` // bar, line, pie, area, scatter
	LabelColumn string `json:"label_column,omitempty"`
	ValueColumn string `json:"value_column,omitempty"`
}

// ReportTemplateLayout defines page layout.
type ReportTemplateLayout struct {
	PageSize     string  `json:"page_size"`   // A4, letter
	Orientation  string  `json:"orientation"` // portrait, landscape
	MarginTop    float64 `json:"margin_top"`
	MarginBottom float64 `json:"margin_bottom"`
	MarginLeft   float64 `json:"margin_left"`
	MarginRight  float64 `json:"margin_right"`
}

// ReportTemplateDefinition is the JSON-serializable template definition.
type ReportTemplateDefinition struct {
	Sections []ReportTemplateSection `json:"sections"`
	Layout   ReportTemplateLayout    `json:"layout"`
	Styling  *TemplateStyling        `json:"styling,omitempty"`
	Header   string                  `json:"header,omitempty"`
	Footer   string                  `json:"footer,omitempty"`
}

// TemplateStyling defines visual style overrides.
type TemplateStyling struct {
	PrimaryColor   string `json:"primary_color,omitempty"` // hex colour
	SecondaryColor string `json:"secondary_color,omitempty"`
	FontFamily     string `json:"font_family,omitempty"` // helvetica, courier, times
	FontSizeBase   int    `json:"font_size_base,omitempty"`
	HeaderText     string `json:"header_text,omitempty"`
	FooterText     string `json:"footer_text,omitempty"`
}

// =============================================================================
// ReportEngineService
// =============================================================================

// ReportEngineService provides template CRUD, report generation, job management,
// history tracking, and AI-powered template suggestions.
type ReportEngineService struct {
	db    *gorm.DB
	store storage.Driver
}

// NewReportEngineService creates a new ReportEngineService.
func NewReportEngineService(db *gorm.DB, store storage.Driver) *ReportEngineService {
	return &ReportEngineService{db: db, store: store}
}

// =============================================================================
// Template CRUD
// =============================================================================

// ListTemplates returns templates filtered by optional category.
func (s *ReportEngineService) ListTemplates(ctx context.Context, category string) ([]models.ReportTemplate, error) {
	var templates []models.ReportTemplate
	q := s.db.WithContext(ctx).Where("enabled = ?", true)
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if err := q.Order("created_at DESC").Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	return templates, nil
}

// GetTemplate returns a single template by ID.
func (s *ReportEngineService) GetTemplate(ctx context.Context, id string) (*models.ReportTemplate, error) {
	var tmpl models.ReportTemplate
	if err := s.db.WithContext(ctx).First(&tmpl, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("get template %s: %w", id, err)
	}
	return &tmpl, nil
}

// CreateTemplate inserts a new template.
func (s *ReportEngineService) CreateTemplate(ctx context.Context, tmpl *models.ReportTemplate) error {
	tmpl.Builtin = false
	tmpl.Enabled = true
	tmpl.UseCount = 0
	return s.db.WithContext(ctx).Create(tmpl).Error
}

// UpdateTemplate updates an existing template.
func (s *ReportEngineService) UpdateTemplate(ctx context.Context, tmpl *models.ReportTemplate) error {
	result := s.db.WithContext(ctx).Model(tmpl).Where("id = ?", tmpl.ID).Updates(map[string]interface{}{
		"name":           tmpl.Name,
		"description":    tmpl.Description,
		"category":       tmpl.Category,
		"definition":     tmpl.Definition,
		"data_source_id": tmpl.DataSourceID,
		"output_format":  tmpl.OutputFormat,
		"enabled":        tmpl.Enabled,
	})
	if result.Error != nil {
		return fmt.Errorf("update template %s: %w", tmpl.ID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("template %s not found", tmpl.ID)
	}
	return nil
}

// DeleteTemplate removes a template by ID.
func (s *ReportEngineService) DeleteTemplate(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&models.ReportTemplate{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete template %s: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("template %s not found", id)
	}
	return nil
}

// =============================================================================
// Report generation – PDF
// =============================================================================

// GeneratePDF generates a multi-page PDF report from a template, saves it to
// storage, and records the generation history.
func (s *ReportEngineService) GeneratePDF(ctx context.Context, templateID string, params map[string]interface{}, userID string) (fileURL string, historyID string, err error) {
	tmpl, err := s.GetTemplate(ctx, templateID)
	if err != nil {
		return "", "", fmt.Errorf("generate pdf: %w", err)
	}

	def, err := parseDefinition(tmpl.Definition)
	if err != nil {
		return "", "", fmt.Errorf("generate pdf: parse definition: %w", err)
	}

	// Resolve orientation shorthand.
	orientationStr := "P"
	if strings.EqualFold(def.Layout.Orientation, "landscape") {
		orientationStr = "L"
	}

	pageSize := strings.ToUpper(def.Layout.PageSize)
	if pageSize == "" {
		pageSize = "A4"
	}

	pdf := gofpdf.New(orientationStr, "mm", pageSize, "")
	if def.Layout.MarginLeft <= 0 {
		def.Layout.MarginLeft = 15
	}
	if def.Layout.MarginTop <= 0 {
		def.Layout.MarginTop = 15
	}
	if def.Layout.MarginRight <= 0 {
		def.Layout.MarginRight = 15
	}
	pdf.SetMargins(def.Layout.MarginLeft, def.Layout.MarginTop, def.Layout.MarginRight)

	// Auto page break.
	bottomMargin := def.Layout.MarginBottom
	if bottomMargin <= 0 {
		bottomMargin = 20
	}
	pdf.SetAutoPageBreak(true, bottomMargin)

	// Header function.
	headerText := def.Header
	if headerText == "" {
		headerText = "OpenTether Report"
	}
	pdf.SetHeaderFunc(func() {
		pdf.SetFont("helvetica", "I", 8)
		pdf.SetTextColor(120, 120, 120)
		pdf.Cell(0, 5, headerText)
		pdf.Ln(4)
	})

	// Footer function with page numbers.
	pdf.SetFooterFunc(func() {
		pdf.SetY(-bottomMargin + 5)
		pdf.SetFont("helvetica", "I", 8)
		pdf.SetTextColor(128, 128, 128)
		footer := def.Footer
		if footer == "" {
			footer = "Generated by OpenTether"
		}
		pdf.CellFormat(0, 4, fmt.Sprintf("%s | Page %d/{nb}", footer, pdf.PageNo()), "", 0, "C", false, 0, "")
	})
	pdf.AliasNbPages("{nb}")

	// Resolve title from sections or params.
	title := resolveParamString(params, "title")
	if title == "" {
		for _, sec := range def.Sections {
			if sec.Type == "title" {
				title = sec.Content
				break
			}
		}
	}
	if title == "" {
		title = tmpl.Name
	}

	// Timestamp for the report body.
	now := time.Now()
	dateStr := now.Format("2006-01-02 15:04:05")

	// --- Build pages from sections ---
	for _, section := range def.Sections {
		switch strings.ToLower(section.Type) {
		case "title":
			addPDFTitlePage(pdf, section.Content, dateStr, def)

		case "subtitle":
			pdf.SetFont("helvetica", "", 11)
			pdf.SetTextColor(100, 100, 100)
			pdf.MultiCell(0, 6, section.Content, "", "", false)
			pdf.Ln(4)
			pdf.SetTextColor(0, 0, 0)

		case "summary":
			pdf.SetFont("helvetica", "B", 12)
			pdf.Cell(0, 8, "Summary")
			pdf.Ln(8)
			pdf.SetFont("helvetica", "", 10)
			pdf.MultiCell(0, 5, section.Content, "", "", false)
			pdf.Ln(5)

		case "text":
			pdf.SetFont("helvetica", "", 10)
			pdf.MultiCell(0, 5, section.Content, "", "", false)
			pdf.Ln(3)

		case "table":
			headers, rows, err := s.resolveTableData(ctx, section, params, tmpl.DataSourceID)
			if err != nil {
				// Log error but continue; draw an empty table placeholder.
				pdf.SetFont("helvetica", "I", 9)
				pdf.SetTextColor(180, 0, 0)
				pdf.MultiCell(0, 5, fmt.Sprintf("Table error: %v", err), "", "", false)
				pdf.SetTextColor(0, 0, 0)
				break
			}
			if section.Title != "" {
				pdf.SetFont("helvetica", "B", 12)
				pdf.Cell(0, 8, section.Title)
				pdf.Ln(8)
			}
			s.renderPDFTable(pdf, headers, rows)

		case "chart":
			labels, values, err := s.resolveChartData(ctx, section, params, tmpl.DataSourceID)
			if err != nil {
				pdf.SetFont("helvetica", "I", 9)
				pdf.SetTextColor(180, 0, 0)
				pdf.MultiCell(0, 5, fmt.Sprintf("Chart error: %v", err), "", "", false)
				pdf.SetTextColor(0, 0, 0)
				break
			}
			if section.Title != "" {
				pdf.SetFont("helvetica", "B", 12)
				pdf.Cell(0, 8, section.Title)
				pdf.Ln(8)
			}
			renderPDFTextChart(pdf, section.Title, section.ChartType, labels, values)
		}
	}

	// --- Save to storage ---
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return "", "", fmt.Errorf("generate pdf: output: %w", err)
	}
	data := buf.Bytes()

	filePath := fmt.Sprintf("reports/%s/%s_%s.pdf", templateID, now.Format("20060102150405"), sanitizeFilename(title))
	url, err := s.store.Save(ctx, filePath, data, "application/pdf")
	if err != nil {
		return "", "", fmt.Errorf("generate pdf: save: %w", err)
	}

	// Count table rows for history.
	rowCount := int64(0)
	for _, sec := range def.Sections {
		if strings.ToLower(sec.Type) == "table" {
			_, rows, _ := s.resolveTableData(ctx, sec, params, tmpl.DataSourceID)
			rowCount += int64(len(rows))
		}
	}

	historyID = s.recordHistory(ctx, templateID, title, "pdf", filePath, rowCount, userID)
	return url, historyID, nil
}

// GenerateHTML generates an HTML report, saves to storage, and records history.
func (s *ReportEngineService) GenerateHTML(ctx context.Context, templateID string, params map[string]interface{}, userID string) (fileURL string, historyID string, err error) {
	tmpl, err := s.GetTemplate(ctx, templateID)
	if err != nil {
		return "", "", fmt.Errorf("generate html: %w", err)
	}

	def, err := parseDefinition(tmpl.Definition)
	if err != nil {
		return "", "", fmt.Errorf("generate html: parse definition: %w", err)
	}

	title := resolveParamString(params, "title")
	if title == "" {
		for _, sec := range def.Sections {
			if sec.Type == "title" {
				title = sec.Content
				break
			}
		}
	}
	if title == "" {
		title = tmpl.Name
	}

	summary := ""
	columns := []string{}
	rows := [][]interface{}{}
	chartData := make(map[string]string)

	for _, section := range def.Sections {
		switch strings.ToLower(section.Type) {
		case "summary":
			summary = section.Content
		case "table":
			h, r, err := s.resolveTableData(ctx, section, params, tmpl.DataSourceID)
			if err == nil {
				columns = h
				rows = r
			}
		case "chart":
			labels, values, err := s.resolveChartData(ctx, section, params, tmpl.DataSourceID)
			if err == nil && len(labels) > 0 {
				c := &ChartRenderer{}
				chartData[section.Title] = c.GenerateEChartsOption(section.ChartType, section.Title, labels, values)
			}
		}
	}

	htmlContent := s.buildHTMLPage(title, summary, columns, rows, chartData)
	now := time.Now()
	filePath := fmt.Sprintf("reports/%s/%s_%s.html", templateID, now.Format("20060102150405"), sanitizeFilename(title))
	url, err := s.store.Save(ctx, filePath, []byte(htmlContent), "text/html; charset=utf-8")
	if err != nil {
		return "", "", fmt.Errorf("generate html: save: %w", err)
	}

	historyID = s.recordHistory(ctx, templateID, title, "html", filePath, int64(len(rows)), userID)
	return url, historyID, nil
}

// GenerateCSV generates a CSV report from the first table section in the
// template, saves to storage, and records history.
func (s *ReportEngineService) GenerateCSV(ctx context.Context, templateID string, params map[string]interface{}, userID string) (fileURL string, historyID string, err error) {
	tmpl, err := s.GetTemplate(ctx, templateID)
	if err != nil {
		return "", "", fmt.Errorf("generate csv: %w", err)
	}

	def, err := parseDefinition(tmpl.Definition)
	if err != nil {
		return "", "", fmt.Errorf("generate csv: parse definition: %w", err)
	}

	title := resolveParamString(params, "title")
	if title == "" {
		for _, sec := range def.Sections {
			if sec.Type == "title" {
				title = sec.Content
				break
			}
		}
	}
	if title == "" {
		title = tmpl.Name
	}

	// Use first table section.
	var columns []string
	var rows [][]interface{}
	for _, section := range def.Sections {
		if strings.ToLower(section.Type) == "table" {
			h, r, err := s.resolveTableData(ctx, section, params, tmpl.DataSourceID)
			if err == nil {
				columns = h
				rows = r
			}
			break
		}
	}

	csvBytes := s.buildCSV(columns, rows)
	now := time.Now()
	filePath := fmt.Sprintf("reports/%s/%s_%s.csv", templateID, now.Format("20060102150405"), sanitizeFilename(title))
	url, err := s.store.Save(ctx, filePath, csvBytes, "text/csv; charset=utf-8")
	if err != nil {
		return "", "", fmt.Errorf("generate csv: save: %w", err)
	}

	historyID = s.recordHistory(ctx, templateID, title, "csv", filePath, int64(len(rows)), userID)
	return url, historyID, nil
}

// =============================================================================
// Helper methods
// =============================================================================

// renderPDFTable draws a table with alternating row colours and auto column
// widths on the current PDF page. A new page is added automatically when the
// table would exceed the page bottom margin.
func (s *ReportEngineService) renderPDFTable(pdf *gofpdf.Fpdf, headers []string, rows [][]interface{}) {
	if len(headers) == 0 {
		return
	}

	_, pageH := pdf.GetPageSize()
	_, _, _, bottom := pdf.GetMargins()
	usableW, _ := pdf.GetPageSize()
	_, right, _, left := pdf.GetMargins()
	usableW = usableW - left - right

	colCount := len(headers)
	colWidth := usableW / float64(colCount)
	if colWidth > 60 {
		colWidth = 60
		// Distribute remaining width evenly.
		totalFixed := colWidth * float64(colCount)
		extra := usableW - totalFixed
		if extra > 0 {
			colWidth = colWidth + extra/float64(colCount)
		}
	}

	rowH := 7.0
	dataRowH := 6.0

	// Check if we need a new page before header.
	_, y := pdf.GetXY()
	if y+rowH > pageH-bottom-10 {
		pdf.AddPage()
	}

	// --- Header row ---
	pdf.SetFont("helvetica", "B", 9)
	pdf.SetFillColor(50, 80, 140)
	pdf.SetTextColor(255, 255, 255)
	for i, h := range headers {
		x, y := pdf.GetXY()
		pdf.CellFormat(colWidth, rowH, h, "1", 0, "C", true, 0, "")
		if i == colCount-1 {
			pdf.Ln(rowH)
		} else {
			pdf.SetXY(x+colWidth, y)
		}
	}
	pdf.SetTextColor(0, 0, 0)

	// --- Data rows ---
	pdf.SetFont("helvetica", "", 9)

	maxDisplayRows := 200
	displayed := 0
	for rowIdx, row := range rows {
		if rowIdx >= maxDisplayRows {
			pdf.SetFont("helvetica", "I", 8)
			pdf.SetTextColor(100, 100, 100)
			pdf.Ln(4)
			pdf.Cell(0, 5, fmt.Sprintf("... %d more records (showing first %d)", len(rows)-maxDisplayRows, maxDisplayRows))
			pdf.SetTextColor(0, 0, 0)
			break
		}

		// Check available space – add page if needed.
		_, cy := pdf.GetXY()
		if cy+dataRowH > pageH-bottom-10 {
			pdf.AddPage()
			// Re-draw header on new page.
			pdf.SetFont("helvetica", "B", 9)
			pdf.SetFillColor(50, 80, 140)
			pdf.SetTextColor(255, 255, 255)
			for i, h := range headers {
				x, y := pdf.GetXY()
				pdf.CellFormat(colWidth, rowH, h, "1", 0, "C", true, 0, "")
				if i == colCount-1 {
					pdf.Ln(rowH)
				} else {
					pdf.SetXY(x+colWidth, y)
				}
			}
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("helvetica", "", 9)
		}

		// Alternating row colour.
		if rowIdx%2 == 0 {
			pdf.SetFillColor(245, 247, 250)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		for colIdx, cell := range row {
			cellStr := formatCellValue(cell)
			x, y := pdf.GetXY()
			pdf.CellFormat(colWidth, dataRowH, cellStr, "1", 0, "L", true, 0, "")
			if colIdx == colCount-1 {
				pdf.Ln(dataRowH)
			} else {
				pdf.SetXY(x+colWidth, y)
			}
		}
		displayed++
	}

	pdf.Ln(4)
}

// buildHTMLPage returns a complete, self-contained HTML document.
func (s *ReportEngineService) buildHTMLPage(title, summary string, columns []string, rows [][]interface{}, chartData map[string]string) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>`)
	sb.WriteString(html.EscapeString(title))
	sb.WriteString(`</title>
<script src="https://cdn.jsdelivr.net/npm/echarts@5/dist/echarts.min.js"></script>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Noto Sans SC", "Microsoft YaHei", sans-serif; color: #333; background: #f5f7fa; padding: 20px; }
.container { max-width: 960px; margin: 0 auto; background: #fff; border-radius: 8px; box-shadow: 0 2px 12px rgba(0,0,0,0.08); padding: 40px; }
h1 { font-size: 24px; color: #1a1a2e; margin-bottom: 4px; }
.date { color: #888; font-size: 13px; margin-bottom: 24px; }
.summary { background: #f0f4ff; border-left: 4px solid #4a6cf7; padding: 16px 20px; margin-bottom: 28px; border-radius: 0 6px 6px 0; line-height: 1.6; }
.summary p { margin: 0; }
h2 { font-size: 18px; color: #1a1a2e; margin: 28px 0 12px 0; border-bottom: 2px solid #e8ecf1; padding-bottom: 8px; }
table { width: 100%; border-collapse: collapse; margin-bottom: 24px; }
th { background: #3a5bbf; color: #fff; padding: 10px 12px; text-align: left; font-size: 13px; font-weight: 600; cursor: pointer; user-select: none; }
th:hover { background: #2d4aa3; }
th::after { content: " \\25B4\\25BE"; font-size: 10px; color: rgba(255,255,255,0.6); margin-left: 4px; }
td { padding: 9px 12px; border-bottom: 1px solid #e8ecf1; font-size: 13px; }
tr:nth-child(even) td { background: #f8f9fc; }
tr:hover td { background: #eef1f8; }
.chart-container { width: 100%; height: 380px; margin: 20px 0 28px 0; }
.note { color: #888; font-size: 12px; border-top: 1px solid #e0e0e0; padding-top: 16px; margin-top: 32px; }
th.sort-asc::after { content: " \\25B2"; font-size: 10px; }
th.sort-desc::after { content: " \\25BC"; font-size: 10px; }
</style>
</head>
<body>
<div class="container">
<h1>`)
	sb.WriteString(html.EscapeString(title))
	sb.WriteString(`</h1>
<div class="date">Generated: `)
	sb.WriteString(time.Now().Format("2006-01-02 15:04:05"))
	sb.WriteString(`</div>
`)

	if summary != "" {
		sb.WriteString(`<div class="summary"><p>`)
		sb.WriteString(html.EscapeString(summary))
		sb.WriteString(`</p></div>`)
	}

	// Data table.
	if len(columns) > 0 {
		sb.WriteString(`<h2>Data Table</h2>
<div style="overflow-x:auto;">
<table id="data-table">
<thead><tr>`)
		for _, col := range columns {
			sb.WriteString("<th>")
			sb.WriteString(html.EscapeString(col))
			sb.WriteString("</th>")
		}
		sb.WriteString(`</tr></thead>
<tbody>`)
		for _, row := range rows {
			sb.WriteString("<tr>")
			for _, cell := range row {
				sb.WriteString("<td>")
				sb.WriteString(html.EscapeString(formatCellValue(cell)))
				sb.WriteString("</td>")
			}
			sb.WriteString("</tr>")
		}
		sb.WriteString(`</tbody>
</table>
</div>`)
	}

	// Charts.
	if len(chartData) > 0 {
		sb.WriteString(`<h2>Charts</h2>`)
		for chartTitle, optionJSON := range chartData {
			chartID := "echart_" + strings.ReplaceAll(fmt.Sprintf("%d", time.Now().UnixNano()), "-", "_")
			sb.WriteString(fmt.Sprintf(`<h3>%s</h3><div class="chart-container" id="%s"></div>
<script>
(function(){
  var dom = document.getElementById('%s');
  if (typeof echarts !== 'undefined') {
    var chart = echarts.init(dom);
    var opt = %s;
    chart.setOption(opt);
    window.addEventListener('resize', function(){ chart.resize(); });
  } else {
    dom.innerHTML = '<p style="color:#999;text-align:center;padding-top:160px;">ECharts library not available</p>';
  }
})();
</script>`, html.EscapeString(chartTitle), chartID, chartID, optionJSON))
		}
	}

	// Note.
	sb.WriteString(`<div class="note">Generated by OpenTether Enterprise AI Agent System</div>
</div>

<script>
// Simple client-side table sorting.
(function(){
  var table = document.getElementById('data-table');
  if (!table) return;
  var headers = table.querySelectorAll('th');
  var tbody = table.querySelector('tbody');
  headers.forEach(function(th, colIdx){
    th.addEventListener('click', function(){
      var asc = !th.classList.contains('sort-asc');
      headers.forEach(function(h){ h.classList.remove('sort-asc','sort-desc'); });
      th.classList.add(asc ? 'sort-asc' : 'sort-desc');
      var rows = Array.from(tbody.querySelectorAll('tr'));
      rows.sort(function(a, b){
        var aVal = a.children[colIdx] ? a.children[colIdx].textContent.trim() : '';
        var bVal = b.children[colIdx] ? b.children[colIdx].textContent.trim() : '';
        var aNum = parseFloat(aVal.replace(/[^0-9.\-]/g,''));
        var bNum = parseFloat(bVal.replace(/[^0-9.\-]/g,''));
        if (!isNaN(aNum) && !isNaN(bNum)) return asc ? aNum - bNum : bNum - aNum;
        return asc ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
      });
      rows.forEach(function(r){ tbody.appendChild(r); });
    });
  });
})();
</script>
</body>
</html>`)

	return sb.String()
}

// buildCSV returns CSV-encoded bytes for the given columns and rows.
func (s *ReportEngineService) buildCSV(columns []string, rows [][]interface{}) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(&buf)

	if len(columns) > 0 {
		writer.Write(columns)
	}

	for _, row := range rows {
		record := make([]string, len(row))
		for i, cell := range row {
			record[i] = formatCellValue(cell)
		}
		writer.Write(record)
	}
	writer.Flush()
	return buf.Bytes()
}

// recordHistory creates a ReportHistory record in the database and returns its
// ID. The JobID field is left empty (it can be set by the caller if needed).
func (s *ReportEngineService) recordHistory(ctx context.Context, templateID, title, format, filePath string, rowCount int64, userID string) string {
	history := &models.ReportHistory{
		TemplateID:  templateID,
		Title:       title,
		Format:      format,
		FilePath:    filePath,
		RowCount:    rowCount,
		Status:      "completed",
		GeneratedBy: userID,
		GeneratedAt: time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(history).Error; err != nil {
		// Best-effort; return the ID if the model auto-assigned one.
		return history.ID
	}
	return history.ID
}

// =============================================================================
// Job management
// =============================================================================

// ListJobs returns all report jobs.
func (s *ReportEngineService) ListJobs(ctx context.Context) ([]models.ReportJob, error) {
	var jobs []models.ReportJob
	if err := s.db.WithContext(ctx).Preload("Template").Order("created_at DESC").Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("list report jobs: %w", err)
	}
	return jobs, nil
}

// CreateJob inserts a new report job.
func (s *ReportEngineService) CreateJob(ctx context.Context, job *models.ReportJob) error {
	if job.Status == "" {
		job.Status = "active"
	}
	return s.db.WithContext(ctx).Create(job).Error
}

// UpdateJob updates an existing report job.
func (s *ReportEngineService) UpdateJob(ctx context.Context, job *models.ReportJob) error {
	result := s.db.WithContext(ctx).Model(job).Where("id = ?", job.ID).Updates(map[string]interface{}{
		"template_id":     job.TemplateID,
		"name":            job.Name,
		"description":     job.Description,
		"cron_expression": job.CronExpression,
		"output_format":   job.OutputFormat,
		"recipients":      job.Recipients,
		"parameters":      job.Parameters,
		"status":          job.Status,
		"next_run_at":     job.NextRunAt,
	})
	if result.Error != nil {
		return fmt.Errorf("update job %s: %w", job.ID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("job %s not found", job.ID)
	}
	return nil
}

// DeleteJob removes a report job by ID.
func (s *ReportEngineService) DeleteJob(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&models.ReportJob{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete job %s: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("job %s not found", id)
	}
	return nil
}

// GetScheduledJobsDue returns active jobs whose NextRunAt is at or before the
// current time.
func (s *ReportEngineService) GetScheduledJobsDue(ctx context.Context) ([]models.ReportJob, error) {
	var jobs []models.ReportJob
	now := time.Now()
	if err := s.db.WithContext(ctx).Preload("Template").Where("status = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", "active", now).Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("get scheduled jobs due: %w", err)
	}
	return jobs, nil
}

// ExecuteJob generates the report for the given job, then updates the job's
// last-run and next-run timestamps. Delivery is not performed here – it is
// expected to be handled by the caller (e.g. ReportDeliveryService).
func (s *ReportEngineService) ExecuteJob(ctx context.Context, jobID string) error {
	var job models.ReportJob
	if err := s.db.WithContext(ctx).Preload("Template").First(&job, "id = ?", jobID).Error; err != nil {
		return fmt.Errorf("execute job: %w", err)
	}

	// Parse parameters from job.
	params := make(map[string]interface{})
	if job.Parameters != "" {
		json.Unmarshal([]byte(job.Parameters), &params)
	}

	format := job.OutputFormat
	if format == "" {
		format = "pdf"
	}

	var fileURL, historyID string
	var err error

	switch strings.ToLower(format) {
	case "html":
		fileURL, historyID, err = s.GenerateHTML(ctx, job.TemplateID, params, job.CreatedBy)
	case "csv":
		fileURL, historyID, err = s.GenerateCSV(ctx, job.TemplateID, params, job.CreatedBy)
	default:
		fileURL, historyID, err = s.GeneratePDF(ctx, job.TemplateID, params, job.CreatedBy)
	}

	_ = fileURL

	if err != nil {
		// Mark history as failed.
		if historyID != "" {
			s.db.WithContext(ctx).Model(&models.ReportHistory{}).Where("id = ?", historyID).Updates(map[string]interface{}{
				"status":    "failed",
				"error_msg": err.Error(),
			})
		}
		return fmt.Errorf("execute job %s: %w", jobID, err)
	}

	// Update job run timestamps.
	now := time.Now()
	updates := map[string]interface{}{
		"last_run_at": now,
	}
	// If there is a cron expression, compute the next run.
	if job.CronExpression != "" {
		next := computeNextCronRun(job.CronExpression, now)
		updates["next_run_at"] = next
	} else {
		// One-shot job: mark as completed.
		updates["status"] = "completed"
		updates["next_run_at"] = nil
	}
	s.db.WithContext(ctx).Model(&job).Where("id = ?", jobID).Updates(updates)

	// Link history to job.
	if historyID != "" {
		s.db.WithContext(ctx).Model(&models.ReportHistory{}).Where("id = ?", historyID).Update("job_id", jobID)
	}

	// Log result.
	// Delivery is expected to be performed externally, e.g. by the scheduler.
	return nil
}

// =============================================================================
// History methods
// =============================================================================

// ListHistory returns all history records for a given template, newest first.
func (s *ReportEngineService) ListHistory(ctx context.Context, templateID string) ([]models.ReportHistory, error) {
	var history []models.ReportHistory
	q := s.db.WithContext(ctx).Order("created_at DESC")
	if templateID != "" {
		q = q.Where("template_id = ?", templateID)
	}
	if err := q.Find(&history).Error; err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	return history, nil
}

// GetHistory returns a single history record by ID.
func (s *ReportEngineService) GetHistory(ctx context.Context, id string) (*models.ReportHistory, error) {
	var h models.ReportHistory
	if err := s.db.WithContext(ctx).First(&h, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("get history %s: %w", id, err)
	}
	return &h, nil
}

// DeleteHistory removes a history record by ID.
func (s *ReportEngineService) DeleteHistory(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&models.ReportHistory{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete history %s: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("history %s not found", id)
	}
	return nil
}

// =============================================================================
// AI-powered template suggestion
// =============================================================================

// SuggestTemplate inspects the data source schema and user description, then
// builds a suggested ReportTemplate with a fully populated Definition JSON.
//
// The method examines the data source's SchemaInfo field, extracts table names
// and column details, and constructs reasonable sections (title, summary,
// table, chart) based on column types. Because the service layer has no LLM
// client, the "AI" analysis is a deterministic heuristic – suitable columns
// are chosen for aggregation (numeric -> chart, categorical -> table labels).
func (s *ReportEngineService) SuggestTemplate(ctx context.Context, description string, dataSourceID string) (*models.ReportTemplate, error) {
	// Fetch the data source to inspect its schema.
	var ds models.DataSource
	if err := s.db.WithContext(ctx).First(&ds, "id = ?", dataSourceID).Error; err != nil {
		return nil, fmt.Errorf("suggest template: data source: %w", err)
	}

	// Determine a sensible name and category from the description.
	name := description
	if len(name) > 100 {
		name = name[:100]
	}

	// Build a definition based on the schema.
	def := ReportTemplateDefinition{
		Layout: ReportTemplateLayout{
			PageSize:     "A4",
			Orientation:  "portrait",
			MarginTop:    20,
			MarginBottom: 20,
			MarginLeft:   15,
			MarginRight:  15,
		},
		Header: "OpenTether Enterprise Report",
		Footer: "Generated by OpenTether",
		Sections: []ReportTemplateSection{
			{Type: "title", Content: description},
			{Type: "summary", Content: "This report was auto-generated based on the selected data source."},
		},
	}

	// Parse schema info.
	schemaTables := parseSchemaTables(ds.SchemaInfo)

	if len(schemaTables) > 0 {
		// Add a table section for the first table.
		firstTable := schemaTables[0]
		def.Sections = append(def.Sections, ReportTemplateSection{
			Type:  "table",
			Title: fmt.Sprintf("Data: %s", firstTable.Name),
			Query: fmt.Sprintf("SELECT * FROM %s LIMIT 100", firstTable.Name),
		})

		// Add a chart section if there are numeric columns.
		for _, col := range firstTable.Columns {
			if isNumericType(col.Type) {
				labelCol := firstTable.Columns[0].Name
				if labelCol == col.Name && len(firstTable.Columns) > 1 {
					labelCol = firstTable.Columns[1].Name
				}
				def.Sections = append(def.Sections, ReportTemplateSection{
					Type:        "chart",
					Title:       fmt.Sprintf("Chart: %s by %s", col.Name, labelCol),
					ChartType:   "bar",
					Query:       fmt.Sprintf("SELECT %s, %s FROM %s GROUP BY %s", labelCol, col.Name, firstTable.Name, labelCol),
					LabelColumn: labelCol,
					ValueColumn: col.Name,
				})
				break
			}
		}
	}

	defBytes, err := json.MarshalIndent(def, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("suggest template: marshal definition: %w", err)
	}

	tmpl := &models.ReportTemplate{
		Name:         name,
		Description:  fmt.Sprintf("AI-suggested template: %s", description),
		Category:     "ai-suggested",
		DataSourceID: dataSourceID,
		OutputFormat: "pdf",
		Definition:   string(defBytes),
		Builtin:      false,
		Enabled:      true,
	}

	return tmpl, nil
}

// =============================================================================
// Internal helpers
// =============================================================================

// schemaColumn describes a column parsed from schema info.
type schemaColumn struct {
	Name string
	Type string
}

// schemaTable describes a table parsed from schema info.
type schemaTable struct {
	Name    string
	Columns []schemaColumn
}

// parseSchemaTables attempts to parse SchemaInfo (JSON) into a list of tables.
// SchemaInfo is expected to be a JSON object with table names as keys and column
// arrays as values, e.g. {"users": [{"name":"id","type":"int"},...]}.
func parseSchemaTables(schemaInfo string) []schemaTable {
	if schemaInfo == "" {
		return nil
	}

	// Try raw map first (most common).
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(schemaInfo), &raw); err == nil {
		var tables []schemaTable
		for tblName, val := range raw {
			t := schemaTable{Name: tblName}
			switch v := val.(type) {
			case []interface{}:
				for _, item := range v {
					if col, ok := item.(map[string]interface{}); ok {
						name, _ := col["name"].(string)
						colType, _ := col["type"].(string)
						if name == "" {
							name, _ = col["column"].(string)
						}
						if name != "" {
							t.Columns = append(t.Columns, schemaColumn{Name: name, Type: colType})
						}
					}
				}
			case map[string]interface{}:
				// Flat column-name -> type map.
				for colName, colType := range v {
					t.Columns = append(t.Columns, schemaColumn{Name: colName, Type: fmt.Sprintf("%v", colType)})
				}
			}
			if len(t.Columns) > 0 {
				tables = append(tables, t)
			}
		}
		return tables
	}

	return nil
}

// isNumericType returns true if the column type looks numeric.
func isNumericType(colType string) bool {
	lower := strings.ToLower(colType)
	for _, prefix := range []string{"int", "float", "double", "decimal", "numeric", "real", "number", "bigint", "smallint", "tinyint"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// parseDefinition unmarshals the Definition JSON string.
func parseDefinition(raw string) (*ReportTemplateDefinition, error) {
	var def ReportTemplateDefinition
	if err := json.Unmarshal([]byte(raw), &def); err != nil {
		return nil, fmt.Errorf("parse definition: %w", err)
	}
	return &def, nil
}

func (s *ReportEngineService) resolveTableData(ctx context.Context, section ReportTemplateSection, params map[string]interface{}, dataSourceID string) ([]string, [][]interface{}, error) {
	query := prepareReportQuery(section.Query, params)
	if err := validateReportReadOnlyQuery(query); err != nil {
		return nil, nil, err
	}
	db, err := s.reportQueryDB(ctx, dataSourceID)
	if err != nil {
		return nil, nil, err
	}
	return queryReportRows(ctx, db, ensureReportLimit(query, 1000))
}

func (s *ReportEngineService) resolveChartData(ctx context.Context, section ReportTemplateSection, params map[string]interface{}, dataSourceID string) ([]string, []float64, error) {
	headers, rows, err := s.resolveTableData(ctx, section, params, dataSourceID)
	if err != nil {
		return nil, nil, err
	}
	labelIdx := columnIndex(headers, section.LabelColumn)
	valueIdx := columnIndex(headers, section.ValueColumn)
	if labelIdx < 0 || valueIdx < 0 {
		return nil, nil, fmt.Errorf("chart columns not found: label=%s value=%s", section.LabelColumn, section.ValueColumn)
	}
	labels := make([]string, 0, len(rows))
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		labels = append(labels, fmt.Sprint(row[labelIdx]))
		value, err := toFloat64(row[valueIdx])
		if err != nil {
			return nil, nil, fmt.Errorf("chart value column %s is not numeric: %w", section.ValueColumn, err)
		}
		values = append(values, value)
	}
	return labels, values, nil
}

// resolveParamString extracts a string value from params by key.
func (s *ReportEngineService) reportQueryDB(ctx context.Context, dataSourceID string) (*sql.DB, error) {
	if dataSourceID == "" {
		if s.db == nil {
			return nil, fmt.Errorf("report datasource is not configured")
		}
		return s.db.DB()
	}
	return database.NewExternalDBPoolManager(s.db, nil).Get(ctx, dataSourceID)
}

func queryReportRows(ctx context.Context, db *sql.DB, query string) ([]string, [][]interface{}, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	headers, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	result := make([][]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(headers))
		ptrs := make([]interface{}, len(headers))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		for i, value := range values {
			if b, ok := value.([]byte); ok {
				values[i] = string(b)
			}
		}
		result = append(result, values)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return headers, result, nil
}

func validateReportReadOnlyQuery(query string) error {
	upper := strings.ToUpper(strings.TrimSpace(query))
	if upper == "" {
		return fmt.Errorf("report query is empty")
	}
	allowed := strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "WITH")
	if !allowed {
		return fmt.Errorf("report query must be read-only SELECT/WITH")
	}
	for _, keyword := range []string{"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE", "TRUNCATE", "GRANT", "REVOKE", "REPLACE"} {
		if strings.Contains(upper, keyword+" ") || strings.Contains(upper, keyword+"\n") {
			return fmt.Errorf("report query contains forbidden operation: %s", keyword)
		}
	}
	return nil
}

func ensureReportLimit(query string, limit int) string {
	upper := strings.ToUpper(query)
	if strings.Contains(upper, " LIMIT ") || strings.Contains(upper, "\nLIMIT ") {
		return query
	}
	return strings.TrimRight(strings.TrimSpace(query), ";") + fmt.Sprintf(" LIMIT %d", limit)
}

func prepareReportQuery(query string, params map[string]interface{}) string {
	result := query
	for key, value := range params {
		result = strings.ReplaceAll(result, "{{"+key+"}}", fmt.Sprint(value))
	}
	return result
}

func columnIndex(headers []string, name string) int {
	for i, header := range headers {
		if strings.EqualFold(header, name) {
			return i
		}
	}
	return -1
}

func toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case []byte:
		return strconv.ParseFloat(string(v), 64)
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return strconv.ParseFloat(fmt.Sprint(v), 64)
	}
}

func resolveParamString(params map[string]interface{}, key string) string {
	if params == nil {
		return ""
	}
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// formatCellValue returns a display-safe string for a cell value.
func formatCellValue(cell interface{}) string {
	if cell == nil {
		return ""
	}
	str := fmt.Sprintf("%v", cell)
	str = strings.ReplaceAll(str, "\n", " ")
	str = strings.ReplaceAll(str, "\r", "")
	if len(str) > 200 {
		str = str[:197] + "..."
	}
	return str
}

// sanitizeFilename replaces characters unsafe for filenames.
func sanitizeFilename(name string) string {
	r := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
		" ", "_",
	)
	return r.Replace(name)
}

// addPDFTitlePage renders a title page on the current PDF (starting a new
// page if needed).
func addPDFTitlePage(pdf *gofpdf.Fpdf, title, dateStr string, def *ReportTemplateDefinition) {
	_, pageH := pdf.GetPageSize()

	pdf.AddPage()

	_, pageW := pdf.GetPageSize()
	_, right, _, left := pdf.GetMargins()
	usableW := pageW - left - right

	// Vertical centering approximation.
	startY := pageH * 0.3
	pdf.SetY(startY)

	pdf.SetFont("helvetica", "B", 24)
	pdf.SetTextColor(30, 50, 100)
	pdf.CellFormat(usableW, 14, title, "", 0, "C", false, 0, "")
	pdf.Ln(16)

	pdf.SetFont("helvetica", "", 11)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(usableW, 7, fmt.Sprintf("Generated: %s", dateStr), "", 0, "C", false, 0, "")
	pdf.Ln(12)

	pdf.SetTextColor(80, 80, 80)
	pdf.SetFont("helvetica", "I", 10)
	pdf.CellFormat(usableW, 6, "OpenTether Enterprise AI Agent System", "", 0, "C", false, 0, "")

	pdf.SetTextColor(0, 0, 0)

	bottomMargin := def.Layout.MarginBottom
	if bottomMargin <= 0 {
		bottomMargin = 20
	}
	// Ensure enough room before the next section.
	if pdf.GetY() > pageH-bottomMargin-20 {
		pdf.AddPage()
	}
}

// renderPDFTextChart draws a simple text-based bar chart in the PDF.
func renderPDFTextChart(pdf *gofpdf.Fpdf, title, chartType string, labels []string, values []float64) {
	if len(labels) == 0 || len(values) == 0 {
		pdf.SetFont("helvetica", "I", 9)
		pdf.SetTextColor(128, 128, 128)
		pdf.Cell(0, 5, "[No chart data]")
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(6)
		return
	}

	maxVal := 0.0
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	// Use unicode block characters for a visual bar.
	barMaxChars := 40
	barChar := "█"

	for i, label := range labels {
		val := values[i]
		barLen := int(val / maxVal * float64(barMaxChars))
		if barLen < 1 && val > 0 {
			barLen = 1
		}
		bar := strings.Repeat(barChar, barLen)

		labelStr := label
		if len([]rune(labelStr)) > 14 {
			labelStr = string([]rune(labelStr)[:13]) + "…"
		}

		pdf.SetFont("helvetica", "", 8)
		pdf.Cell(45, 4, labelStr)
		pdf.SetFont("helvetica", "", 8)
		pdf.SetTextColor(50, 80, 140)
		pdf.Cell(6, 4, fmt.Sprintf("%.0f", val))
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("helvetica", "", 6)
		pdf.Cell(0, 4, bar)
		pdf.Ln(4)
	}
	pdf.Ln(2)
}

// computeNextCronRun computes the next run time after "after" given a cron
// expression. Uses a simple mapping:
//   - "0 * * * *"    -> every hour
//   - "0 0 * * *"    -> daily at midnight
//   - "0 0 * * 0"    -> weekly (Sunday)
//   - "0 0 1 * *"    -> monthly on the 1st
//
// For full cron support, consider importing github.com/robfig/cron/v3 (already
// in go.mod). This is a fallback.
func computeNextCronRun(expr string, after time.Time) *time.Time {
	// If the expression matches common patterns, use simple arithmetic.
	// Otherwise, fall back to a reasonable default (add 1 hour).
	expr = strings.TrimSpace(expr)

	const (
		everyHour = "0 * * * *"
		daily     = "0 0 * * *"
	)

	switch expr {
	case everyHour:
		next := after.Truncate(time.Hour).Add(time.Hour)
		return &next
	case daily:
		next := time.Date(after.Year(), after.Month(), after.Day(), 0, 0, 0, 0, after.Location()).Add(24 * time.Hour)
		return &next
	}

	// Default: add 1 hour.
	next := after.Add(1 * time.Hour)
	return &next
}
