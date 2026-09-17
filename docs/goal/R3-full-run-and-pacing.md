# 第 3 輪：往全程推進、定案 CPU 速度

## 目標

**把重播從第一場戰鬥往後推進到離開起始區域，並定案給人玩的 CPU 速度。**
全程可跑（#8）是整個移植的地基；CPU 速度（#9）決定遊戲能不能玩、難度對不對。
這一輪同時補齊「往後推進」需要的觀測工具：位置、區域、角色狀態的記憶體位址。

使用者定案：**dosgolem 的修改在整個遊戲移植完成後一次發 PR**，這一輪繼續疊在本機分支上。

## 起點（第 2 輪留下的東西）

| 產物 | 位置 |
|---|---|
| 重播檔 11 段：開機 → 按住打贏第一場戰鬥 → 存讀檔 | `replay/title-to-first-save.json`、`tools/states.sh --check` |
| 按住按鍵、暫存層 | dosgolem 規格 `197`、probe `-hold`、`-scratch` |
| 敵人 HP 位址 `0x509C` | `docs/re/009` §3 |
| 戰鬥長短綁 CPU 指令數 | `docs/re/009` §5 |
| 決定性、亂數 | `docs/re/004`、`tools/determinism.sh` |
| 色盤、PC 喇叭已與 DOSBox-X 對齊 | `docs/re/005`、`006` |
| OPL2 雛形（比對未通過） | `docs/re/007` |

dosgolem 本機分支鏈（依序疊加，都未推上游）：
`psychic-war/m1-ega-palette` → `m1-pit-tone` → `m1-opl2`（含 `-scratch`）→ `m1-key-hold`（含 `-cpuhz` 修正）。

## 本輪完成條件

以下全部成立才算完成：

1. **CPU 速度（#9）**
   - 讀出敵人攻擊的時間基準（計時器或 CPU），寫進 `docs/re/`，附位址與證據等級。
   - DOSBox-X 以至少兩種固定 `cycles` 量第一場戰鬥（按住攻擊）的持續時間與玩家損失 HP，當參照。
   - 規格 READY：給人玩的 `CPUHz` 目標值與依據；dosgolem 在該 `CPUHz` 下同一場戰鬥的持續時間（以計時器刻數換算秒）與參照相差 ≤ 10%，玩家損失 HP 與參照相符。
2. **觀測位址（#8、#33 的前置）**：玩家 HP、能量、目前位置（座標或格子）、朝向、區域編號的記憶體位址，
   每一個都用「改值後畫面或行為跟著變」驗證，寫進 `docs/re/`。
3. **全程推進（#8）**
   - 重播往後推進到**離開起始區域**（載入第二張地圖或 `I_MAP01` 之類的新區域檔，以開檔紀錄為證），途中至少再打贏一場戰鬥、做一次存讀檔往返。
   - 探路時可以改記憶體（位置、HP）加速找路線；**正式重播不得含改記憶體**，全部以正常輸入完成，`tools/states.sh --check` 通過。
   - 新路段沒實作的服務為 0 種；若出現，補 dosgolem（依 DOSBox-X 原始碼）並寫規格。
   - 路線寫成文件（每一段的目的、按鍵、遇到的敵人與結果）。
4. **OPL2 事件層（#5）**：DOSBox-X 的 raw OPL 擷取與 dosgolem `-opl-log` 逐筆比對標題音樂，結果寫進 `docs/re/`（通過或不通過都寫）。
5. `tools/py.sh tools/worklist.py verify` 沒有「可能已完成」；做完的條目移到 `done`；本機 commit。

## 量測指標

- 第一場戰鬥持續時間：dosgolem（目標 `CPUHz`）vs DOSBox-X 參照，差 ≤ 10%。
- 重播段數與涵蓋的區域檔；新路段沒實作的服務數：0。
- 每個位址的驗證方式與證據等級。
- OPL2 事件層：逐筆不一致數。

## 工作邊界

- SDD：改 dosgolem 前先寫規格標 READY；本 repo 的格式與方法寫 `docs/spec/`。
- dosgolem 修改只疊在 `worktrees/dosgolem` 的本機分支，**不 push、不開 PR**；每次改完跑 dosgolem 全套測試。
  碰到 EGA／VGA 顯示的修改，另外用 `worktrees/eob1-base`／`eob1-pal` 的方式跑 eob1 真實資料回歸（`docs/re/009` §6）。
- 全部走 docker；只清理自己建立的 container，禁止 prune／`rmi`。
- 原版與說明書掃描只在 `workplace/`；文件只摘要說明書，不轉錄。
- 防拷：依老遊戲保存例外，可讀判定邏輯；略過功能屬 M5，不在本輪。
- **issue 狀態與留言可以直接更新**（使用者 2026-09-17 授權）；**push 仍要先問**。

## 收尾

- 每項完成：證據進 `docs/re/` 或 `docs/spec/`，worklist 移到 `done` 或更新現況，`render`，本機 commit。
- 在對應 issue 留言進度；完成的關閉。
- 回報附數字：戰鬥時間與參照的差、重播段數與區域、各位址證據、OPL2 事件比對結果。
- 產生下一輪的 goal markdown。
