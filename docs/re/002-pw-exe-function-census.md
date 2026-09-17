# 002：PW.EXE／LOGO.EXE 函式普查

狀態：量測紀錄（2026-09-17）。對應 issue #1。

## 1. 結論

- **`PW.EXE` 是 Microsoft EXEPACK 壓縮的執行檔。** 直接丟進 IDA 分析的是壓縮狀態，函式與位址全部不能用。
  本文所有 `PW` 的數字都來自解壓後的 `PW_UNP.EXE`。
- 解壓後：**419 個函式**，程式碼 38,592 bytes；**93 處 `INT` 指令**。
- 函式分三群：遊戲本體 263 個、Microsoft C 啟動碼與程式庫 61 個（強推論）、其他組合語言模組 95 個
  （含 Covox 音效函式庫，範圍是假說）。
- 從開機走到「輸入名字」的這段實跑中，執行過至少一道指令的函式有 **142／419**。
- `LOGO.EXE` 沒有壓縮：16 個函式，執行過 14 個。

## 2. 輸入

| 檔案 | SHA-256 | 說明 |
|---|---|---|
| `PW.EXE` | `88321206d5400b2276aba0f268e2daaa8355a733843dd9e89f9e7116ae690c49` | 原版（EXEPACK） |
| `PW_UNP.EXE` | `fd5115b91f014c47a1fb1a6e2ecfd645e293264fe614a4b350e92637cad2fbd9` | `tools/unexepack.py` 輸出 |
| `PW_UNP.EXE.i64` | `4db51a199f5913c52dc4964557b9a4aafb965754d663cb0173c9dfdfe9a00c56` | IDA 9.4，`ida-pro-9.4-idapython:locked-v1` |
| `LOGO.EXE` | `08a4e38b051c29381296fdf3d075eac55dc215d52cf43b9caf955c8deccfc1ba` | 原版（未壓縮） |
| `LOGO.EXE.i64` | `c40bc87022f79257ab8fe3bee1a9a74f3ff268c7ea72e60eb7b21cffa96a0caa` | 同上 image |
| `cov.json` | `5ea6c74ea53d80b58a6dbc448bd338011776cb713659d2d1e2c6b4b3521549bd` | dosgolem `d9c0c27` 覆蓋率，4 億步，走到輸入名字（docs/re/001 §2） |

所有產物在 `workplace/`（gitignore）。

## 3. EXEPACK

### 3.1 證據

| 觀察 | 值 |
|---|---|
| MZ 重定位數 | 0 |
| MZ 進入點 | `0E00:0010`，映像尾端 |
| `0E00:0000` 的標頭 | 16 bytes 版本，簽章 `RB` 在 +14 |
| 真正的進入點／堆疊 | `0051:B7C6`／`1F40:0800` |
| 解壓後大小 | `1B9D` 段 × 16 ＝ 113,104 bytes |
| 解壓器錯誤訊息 | `Packed file is corrupt`（檔案位移 `0xE30F`） |
| 重定位表 | 234 筆，結束位置剛好等於 EXEPACK 區段長 `0x319` |

### 3.2 驗證

`tools/unexepack.py` 依公開的 EXEPACK 格式靜態解壓，輸出帶重定位表的 MZ。
另外用 dosgolem 跑原版，在第 5,591 步（第一次執行到 `0161:B7C6`）傾印 `lin 01100` 起 113,104 bytes：

```sh
tools/py.sh tools/unexepack.py workplace/original/psychic-war/PW.EXE workplace/unpacked/PW_UNP.EXE \
    --verify workplace/probe/pw-unpacked-at-entry.bin 110
# 與傾印比對 113104 bytes：不一致 0 處
```

靜態解壓套上載入段 `0110h` 的重定位後，與執行期解壓器的結果**逐位元組相同**。等級：confirmed。

### 3.3 位址換算

| 空間 | 換算 |
|---|---|
| IDA ea（`PW_UNP.EXE.i64`） | `10000h` ＋ 映像位移 |
| dosgolem 執行期線性（PSP `0100`，映像段 `0110`） | `01100h` ＋ 映像位移 |
| IDA ea ↔ 執行期線性 | ea ＝ 線性 ＋ `EF00h` |
| `PW_UNP.EXE` 檔案位移 | 映像位移 ＋ `3D0h`（標頭 61 段） |

