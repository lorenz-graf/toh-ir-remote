# Makes a GIF as wide as fits under a size limit at a fixed frame rate. GIF size grows with the area (width²),
# so one test encode predicts the width; one or two more encodes land just under the limit.
# Phone footage is noisy: a light denoise (hqdn3d) and ordered (bayer) dithering roughly halve the size.
# Usage: python3 make_gif.py <in.mp4> <out.gif> <fps> [limit_bytes, default 10000000]
import math, os, subprocess, sys
src, dst, fps = sys.argv[1], sys.argv[2], int(sys.argv[3])
limit = int(sys.argv[4]) if len(sys.argv) > 4 else 10_000_000

def encode(width):
    vf = (f"fps={fps},scale={width}:-2:flags=lanczos,hqdn3d=4:3:8:8,split[a][b];"
          "[a]palettegen=stats_mode=diff[p];[b][p]paletteuse=dither=bayer:bayer_scale=3:diff_mode=rectangle")
    subprocess.run(["ffmpeg", "-v", "error", "-y", "-i", src, "-vf", vf, "-loop", "0", dst], check=True)
    size = os.path.getsize(dst)
    print(f"width {width}: {size / 1e6:.2f} MB", flush=True)
    return size

w = 320
size = encode(w)
for _ in range(4):
    nw = int(w * math.sqrt(limit * 0.97 / size)) // 2 * 2
    if size <= limit and nw <= w + 8:
        break
    w = min(nw, 1080) if size <= limit else min(nw, w - 8)
    size = encode(w)
while size > limit:
    w = int(w * 0.97) // 2 * 2
    size = encode(w)
print(f"done: {dst} {w} px, {fps} fps, {size / 1e6:.2f} MB")
