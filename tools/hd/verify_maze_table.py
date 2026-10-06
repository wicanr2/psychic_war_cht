"""研究038 §72：獨立核對CS:307B的18×18原始slot表、正常貼圖及三份狀態。"""
import hashlib
import json
from pathlib import Path

from verify_maze_draw import crop, unpack


ROOT = Path('/src')
OUT = ROOT / 'workplace/hd'


def main():
    target = OUT / 'maze-table-verification-v1-20261003.json'
    assert not target.exists(), '拒絕覆寫'
    paths = [Path(__file__), ROOT/'tools/hd/verify_maze_draw.py',
             OUT/'maze-draw-source-v2-20261003.json',
             ROOT/'workplace/ida/hd-maze-20261003/parent.json',
             Path('/orig/psychic-war/MAZE.BIN')]
    inputs = {}

    def read(path):
        value = path.read_bytes()
        inputs[str(path)] = hashlib.sha256(value).hexdigest()
        return value

    for path in paths:
        read(path)
    draw = json.loads(read(paths[2]))
    maze = read(paths[4])
    parent = json.loads(read(paths[3]))
    decoded = {x['segment_offset']:x for x in parent['decoded_instructions']}
    assert decoded[0x4fa8]['bytes'] == 'bb7b30'
    assert decoded[0x4fb7]['bytes'] == '2e8a07'
    assert decoded[0x4fc0]['bytes'] == 'b91200'
    assert decoded[0x4fc3]['bytes'] == '03d9'
    assert decoded[0x4fd5]['bytes'] == 'c3'
    events = draw['events']
    assert len(events) == 648
    grids = []
    for cycle, block in enumerate([events[:324], events[324:]]):
        grid = bytearray(324)
        coords = set()
        for ordinal, event in enumerate(block):
            x, y = (event['X']-4)//4, (event['Y']-124)//4
            assert (x,y) == (ordinal//18,ordinal%18)
            assert (x,y) not in coords and event['ReturnIP'] == 0x4fbe
            coords.add((x,y))
            grid[y*18+x] = event['Slot']
        assert len(coords) == 324
        grids.append({'cycle':cycle,'first_entry':block[0]['Entry'],
                      'last_tile_return':block[-1]['Return'], 'slots':list(grid),
                      'selector_sha256':hashlib.sha256(grid).hexdigest()})
    samples = []
    for label, frame in [
        ('initial',ROOT/'workplace/states/07-first-play.frame'),
        ('bbs',OUT/'room-nearby-explore-v1-20261003-bbs.frame'),
        ('radar',OUT/'room-nearby-explore-v1-20261003-radar.frame'),
    ]:
        ram_path = OUT/f'maze-tile-sources-v2-20261003-{label}.ram.bin'
        ram = read(ram_path)
        original = read(frame)
        table = ram[0x1610+0x307b:0x1610+0x307b+324]
        assert len(table) == 324 and len(original) == 64000
        assert ram[0x12e16:0x13616] == maze
        region = b''.join(original[y*320+4:y*320+76] for y in range(124,196))
        rebuilt = bytearray(72*72)
        for row in range(18):
            for col in range(18):
                tile = unpack(maze[table[row*18+col]*8:table[row*18+col]*8+8])
                for y in range(4):
                    pos = (row*4+y)*72+col*4
                    rebuilt[pos:pos+4] = tile[y*4:y*4+4]
        assert rebuilt == region, label+'的slot表與原版視野不符'
        changed = bytearray(rebuilt); changed[0] ^= 1
        assert changed != region
        if label == 'bbs':
            assert list(table) == grids[-1]['slots'], '實際AL slot序列與保存表不同'
        samples.append({'label':label,'ram':str(ram_path),'frame':str(frame),
                        'slots':list(table),'selector_sha256':hashlib.sha256(table).hexdigest(),
                        'region_sha256':hashlib.sha256(region).hexdigest(),
                        'mismatch':0,'negative_pixel_mismatch':1})
    # 同bytes的slot不以像素判身份，保留表中的原始圖號。
    duplicates = [[3,235],[6,201],[70,220]]
    for a,b in duplicates:
        assert maze[a*8:a*8+8] == maze[b*8:b*8+8]
    data = {
        'scope':'正常原版迷宮slot表、實際AL身份及三份保存狀態；不是HD驗收',
        'inputs_sha256':inputs,'runtime_table_cs_ip':'0161:307B',
        'runtime_table_linear':0x1610+0x307b,'table_size':[18,18],
        'table_order':'slot = table[row*18+column]; drawing visits column first, then row',
        'draw_cycles':grids,'samples':samples,'duplicate_slot_pairs':duplicates,
        'limits':'只證明所列正常狀態。未證明其他場景slot表有效、ROOM優先序、共同函式入口/返回實測或正式主題資料契約。',
    }
    with target.open('x') as stream:
        json.dump(data,stream,ensure_ascii=False,indent=2);stream.write('\n')
    print('三份原版slot表重建不符0，BBS表與324次AL身份完全相同。')


if __name__ == '__main__':
    main()