⚠ docs/re/001 §4.1 的 `CD 75` 計數與 `INT` 計數是在**壓縮檔**上掃的，不能拿來對應本文的位址。

## 4. 方法

1. `tools/ida.sh build` 建庫（自動分析）。
2. `tools/ida/census.py`：匯出區段、函式、每一處 IDA 解成指令的 `int`、字串數。
3. 把覆蓋率中「執行過、IDA 還不是程式碼」的位址種成指令，再普查一次。
4. `tools/census_report.py` 合併覆蓋率，算執行過的函式數與 `INT` 呼叫點表。

dosgolem 的覆蓋率記的是**指令起點**，不是 byte；「執行過的函式」＝函式範圍內至少有一個指令起點被執行。

## 5. 結果

### 5.1 PW_UNP.EXE 總量

| 項目 | 加種子前 | 加種子後 |
|---|---:|---:|
| 函式 | 417 | 419 |
| 程式碼 bytes | 38,511 | 38,592 |
| `INT` 指令 | 93 | 93 |
| 字串 | 38 | 40 |

種子只新增 23 道指令、2 個函式，表示 IDA 自動分析已涵蓋這次實跑走過的路徑。
覆蓋率中有 600 個執行過的指令起點落在函式範圍外，推定是中斷處理常式等未建成函式的程式碼（假說）。

### 5.2 分群

| 群 | IDA 範圍 | 函式 | 執行過 | 函式 bytes | `push bp; mov bp,sp` 開頭 | 等級 |
|---|---|---:|---:|---:|---:|---|
| 遊戲本體 | `10000–1BCD5` | 263 | 97 | 17,874 | 8 | 強推論 |
| Microsoft C 啟動碼＋程式庫 | `1BCD6–1DAC5` | 61 | 23 | 7,209 | 37 | 強推論 |
| `seg003` | `1DAC6–1ECF8` | 39 | 22 | 3,437 | 10 | 未知（含 `INT 08h`／`15h`／`1Ah`，推定計時器與系統服務） |
| `seg004`–`seg005`（Covox） | `1ECF9–1F0B0` | 10 | 0 | 644 | 0 | 假說 |
| `seg006`–`seg015` | `1F0B1–2064F` | 46 | 0 | 2,762 | 13 | 未知（這次實跑沒有執行，可能是 AdLib 或 Covox 路徑） |
| 合計 | | **419** | **142** | 31,926 | 68 | |

分群依據：

- **Microsoft C runtime（強推論）**
  - `start`（`1BCD6`）是 MSC 啟動碼的形狀：`AH=30h` 查 DOS 版本、小於 2 就 `INT 20h`、以 `seg dseg` 設堆疊段。
  - DGROUP 開頭（`20658`）是 `MS Run-Time Library - Copyright (c) 1988, Microsoft Corp`，是 MSC 的慣例位置。
  - 映像中有 `R6xxx` 系列執行期錯誤訊息。
  - MSC 把啟動碼與程式庫目的檔連結在使用者目的檔之後；`1BCD6` 起到段尾的函式有 37／61 使用 BP 框架，
    與前段遊戲碼（8／263）差異明顯。
  - 精確的編譯器版本未證實：版權年份只能給出 1988 年的程式庫範圍。
  - 這套 IDA 沒有 16 位元 MSC 的 FLIRT 簽章，IDA 標 `FUNC_LIB` 的函式數為 0，逐一命名留待需要時再做。
- **Covox 音效函式庫（假說）**：`1F058` 有 `Covox`／`Sound Libraries, Copyright 1989 Covox Inc.` 字串，
  落在 `seg004` 尾端；函式的確切歸屬沒有逐一驗證。
- **遊戲本體（強推論）**：`start` 之前的程式碼以暫存器保存（`push ax/bx/cx/dx`）與 `cmp byte cs:[…]` 開頭為主，
  是手寫組合語言的樣式。哪些函式引用遊戲文字還沒查（issue #15）。

### 5.3 被呼叫最多的函式（前 8）

