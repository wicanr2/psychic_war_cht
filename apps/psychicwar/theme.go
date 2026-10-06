package psychicwar

// 前端的資料目錄解析沿用 paths.go；無視窗工具直接使用 theme 套件。
import "github.com/wicanr2/psychic_war_cht/apps/psychicwar/theme"

type Theme = theme.Theme

func LoadTheme(selection, orig string, scale int) (*Theme, string, error) {
	return theme.LoadTheme(selection, orig, DataDir("theme"), scale)
}

func PremultiplyRGBA(pix []byte) { theme.PremultiplyRGBA(pix) }
