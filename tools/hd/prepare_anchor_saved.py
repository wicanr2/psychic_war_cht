"""選出十九原版檢查點與四組完整角色來源；研究038 §67。"""
import json
from pathlib import Path

from verify_anchor_render import assets_from_files
from verify_next_body_plane import region, require
from verify_over_runtime import read


def main():
    output = Path("workplace/hd/background-anchor-saved-input-v1-20261003.json")
    require(not output.exists(), "拒絕覆寫")
    inputs = {}
    assets = assets_from_files(inputs)
    proof = json.loads(read("workplace/hd/background-anchor-source-v1-20261003.json", inputs))
    replay = json.loads(read("replay/title-to-first-save.json", inputs))
    frames = ["workplace/states/" + row["name"] + ".frame" for row in replay["segments"]]
    selected = []
    groups = [("ENEMY00.PBL", range(0, 3)), ("ENEMY00.PBL", range(3, 6)),
              ("ENEMY00.PBL", range(6, 9)), ("ENEMY01.PBL", range(6, 9))]
    for name, images in groups:
        choices = [a for a in assets if a[0]["pbl"] == name and a[0]["image"] in images]
        found = None
        for row in proof["results"]:
            frame = read(row["frame"], inputs)
            for entry, w, h, source, _, _ in choices:
                if region(frame, *entry["at"], w, h) == source:
                    found = {"frame": row["frame"], "pbl": name, "image": entry["image"]}
                    break
            if found:
                break
        require(found is not None, "找不到完整角色保存幀：" + name + str(list(images)))
        selected.append(found)
        if found["frame"] not in frames:
            frames.append(found["frame"])
    cases = []
    for frame in frames:
        state = frame[:-6] + ".state"
        require(Path(state).is_file(), "來源state不存在：" + state)
        read(frame, inputs)
        read(state, inputs)
        cases.append({"frame": frame, "state": state})
    read(__file__, inputs)
    with output.open("x") as f:
        json.dump({"inputs_sha256": inputs, "cases": cases, "roles": selected,
                   "scope": "十九原版檢查點與四組完整角色，以原版PBL及frame選擇保存狀態",
                   "limits": "保存狀態抽樣，不是正常鍵序或全姿勢／全角色驗收"},
                  f, ensure_ascii=False, indent=2)
        f.write("\n")
    print("保存幀回歸", len(cases), "份，完整來源", selected, flush=True)


if __name__ == "__main__":
    main()