| 函式 | 呼叫端 | 大小 | 開頭 bytes |
|---|---:|---:|---|
| `sub_18645` | 29 | 76 | `50 53 51 52 56 2e 80 3e` |
| `sub_20619` | 25 | 41 | `53 51 52 32 ff 8a dc 88` |
| `sub_1FC4F` | 24 | 43 | `50 53 51 52 b9 0a 00 86` |
| `sub_10ABA` | 19 | 26 | `51 b9 dc 05 e2 fe e8 71` |
| `sub_14F96` | 18 | 23 | `53 8b 1e e6 ae 8a 47 46` |
| `sub_16623` | 18 | 6 | `2e 89 1e 0e 61 c3` |
| `sub_1ACC2` | 18 | 57 | `2e 80 3e 2e 81 02 75 06` |
| `sub_18B76` | 16 | 159 | `50 53 51 52 56 57 53 1e` |

語意都還沒讀；呼叫次數高只是解讀的優先順序，不是 helper 的證據。

### 5.4 `INT` 呼叫點摘要

| 中斷 | PW 全部 | PW 這次執行過 | LOGO 全部 | LOGO 這次執行過 |
|---|---:|---:|---:|---:|
| `08h` | 1 | 1 | 0 | 0 |
| `10h` | 9 | 3 | 6 | 0 |
| `15h` | 3 | 0 | 0 | 0 |
| `1Ah` | 5 | 0 | 0 | 0 |
| `20h` | 1 | 0 | 0 | 0 |
| `21h` | 74 | 31 | 2 | 2 |
| 合計 | **93** | **35** | **8** | **2** |

`INT 1Ah` 的 5 處都在 `seg003`，這次沒有執行；亂數來源由 issue #7 追。

## 6. LOGO.EXE

| 項目 | 值 |
|---|---|
| 函式 | 16（執行過 14） |
| 程式碼／資料 | 3,160／19,720 bytes |
| `INT` | 8（`10h` 6、`21h` 2） |
| 執行期載入 | PSP `2177`，映像段 `2187`（EXEC 紀錄）；覆蓋率 623 個指令起點全部落在函式內 |
| runtime | 沒有 BP 框架函式、沒有字串，推定是組合語言小程式（假說） |

## 7. 限制與重開條件

- 覆蓋率只涵蓋開機到「輸入名字」、沒有 AdLib 的路徑。M1 #8 的全程重播完成後要重算「執行過的函式」。
- runtime 與遊戲碼的邊界是以 `start` 位址切分；若之後在 `1BCD6` 之後找到遊戲專屬邏輯，要回來修正分群。
- `LOGO.EXE` 的第一次普查執行沒有產出 JSON，重跑同一條命令才成功，原因未查。
  之後批次跑 IDA 時一律驗輸出檔，不看 exit code。

## 附錄 A：PW_UNP.EXE 全部 `INT` 呼叫點

「段:位移」以段基底計，與執行期 `CS:IP` 的位移相同（`seg002` → `0161`、`seg001` → `0143`、`seg000` → `0110`）。

