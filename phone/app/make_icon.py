# Draws the app icon (a remote control with an IR beam) in the Sailfish launcher sizes.
# Usage: python3 make_icon.py <out dir>  ->  <out dir>/<s>x<s>/toh-ir.png
import os, sys
from PIL import Image, ImageDraw
out = sys.argv[1]
for s in (86, 108, 128, 172):
    S = s * 4
    im = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    d = ImageDraw.Draw(im)
    d.ellipse([0, 0, S - 1, S - 1], fill=(40, 60, 90, 255))
    w, h = S * 0.30, S * 0.58
    x0, y0 = S * 0.5 - w / 2, S * 0.30
    d.rounded_rectangle([x0, y0, x0 + w, y0 + h], radius=S * 0.06, fill=(235, 240, 245, 255))
    d.ellipse([S * 0.5 - S * 0.05, y0 + S * 0.06, S * 0.5 + S * 0.05, y0 + S * 0.16], fill=(220, 60, 60, 255))
    for r in range(3):
        for c in range(2):
            cx, cy = S * 0.5 + (c - 0.5) * S * 0.11, y0 + S * 0.26 + r * S * 0.09
            d.ellipse([cx - S * 0.03, cy - S * 0.03, cx + S * 0.03, cy + S * 0.03], fill=(90, 100, 120, 255))
    for i, rad in enumerate((0.10, 0.17, 0.24)):
        bb = [S * 0.5 - S * rad, y0 - S * rad, S * 0.5 + S * rad, y0 + S * rad]
        d.arc(bb, 235, 305, fill=(255, 200, 80, 255 - 50 * i), width=max(4, int(S * 0.025)))
    p = os.path.join(out, f"{s}x{s}")
    os.makedirs(p, exist_ok=True)
    im.resize((s, s), Image.LANCZOS).save(os.path.join(p, "toh-ir.png"))
print(out)
