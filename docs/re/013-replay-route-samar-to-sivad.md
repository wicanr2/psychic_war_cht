# 013：重播路線：Samar 到 Sivad

狀態：量測紀錄（2026-09-17）。對應 issue #8（全程重播）。重播檔 `replay/title-to-first-save.json`，格式 `docs/spec/003`。

## 1. 結論

- 重播推進到 **18 段**：開機 → 第一場戰鬥 → 存讀檔 → Samar 內再打三場（Kasuruji、Minton ×2）與治療 → 搭 Launch Pad 抵達 **Sivad**（區域 1）→ 在 Sivad 存讀檔。
- 全部以正常按鍵與按住按鍵完成，**不改記憶體**。改記憶體只用在探路（`docs/re/011` §3.4）。
- 換區域的開檔證據：`16-sivad` 開了 `I_MAP01.BIN`、`CODE1.BIN`、`I_MENU01.BIN`、`ENEMY01.PBL`、`I_ENMY01.BIN`；區域編號 `0x16966` 由 0 變 1。
- 新路段沒實作的服務：0 種（每段 probe 摘要）。
- 讀檔往返兩次：`11-loaded`（畫面與記憶體都與 `09-battle-won` 相同）、`18-loaded2`（記憶體與 `17-saved2` 相同；畫面訊息區不同，見 §3）。

## 2. 各段

指令數是 dosgolem 預設時鐘（指令數時鐘）。座標 (X, Y)、朝向見 `docs/re/011`。按鍵：up 前進、right／left 轉 90°、down 向後轉。

| 段 | 從 | 目的與按鍵 | 遇到 | 結果 |
|---|---|---|---|---|
| 01–08 | — | 標題 → 防拷 → 輸入名字 kai → 往北走（`docs/re/008`） | 第 6 次移動遇 Shulosu | — |
| 09-battle-won | 08 | 按住空白鍵 3,000 ms | Shulosu（HP 38） | 打贏，玩家 40 |
| 10-saved | 09 | Esc → Options → Save Game → TEST | — | 存檔 |
| 11-loaded | 05 | 主選單讀 TEST | — | 畫面與記憶體 ＝ 09 |
| 12-battle2 | 10 | 從 (1,8) 往北 7、往東 6、往南 2，走進 (7,3)；第 238,000,000 步起按住空白鍵 3,000 萬步 | Kasuruji（HP 70，(7,3) 固定遭遇格） | 打贏，玩家 40 → 23、能量 → 1 |
| 13-healed | 12 | 往北 2、往東 2 到 (9,1)，往南進 Bio Room (9,2)，Enter 選 Get healed | — | HP 40、能量 30，得 Yontry 3 瓶 |
| 14-minton1 | 13 | 出房到 (9,1)，往東 5 步；遭遇後第 382,200,000 步起按住空白鍵 | Minton（HP 42），在 (14,1) | 打贏，玩家 40、能量 13 |
| 15-minton2 | 14 | 往南 8 步；第 469,200,000 步起按住空白鍵 | Minton（HP 42），在 (14,9) | 打贏，玩家 22、能量 0 |
| 16-sivad | 15 | 往南 5、往東 1 到 Launch Pad (15,14)，down 選 Sivad、Enter | — | 抵達 Sivad 降落平台，區域 0 → 1 |
| 17-saved2 | 16 | up、Esc → Options（第 5 項）→ Save Game → TEST2 | — | 存檔 |
| 18-loaded2 | 05 | 主選單讀 TEST2 | — | 記憶體 ＝ 17：區域 1、(5,6)、朝北、HP 28、能量 6 |

## 3. 走過的坑（寫成規則）

- **遭遇看步數，不看位置**：新遊戲或讀檔後第 6 次移動必定遇 Shulosu；戰後約 8–10 步會再遇 Minton，換路線、換按鍵間隔都避不開。
  所以讀檔後不能馬上走長路（`TEST2` 讀檔後若再走，第 6 步又會遭遇）。
- **HP、能量不夠就打不贏**：Kasuruji 戰後（HP 23、能量 1）直接打 Minton 輸；讀檔後帶傷打 Shulosu 也輸。Bio Room 治療後才打贏。每走一步 HP、能量各回 1。
- **選單會變**：拿到 Yontry 後 Esc 選單多出 Sip Yontry 等項，Options 從第 4 項變第 5 項。
- **降落平台剛抵達時 Esc 不開選單**，按鍵被當成轉向；先按一次 up。
- **畫面相等斷言不適用所有存讀檔**：在降落平台存檔時訊息區留著「This is Sivad's Landing Pad.」，讀檔後是空的。
  重播格式加了 `expect_same_memory_as`（`docs/spec/003`），比 `memory_ranges`（`lin:16966:52`：區域、座標、朝向、地點、角色數值）。
- **路線不能只靠地圖檔推**：`I_MAP` 的 256 bytes 牆壁格位元意義未解，路線是用探路掃描（`docs/re/011` §3.4）建出通路圖再 BFS。
  掃描只測「走廊 → 走廊」；走進固定遭遇格、房間的最後一步要另外測（(3,1) 往南進 (3,2) 被擋）。

## 4. 限制

- Sivad 的碰撞座標偏移不是固定值（Samar 是 X＋6、Y＋2），探路掃描在 Sivad 還不能用；往 Sivad 內推進要先解這個。
- 重播只在 dosgolem 預設時鐘上驗證；以 750 cycles 跑同一份按鍵，亂數與遭遇可能不同（`docs/re/010` §4.2）。
