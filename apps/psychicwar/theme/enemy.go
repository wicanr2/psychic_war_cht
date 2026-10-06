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

func enemySource(name string, n int) (string, []int, error) {
	var hash string
	var edges map[int][]int
	switch name {
	case "ENEMY00.PBL":
		hash = "8241b0ea73b1e7402b13adc434f10a5b1288f88f01fcfc3ba5710e4923fe6067"
		edges = map[int][]int{0: {1}, 1: {0, 2}, 2: {1}, 3: {4}, 4: {3, 5}, 5: {4}, 6: {7}, 7: {6, 8}, 8: {7}, 9: {10}, 10: {9, 11}, 11: {10}, 12: {13}, 13: {12, 14}, 14: {13}}
	case "ENEMY01.PBL":
		hash = "22f664050ce7ffa4ea0f6941c9c91cd1ab43671ea5b53491f6799f78ba8e64af"
		// 024 §1.22：五組來源及四階段分支皆由原版指令／保存資料核對。
		edges = map[int][]int{0: {1}, 1: {0, 2}, 2: {1}, 3: {4}, 4: {3, 5}, 5: {4}, 6: {7}, 7: {6, 8}, 8: {7}, 9: {10}, 10: {9, 11}, 11: {10}, 12: {13}, 13: {12, 14}, 14: {13}}
	case "ENEMY03.PBL":
		hash = "ad6e8183bbf6c6fc258693b1f0ac726593feff5d052b5da18ac715cc2e63e5c1"
		edges = map[int][]int{0: {1}, 1: {0, 2}, 2: {1}, 3: {4}, 4: {3, 5}, 5: {4}, 6: {7}, 7: {6, 8}, 8: {7}, 9: {10}, 10: {9, 11}, 11: {10}, 12: {13}, 13: {12, 14}, 14: {13}}
	case "ENEMY04.PBL":
		hash = "81cb62cf9a8b64a538d09e50b29cbf8ad123b39b6e86979af6423a2aed20f546"
		// 024 §1.23：反向邊由原版四階段分支與正常保存圖庫證實。
		edges = map[int][]int{0: {1}, 1: {0, 2}, 2: {1}, 3: {4}, 4: {3, 5}, 5: {4}, 6: {7}, 7: {6, 8}, 8: {7}, 9: {10}, 10: {9, 11}, 11: {10}, 12: {13}, 13: {12, 14}, 14: {13}}
	default:
		return "", nil, fmt.Errorf("未支援敵人來源 %q", name)
	}
	from, ok := edges[n]
	if !ok {
		return "", nil, fmt.Errorf("%s 圖號 #%d 尚未有READY契約", name, n)
	}
	return hash, from, nil
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
	if err != nil || w != 24 || h != 32 {
		return nil, fmt.Errorf("主題 %s #%d 尺寸或解碼不符", name, n)
	}
	s := newSprite(px, 32, 152, w, h)
	// 024 §1.22–§1.23：enemySource已限制四檔各五組完整來源與有向差分。
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
