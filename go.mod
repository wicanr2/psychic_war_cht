module github.com/wicanr2/psychic_war_cht

go 1.24.0

toolchain go1.24.13

require (
	github.com/hajimehoshi/ebiten/v2 v2.9.9
	github.com/wicanr2/dosgolem v0.0.0-00010101000000-000000000000
)

require (
	github.com/ebitengine/gomobile v0.0.0-20250923094054-ea854a63cce1 // indirect
	github.com/ebitengine/hideconsole v1.0.0 // indirect
	github.com/ebitengine/oto/v3 v3.4.0 // indirect
	github.com/ebitengine/purego v0.9.0 // indirect
	github.com/jezek/xgb v1.1.1 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
)

// dosgolem 用本機分支（worktrees/dosgolem，psychic-war/r4-live 起）：前端需要規格 199 的即時執行介面。
replace github.com/wicanr2/dosgolem => ./worktrees/dosgolem
