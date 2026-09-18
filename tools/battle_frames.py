"""DOSBox-X 戰鬥錄影逐格判讀：敵人名稱何時消失（issue #9，搭配 tools/dosboxx-battle-speed.sh）。

    tools/py.sh tools/battle_frames.py <rgb 目錄> [fps] [按下的格號]

輸入是 640×400 的 raw RGB（frames/*.png 以 `convert x.png -depth 8 rgb:x.rgb` 轉出）。
判準是**敵人圖像上半身**（x 60–115、y 290–320）全黑的第一格：敵人 HP 歸零時圖像立刻清掉，
名稱（x 12–130、y 262–285）要再過幾秒才擦掉，拿名稱當終點在 240 cycles 下會多算約 5 秒。
戰鬥時間 ＝ （消失格 − 按下格）÷ fps；解析度 ±1 格。兩個時點都印出來。
"""
import pathlib
import sys

W, H = 640, 400
NAME = (12, 130, 262, 285)
SPRITE = (60, 115, 290, 320)  # 只取上半身：雷射光束橫過 y 323–347，會讓「圖像已清掉」被誤判成還在


def lit(buf, box, white_only):
    x0, x1, y0, y1 = box
    n = 0
    for y in range(y0, y1):
        row = y * W * 3
        for x in range(x0, x1):
            i = row + x * 3
            r, g, b = buf[i], buf[i + 1], buf[i + 2]
            if (r > 200 and g > 200 and b > 200) if white_only else (r > 40 or g > 40 or b > 40):
                n += 1
    return n


def main():
    d = pathlib.Path(sys.argv[1])
    fps = float(sys.argv[2]) if len(sys.argv) > 2 else 4.0
    press = int(sys.argv[3]) if len(sys.argv) > 3 else int(fps)
    files = sorted(d.glob("*.rgb"))
    # 沒有 .rgb 就明說。少了這一段的話，下面的 name[press] 會在空清單上爆 IndexError，
    # 而 press 落在清單內但檔案殘缺時更糟——會印「按下時敵人不在畫面上」，
    # 看起來像走位失敗，其實只是忘了把 PNG 轉成 raw RGB。
    if len(files) <= press:
        print("%s 只有 %d 個 .rgb（要 > %d）。錄影是 PNG 的話先轉：" % (d, len(files), press))
        print("  for p in *.png; do convert \"$p\" -depth 8 \"rgb:${p%.png}.rgb\"; done")
        return 2
    bufs = [f.read_bytes() for f in files]
    name = [lit(b, NAME, True) for b in bufs]
    sprite = [lit(b, SPRITE, False) for b in bufs]
    if name[press] == 0 or sprite[press] == 0:
        print("按下時敵人不在畫面上")
        return 1
    gone = next((i for i in range(press, len(bufs)) if sprite[i] == 0), None)
    name_gone = next((i for i in range(press, len(bufs)) if name[i] == 0), None)
    if gone is None:
        print("錄影結束時敵人還在（%d 格）" % len(bufs))
        return 1
    print("frames=%d fps=%.1f press=%d sprite_gone=%d battle_sec=%.2f（±%.2f） name_gone=%s" %
          (len(bufs), fps, press, gone, (gone - press) / fps, 1 / fps,
           "—" if name_gone is None else "%d（%.2f 秒）" % (name_gone, (name_gone - press) / fps)))
    return 0


if __name__ == "__main__":
    sys.exit(main())