| IDA 位址 | 段:位移 | 中斷 | bytes | 函式 | 這次執行過 | 反組譯 |
|---|---|---|---|---|---|---|
| `1041D` | `seg001:00ED` | `21h` | `cd21` | sub_10403 | 是 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1046D` | `seg001:013D` | `10h` | `cd10` | sub_1044D | 是 | `int 10h; - VIDEO - ALTERNATE FUNCTION SELECT (PS, EGA, VGA, MCGA) - GET EGA INFO` |
| `104D0` | `seg001:01A0` | `21h` | `cd21` | — | 否 | `int 21h; DOS - 3+ - GET PSP ADDRESS` |
| `10645` | `seg002:0135` | `10h` | `cd10` | sub_1053C | 否 | `int 10h; - VIDEO - SET VIDEO MODE` |
| `17650` | `seg002:7140` | `10h` | `cd10` | sub_1AB5B | 是 | `int 10h; - VIDEO - SET VIDEO MODE` |
| `17685` | `seg002:7175` | `10h` | `cd10` | sub_1767D | 是 | `int 10h; - VIDEO - SET PALETTE REGISTER (Jr, PS, TANDY 1000, EGA, VGA)` |
| `17C4B` | `seg002:773B` | `10h` | `cd10` | sub_1AB5B | 否 | `int 10h; - VIDEO - SET VIDEO MODE` |
| `17C54` | `seg002:7744` | `10h` | `cd10` | sub_1ABA3 | 否 | `int 10h; - VIDEO - SET COLOR PALETTE` |
| `18075` | `seg002:7B65` | `10h` | `cd10` | sub_1ABA3 | 否 | `int 10h; - VIDEO - SET COLOR PALETTE` |
| `184DD` | `seg002:7FCD` | `10h` | `cd10` | sub_1AB5B | 否 | `int 10h; - VIDEO - SET VIDEO MODE` |
| `18512` | `seg002:8002` | `10h` | `cd10` | sub_1850A | 否 | `int 10h; - VIDEO - SET PALETTE REGISTER (Jr, PS, TANDY 1000, EGA, VGA)` |
| `188A9` | `seg002:8399` | `21h` | `cd21` | sub_18884 | 是 | `int 21h; DOS - 2+ - OPEN DISK FILE WITH HANDLE` |
| `188BA` | `seg002:83AA` | `21h` | `cd21` | sub_18884 | 是 | `int 21h; DOS - 2+ - READ FROM FILE WITH HANDLE` |
| `188BE` | `seg002:83AE` | `21h` | `cd21` | sub_18884 | 是 | `int 21h; DOS - 2+ - CLOSE A FILE WITH HANDLE` |
| `1890C` | `seg002:83FC` | `21h` | `cd21` | sub_188E1 | 否 | `int 21h; DOS - 2+ - FIND FIRST ASCIZ (FINDFIRST)` |
| `18936` | `seg002:8426` | `21h` | `cd21` | sub_1892C | 否 | `int 21h; DOS - 2+ - OPEN DISK FILE WITH HANDLE` |
| `18941` | `seg002:8431` | `21h` | `cd21` | sub_1892C | 否 | `int 21h; DOS - 2+ - READ FROM FILE WITH HANDLE` |
| `18945` | `seg002:8435` | `21h` | `cd21` | sub_1892C | 否 | `int 21h; DOS - 2+ - CLOSE A FILE WITH HANDLE` |
| `1899D` | `seg002:848D` | `21h` | `cd21` | sub_18991 | 否 | `int 21h; DOS - 2+ - CREATE A FILE WITH HANDLE (CREAT)` |
| `189A9` | `seg002:8499` | `21h` | `cd21` | sub_18991 | 否 | `int 21h; DOS - 2+ - WRITE TO FILE WITH HANDLE` |
| `189B4` | `seg002:84A4` | `21h` | `cd21` | sub_18991 | 否 | `int 21h; DOS - 2+ - CLOSE A FILE WITH HANDLE` |
| `189E5` | `seg002:84D5` | `21h` | `cd21` | sub_189BF | 是 | `int 21h; DOS - 2+ - OPEN DISK FILE WITH HANDLE` |
| `189F3` | `seg002:84E3` | `21h` | `cd21` | sub_189BF | 是 | `int 21h; DOS - 2+ - READ FROM FILE WITH HANDLE` |
| `18A0E` | `seg002:84FE` | `21h` | `cd21` | sub_189BF | 是 | `int 21h; DOS - 2+ - MOVE FILE READ/WRITE POINTER (LSEEK)` |
| `18A18` | `seg002:8508` | `21h` | `cd21` | sub_189BF | 是 | `int 21h; DOS - 2+ - READ FROM FILE WITH HANDLE` |
| `18A1E` | `seg002:850E` | `21h` | `cd21` | sub_189BF | 是 | `int 21h; DOS - 2+ - CLOSE A FILE WITH HANDLE` |
| `18CA1` | `seg002:8791` | `21h` | `cd21` | sub_18C9E | 是 | `int 21h; DOS - GET ALLOCATION TABLE INFORMATION FOR DEFAULT DRIVE` |
| `18CDF` | `seg002:87CF` | `21h` | `cd21` | sub_18CCC | 是 | `int 21h; DOS - 2+ - FIND FIRST ASCIZ (FINDFIRST)` |
| `18CEF` | `seg002:87DF` | `21h` | `cd21` | sub_18CCC | 是 | `int 21h; DOS - 2+ - FIND FIRST ASCIZ (FINDFIRST)` |
| `190B3` | `seg002:8BA3` | `21h` | `cd21` | sub_1904B | 否 | `int 21h; DOS - GET DISK TRANSFER AREA ADDRESS` |
| `19165` | `seg002:8C55` | `21h` | `cd21` | sub_19152 | 否 | `int 21h; DOS - 2+ - FIND FIRST ASCIZ (FINDFIRST)` |
| `19172` | `seg002:8C62` | `21h` | `cd21` | sub_19152 | 否 | `int 21h; DOS - 2+ - FIND NEXT ASCIZ (FINDNEXT)` |
| `191D5` | `seg002:8CC5` | `21h` | `cd21` | sub_19187 | 是 | `int 21h; DOS - 2+ - OPEN DISK FILE WITH HANDLE` |
| `191E5` | `seg002:8CD5` | `21h` | `cd21` | sub_19187 | 是 | `int 21h; DOS - 2+ - READ FROM FILE WITH HANDLE` |
| `191F3` | `seg002:8CE3` | `21h` | `cd21` | sub_19187 | 是 | `int 21h; DOS - 2+ - CLOSE A FILE WITH HANDLE` |
| `1B9D6` | `seg002:B4C6` | `21h` | `cd21` | sub_1B9D0 | 是 | `int 21h; DOS - 2+ - GET INTERRUPT VECTOR` |
| `1B9EC` | `seg002:B4DC` | `21h` | `cd21` | sub_1B9D0 | 是 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1BA09` | `seg002:B4F9` | `21h` | `cd21` | sub_1B9F8 | 否 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1BCD8` | `seg002:B7C8` | `21h` | `cd21` | start | 是 | `int 21h; DOS - GET DOS VERSION` |
| `1BCDE` | `seg002:B7CE` | `20h` | `cd20` | start | 否 | `int 20h; DOS - PROGRAM TERMINATION` |
| `1BD0E` | `seg002:B7FE` | `21h` | `cd21` | start | 否 | `int 21h; DOS - 2+ - QUIT WITH EXIT CODE (EXIT)` |
| `1BD36` | `seg002:B826` | `21h` | `cd21` | start | 是 | `int 21h; DOS - 2+ - ADJUST MEMORY BLOCK SIZE (SETBLOCK)` |
| `1BD9E` | `seg002:B88E` | `21h` | `cd21` | sub_1BD9C | 是 | `int 21h; DOS - GET DOS VERSION` |
| `1BDA6` | `seg002:B896` | `21h` | `cd21` | sub_1BD9C | 是 | `int 21h; DOS - 2+ - GET INTERRUPT VECTOR` |
| `1BDB8` | `seg002:B8A8` | `21h` | `cd21` | sub_1BD9C | 是 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1BE3C` | `seg002:B92C` | `21h` | `cd21` | sub_1BD9C | 是 | `int 21h; DOS - 2+ - IOCTL - GET DEVICE INFORMATION` |
| `1BEAF` | `seg002:B99F` | `21h` | `cd21` | sub_1BE77 | 否 | `int 21h; DOS - 2+ - CLOSE A FILE WITH HANDLE` |
| `1BEBC` | `seg002:B9AC` | `21h` | `cd21` | sub_1BE77 | 否 | `int 21h; DOS - 2+ - QUIT WITH EXIT CODE (EXIT)` |
| `1BED3` | `seg002:B9C3` | `21h` | `cd21` | sub_1BEBE | 否 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1BEE7` | `seg002:B9D7` | `21h` | `cd21` | sub_1BEBE | 否 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1C1BF` | `seg002:BCAF` | `21h` | `cd21` | sub_1C19D | 否 | `int 21h; DOS - 2+ - WRITE TO FILE WITH HANDLE` |
| `1C1F0` | `seg002:BCE0` | `21h` | `cd21` | sub_1C1C8 | 否 | `int 21h; DOS - 2+ - ADJUST MEMORY BLOCK SIZE (SETBLOCK)` |
| `1CF4A` | `seg002:CA3A` | `21h` | `cd21` | sub_1CF22 | 否 | `int 21h; DOS - 2+ - MOVE FILE READ/WRITE POINTER (LSEEK)` |
| `1CF6E` | `seg002:CA5E` | `21h` | `cd21` | sub_1CF22 | 否 | `int 21h; DOS - 2+ - MOVE FILE READ/WRITE POINTER (LSEEK)` |
| `1CF81` | `seg002:CA71` | `21h` | `cd21` | sub_1CF22 | 否 | `int 21h; DOS - 2+ - MOVE FILE READ/WRITE POINTER (LSEEK)` |
| `1CF90` | `seg002:CA80` | `21h` | `cd21` | sub_1CF22 | 否 | `int 21h; DOS - 2+ - MOVE FILE READ/WRITE POINTER (LSEEK)` |
| `1CFC0` | `seg002:CAB0` | `21h` | `cd21` | sub_1CF9C | 否 | `int 21h; DOS - 2+ - MOVE FILE READ/WRITE POINTER (LSEEK)` |
| `1D052` | `seg002:CB42` | `21h` | `cd21` | sub_1D044 | 否 | `int 21h; DOS - 2+ - WRITE TO FILE WITH HANDLE` |
| `1D0A3` | `seg002:CB93` | `21h` | `cd21` | sub_1CF9C | 否 | `int 21h; DOS - 2+ - WRITE TO FILE WITH HANDLE` |
| `1D3C7` | `seg002:CEB7` | `21h` | `cd21` | sub_1D370 | 否 | `int 21h; DOS - 2+ - ALLOCATE MEMORY` |
| `1D422` | `seg002:CF12` | `21h` | `cd21` | sub_1D3DE | 否 | `int 21h; DOS - 2+ - ADJUST MEMORY BLOCK SIZE (SETBLOCK)` |
| `1D877` | `seg002:D367` | `21h` | `cd21` | sub_1D834 | 是 | `int 21h; DOS - PARSE FILENAME` |
| `1D87F` | `seg002:D36F` | `21h` | `cd21` | sub_1D834 | 是 | `int 21h; DOS - PARSE FILENAME` |
| `1D8C1` | `seg002:D3B1` | `21h` | `cd21` | sub_1D834 | 是 | `int 21h; DOS - CHECK STANDARD INPUT STATUS` |
| `1D8CF` | `seg002:D3BF` | `21h` | `cd21` | sub_1D834 | 是 | `int 21h; DOS - 2+ - LOAD OR EXECUTE (EXEC)` |
| `1D8D6` | `seg002:D3C6` | `21h` | `cd21` | sub_1D834 | 是 | `int 21h; DOS - GET DOS VERSION` |
| `1D910` | `seg002:D400` | `21h` | `cd21` | sub_1D834 | 是 | `int 21h; DOS - 2+ - GET EXIT CODE OF SUBPROGRAM (WAIT)` |
| `1DBF4` | `seg003:0134` | `21h` | `cd21` | sub_1DBB7 | 是 | `int 21h; DOS - 2+ - GET INTERRUPT VECTOR` |
| `1DC12` | `seg003:0152` | `21h` | `cd21` | sub_1DBB7 | 是 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1DC88` | `seg003:01C8` | `21h` | `cd21` | sub_1DC80 | 否 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1E536` | `seg003:0A76` | `15h` | `cd15` | sub_1E3AD | 否 | `int 15h; SYSTEM - GET CONFIGURATION (XT after 1/10/86,AT mdl 3x9,CONV,XT286,PS)` |
| `1E776` | `seg003:0CB6` | `1Ah` | `cd1a` | — | 否 | `int 1Ah` |
| `1E7AB` | `seg003:0CEB` | `1Ah` | `cd1a` | — | 否 | `int 1Ah` |
| `1E7F4` | `seg003:0D34` | `1Ah` | `cd1a` | sub_1E7D8 | 否 | `int 1Ah` |
| `1E809` | `seg003:0D49` | `1Ah` | `cd1a` | sub_1E7D8 | 否 | `int 1Ah` |
| `1E8E8` | `seg003:0E28` | `1Ah` | `cd1a` | sub_1E827 | 否 | `int 1Ah` |
| `1E92B` | `seg003:0E6B` | `08h` | `cd08` | sub_1E908 | 是 | `int 8; - IRQ0 - TIMER INTERRUPT` |
| `1F0E3` | `seg006:0033` | `21h` | `cd21` | sub_1F0B2 | 否 | `int 21h; DOS - 2+ - GET INTERRUPT VECTOR` |
| `1F0F7` | `seg006:0047` | `21h` | `cd21` | sub_1F0B2 | 否 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1F10A` | `seg006:005A` | `21h` | `cd21` | sub_1F0B2 | 否 | `int 21h; DOS - 2+ - GET INTERRUPT VECTOR` |
| `1F11E` | `seg006:006E` | `21h` | `cd21` | sub_1F0B2 | 否 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1F1B0` | `seg006:0100` | `21h` | `cd21` | sub_1F168 | 否 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1F311` | `seg006:0261` | `21h` | `cd21` | sub_1F2C9 | 否 | `int 21h; DOS - 2+ - GET INTERRUPT VECTOR` |
| `1F325` | `seg006:0275` | `21h` | `cd21` | sub_1F2C9 | 否 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1F336` | `seg006:0286` | `21h` | `cd21` | sub_1F2C9 | 否 | `int 21h; DOS - 2+ - GET INTERRUPT VECTOR` |
| `1F34A` | `seg006:029A` | `21h` | `cd21` | sub_1F2C9 | 否 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1F3EE` | `seg006:033E` | `21h` | `cd21` | sub_1F2C9 | 否 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1F406` | `seg006:0356` | `21h` | `cd21` | sub_1F2C9 | 否 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `1F5C9` | `seg006:0519` | `15h` | `cd15` | — | 否 | `int 15h; OS HOOK - SET FLAG AND COMPLETE INTERRUPT (AT,XT2,XT286,CONV,PS)` |
| `1F616` | `seg006:0566` | `15h` | `cd15` | — | 否 | `int 15h; OS HOOK - SET FLAG AND COMPLETE INTERRUPT (AT,XT2,XT286,CONV,PS)` |
| `201CD` | `seg015:002D` | `21h` | `cd21` | — | 否 | `int 21h; DOS - 2+ - GET INTERRUPT VECTOR` |
| `201E1` | `seg015:0041` | `21h` | `cd21` | — | 否 | `int 21h; DOS - SET INTERRUPT VECTOR` |
| `20233` | `seg015:0093` | `21h` | `cd21` | — | 否 | `int 21h; DOS - SET INTERRUPT VECTOR` |

