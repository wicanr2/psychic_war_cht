package theme

// 已證實原圖及有向差分；既有實跑見研究§48／50／90／142–143，共用分支及圖庫見§146–147。
import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/psychic_war_cht/apps/psychicwar/pbl"
)

func loadEnemy0(orig string, n int) (*spritePresence, error) {
	return loadEnemy(orig, "ENEMY00.PBL", n)
}

// §1.47：實際原版C6資料與共用階段分支限定來源；正常GUI另驗。
var enemySourceHashes = map[string]string{
	"ENEMY00.PBL": "8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067",
	"ENEMY01.PBL": "22f664050ce7ffa4ea0f6941c9c91cd1ab43671ea5b53491f6799f78ba8e64af",
	"ENEMY02.PBL": "c38beb2879508466f0c316185eb7a489071279c34c2a89678bb1e80598de4c3f",
	"ENEMY03.PBL": "ad6e8183bbf6c6fc258693b1f0ac726593feff5d052b5da18ac715cc2e63e5c1",
	"ENEMY04.PBL": "81cb62cf9a8b64a538d09e50b29cbf8ad123b39b6e86979af6423a2aed20f546",
	"ENEMY05.PBL": "2b390a5c4a5a2c6e49a9e27b02c89dd839c6c932dad0f568aee6b88e2397180a",
	"ENEMY06.PBL": "489364bda2f9356d3b386ae71ad90cfc6064a1e04d33a5569e5e653a27d220a7",
	"ENEMY07.PBL": "c2924057e1d704d30be7a644c870b0879bb72e155183b6ab7c4c6e864904d519",
	"ENEMY08.PBL": "eb4e8673858a5425bba68edba74cad1138caf2c64d58d429e40c7d7a7ec072a7",
	"ENEMY09.PBL": "f6e09a218300e9848360493ecac122617639475106753350944ef078b91e71d9",
	"ENEMY10.PBL": "d371d4065a76a374329128c6d4a35f52f578322f7f2e4458252592b26fa0a482",
	"ENEMY11.PBL": "f5c29f254baf0ec1ef2dc9db61596efbd2cfb941ed992534358899386a29dc44",
}

func knownEnemySource(name string) bool {
	_, ok := enemySourceHashes[name]
	return ok
}

func enemySource(name string, n int) (string, []int, error) {
	hash, ok := enemySourceHashes[name]
	if !ok {
		return "", nil, fmt.Errorf("未支援敵人來源 %q", name)
	}
	limit := 15
	if n < 0 || n >= limit {
		return "", nil, fmt.Errorf("%s 圖號 #%d 尚未有READY契約", name, n)
	}
	// 原版四階段：3g→3g+1→3g+2→3g+1→3g；不是依相鄰圖號推測。
	switch n % 3 {
	case 0:
		return hash, []int{n + 1}, nil
	case 1:
		return hash, []int{n - 1, n + 1}, nil
	default:
		return hash, []int{n - 1}, nil
	}
}

func loadEnemy(orig, name string, n int) (*spritePresence, error) {
	expectedHash, fromImages, err := enemySource(name, n)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(orig, name))
	if err != nil {
		return nil, fmt.Errorf("主題敵人來源：%w", err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(b)) != expectedHash {
		return nil, fmt.Errorf("主題 %s 的 SHA-256 不符", name)
	}
	offsets, err := pbl.Offsets(b)
	if err != nil || len(offsets) != 30 {
		return nil, fmt.Errorf("主題 %s 圖數不符", name)
	}
	w, h, px, err := pbl.Decode(b, n)
	if err != nil || w != 24 || h != enemyHeight(name, n) {
		return nil, fmt.Errorf("主題 %s #%d 尺寸或解碼不符", name, n)
	}
	s := newSprite(px, 32, 152, w, h)
	if h == 24 {
		s.nativeH = 32 // 原版多讀區不參與身體辨識，024 §1.51。
	}
	// 024 §1.47：enemySource限制58組完整唯一來源與有向差分。
	s.retainDelta = true
	// 每條邊由原版呼叫／畫面，或限定的原版分支／來源資料證實；不由圖號相鄰或 XOR 性質猜補。
	for _, from := range fromImages {
		pw, ph, prev, err := pbl.Decode(b, from)
		if err != nil || pw != w || ph != h {
			return nil, fmt.Errorf("主題 %s 前一動作尺寸或解碼不符", name)
		}
		delta := spriteDelta{from: prev, packed: make([]byte, len(s.packed))}
		for i := range delta.packed {
			delta.packed[i] = (prev[i*2]<<4 | prev[i*2+1]) ^ s.packed[i]
		}
		s.deltas = append(s.deltas, delta)
	}
	return s, nil
}

func enemyHeight(name string, n int) int {
	if name == "ENEMY08.PBL" && n >= 12 && n <= 14 {
		return 24
	}
	return 32
}
