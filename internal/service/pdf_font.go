package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

func initCJKFont(pdf *gofpdf.Fpdf) string {
	fontPaths := findCJKFont()
	if fontPaths.regular != "" {
		pdf.AddUTF8Font("cjk", "", fontPaths.regular)
		if fontPaths.bold != "" {
			pdf.AddUTF8Font("cjk", "B", fontPaths.bold)
		}
		return "cjk"
	}
	pdf.AddUTF8Font("cjk", "", "helvetica")
	return "cjk"
}

type cjkFontPaths struct {
	regular string
	bold    string
}

func findCJKFont() cjkFontPaths {
	searchDirs := []string{
		"data",
		"data/fonts",
		"/usr/share/fonts",
		"/usr/share/fonts/opentype/noto",
		"/usr/share/fonts/truetype",
		"/usr/share/fonts/truetype/noto",
		"/usr/share/fonts/truetype/droid",
		"/usr/share/fonts/truetype/wqy",
		"/Library/Fonts",
		"/System/Library/Fonts",
		"/System/Library/Fonts/Supplemental",
		"C:\\Windows\\Fonts",
	}
	patterns := []string{
		"NotoSansSC*.ttf",
		"NotoSansCJK*.ttf",
		"NotoSansCJK*.ttc",
		"wqy*.ttf",
		"WenQuanYi*.ttf",
		"DroidSansFallback*.ttf",
		"simsun*.ttf",
		"SimSun*.ttf",
		"simhei*.ttf",
		"SimHei*.ttf",
		"msyh*.ttf",
		"Microsoft YaHei*.ttf",
		"PingFang*.ttf",
		"STSong*.ttf",
		"Songti*.ttf",
		"Hiragino*.ttf",
	}
	var regular, bold string
	for _, dir := range searchDirs {
		for _, pattern := range patterns {
			matches, err := filepath.Glob(filepath.Join(dir, pattern))
			if err != nil {
				continue
			}
			for _, match := range matches {
				base := strings.ToLower(filepath.Base(match))
				if regular == "" && !strings.Contains(base, "bold") && !strings.Contains(base, "b.") {
					regular = match
				}
				if bold == "" && (strings.Contains(base, "bold") || strings.Contains(base, "b.")) {
					bold = match
				}
			}
		}
	}
	return cjkFontPaths{regular: regular, bold: bold}
}

func pdfFont(path string) string {
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

var _ = fmt.Println
var _ = gofpdf.New("P", "mm", "A4", "")