## 附錄 B：LOGO.EXE 全部 `INT` 呼叫點

| IDA 位址 | 段:位移 | 中斷 | bytes | 函式 | 這次執行過 | 反組譯 |
|---|---|---|---|---|---|---|
| `1001F` | `seg000:001F` | `21h` | `cd21` | start | 是 | `int 21h; DOS - 3+ - GET PSP ADDRESS` |
| `10039` | `seg000:0039` | `21h` | `cd21` | start | 是 | `int 21h; DOS - 2+ - QUIT WITH EXIT CODE (EXIT)` |
| `10A3E` | `seg000:0A3E` | `10h` | `cd10` | sub_10A32 | 否 | `int 10h; - VIDEO - SET VIDEO MODE` |
| `10A4F` | `seg000:0A4F` | `10h` | `cd10` | sub_10A32 | 否 | `int 10h; - VIDEO - SET VIDEO MODE` |
| `10A96` | `seg000:0A96` | `10h` | `cd10` | sub_10A88 | 否 | `int 10h; - VIDEO - SET COLOR PALETTE` |
| `10A9E` | `seg000:0A9E` | `10h` | `cd10` | sub_10A88 | 否 | `int 10h; - VIDEO - SET COLOR PALETTE` |
| `10AC9` | `seg000:0AC9` | `10h` | `cd10` | sub_10ABC | 否 | `int 10h; - VIDEO - SET COLOR PALETTE` |
| `10ADD` | `seg000:0ADD` | `10h` | `cd10` | sub_10ABC | 否 | `int 10h; - VIDEO - SET PALETTE REGISTER (Jr, PS, TANDY 1000, EGA, VGA)` |
