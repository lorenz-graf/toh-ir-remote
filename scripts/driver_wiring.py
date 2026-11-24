# Wiring diagram for the IR LED driver: IR LED switched by an S8050 NPN from the TOH's 5 V, base driven by the Zero's pin 7.
# Usage (repo root): python3 scripts/driver_wiring.py renders/ir-driver-wiring.png
import sys
from PIL import Image, ImageDraw, ImageFont

out = sys.argv[1]
F = "/usr/share/fonts/TTF/DejaVuSans.ttf"; FB = "/usr/share/fonts/TTF/DejaVuSans-Bold.ttf"
f22, f26, b26, b30, b34, b44 = (ImageFont.truetype(F, 22), ImageFont.truetype(F, 26), ImageFont.truetype(FB, 26),
                                ImageFont.truetype(FB, 30), ImageFont.truetype(FB, 34), ImageFont.truetype(FB, 44))
INK, GREY, RED, BLUE, IR = (25, 25, 30), (110, 110, 120), (210, 40, 40), (30, 90, 200), (150, 40, 160)
W, H = 1900, 1160
cv = Image.new("RGB", (W, H), "white"); d = ImageDraw.Draw(cv)
LW = 5

def line(*pts, col=INK, w=LW): d.line(pts, fill=col, width=w, joint="curve")
def dot(x, y, col=INK): d.ellipse([x - 9, y - 9, x + 9, y + 9], fill=col)

d.text((40, 20), "IR LED driver: S8050 + IR LED from 5 V", fill=INK, font=b44)
d.text((40, 82), "Replaces the transmitter module on pin 7. Firmware unchanged (pin 7 high = LED on).", fill=GREY, font=f26)

# rails
Y5, YG, XL, XR = 220, 1000, 330, 1340
line((XL, Y5), (XR, Y5), col=RED, w=8); line((XL, YG), (XR, YG), col=INK, w=8)
d.text((XL - 20, Y5), "5V", fill=RED, font=b34, anchor="rm")
d.text((XL - 20, Y5 + 38), "breakout header row", fill=GREY, font=f22, anchor="rm")
d.text((XL - 20, YG), "GND", fill=INK, font=b34, anchor="rm")
d.text((XL - 20, YG + 38), "same GND as the Zero", fill=GREY, font=f22, anchor="rm")

# 22 ohm
X = 800
dot(X, Y5, RED); line((X, Y5), (X, 270))
d.rectangle([X - 28, 270, X + 28, 400], outline=INK, width=LW, fill="white")
d.text((X + 50, 335), "22 Ω", fill=INK, font=b30, anchor="lm")
line((X, 400), (X, 470))

# IR LED (anode top)
d.polygon([(X - 45, 470), (X + 45, 470), (X, 545)], outline=INK, fill="white", width=LW)
line((X - 45, 548), (X + 45, 548))
line((X, 548), (X, 712))
for dy in (0, 36):
    line((X + 55, 490 + dy), (X + 115, 450 + dy), col=IR, w=4)
    d.polygon([(X + 115, 450 + dy), (X + 95, 452 + dy), (X + 108, 469 + dy)], fill=IR)
d.text((X - 60, 470), "+", fill=INK, font=b34, anchor="rm")
d.text((X - 60, 545), "−", fill=INK, font=b34, anchor="rm")
d.text((X + 135, 485), "IR LED", fill=INK, font=b30, anchor="lm")
d.text((X + 135, 522), "(from the spare module)", fill=GREY, font=f22, anchor="lm")

# S8050 (NPN), collector/emitter on the X line
cx, cy = X - 25, 770
d.ellipse([cx - 70, cy - 70, cx + 70, cy + 70], outline=INK, width=4)
bx = cx - 25
line((bx, cy - 40), (bx, cy + 40), w=8)
line((bx, cy - 15), (X, 712))                     # collector
line((bx, cy + 15), (X, 828))                     # emitter
d.polygon([(X, 828), (X - 30, 822), (X - 14, 803)], fill=INK)
line((X, 828), (X, YG)); dot(X, YG)
d.text((X + 18, 700), "C", fill=BLUE, font=b30, anchor="lm")
d.text((X + 18, 840), "E", fill=BLUE, font=b30, anchor="lm")
d.text((bx - 20, cy - 40), "B", fill=BLUE, font=b30, anchor="rb")
d.text((cx + 90, cy), "S8050", fill=INK, font=b30, anchor="lm")
d.text((cx + 90, cy + 37), "NPN transistor", fill=GREY, font=f22, anchor="lm")

# base resistor + pin 7
line((bx, cy), (640, cy))
d.rectangle([480, cy - 26, 640, cy + 26], outline=INK, width=LW, fill="white")
d.text((560, cy - 40), "470 Ω", fill=INK, font=b30, anchor="mb")
line((480, cy), (360, cy))
d.rounded_rectangle([60, cy - 60, 360, cy + 60], radius=14, outline=BLUE, width=5, fill=(235, 242, 255))
d.text((210, cy - 16), "RP2040-Zero", fill=INK, font=b26, anchor="mm")
d.text((210, cy + 24), "pin 7", fill=BLUE, font=b34, anchor="mm")

# 100 uF across the rails
XC = 1290
dot(XC, Y5, RED); line((XC, Y5), (XC, 570))
line((XC - 50, 570), (XC + 50, 570))
d.rectangle([XC - 50, 590, XC + 50, 606], fill=INK)
line((XC, 606), (XC, YG)); dot(XC, YG)
d.text((XC + 62, 560), "+", fill=INK, font=b30, anchor="lm")
d.text((XC + 62, 612), "−", fill=INK, font=b30, anchor="lm")
d.text((XC - 65, 575), "100 µF", fill=INK, font=b30, anchor="rm")
d.text((XC - 65, 612), "striped leg = −", fill=GREY, font=f22, anchor="rm")
d.text((XC - 65, 642), "→ GND", fill=GREY, font=f22, anchor="rm")
d.text((XC - 40, YG + 50), "capacitor: close to the LED", fill=GREY, font=f22, anchor="mm")

# right panel: parts as they look
PX = 1480
d.line([(PX - 60, 150), (PX - 60, 1110)], fill=(220, 220, 225), width=3)
d.text((PX + 170, 160), "How to tell the legs", fill=BLUE, font=b34, anchor="mm")

# TO-92, flat side facing the viewer
tx, ty = PX + 170, 260
d.rounded_rectangle([tx - 75, ty, tx + 75, ty + 130], radius=40, fill=(45, 45, 50))
d.rectangle([tx - 75, ty + 60, tx + 75, ty + 130], fill=(45, 45, 50))
d.text((tx, ty + 75), "S8050", fill="white", font=b26, anchor="mm")
for i, (lab, col) in enumerate([("E", INK), ("B", INK), ("C", INK)]):
    lx = tx - 50 + i * 50
    d.rectangle([lx - 5, ty + 130, lx + 5, ty + 250], fill=(150, 150, 155))
    d.text((lx, ty + 260), lab, fill=BLUE, font=b34, anchor="mt")
d.text((tx, ty + 315), "flat side with the print", fill=INK, font=f22, anchor="mt")
d.text((tx, ty + 343), "facing you, legs down:", fill=INK, font=f22, anchor="mt")
d.text((tx, ty + 371), "E  B  C  (left → right)", fill=INK, font=b26, anchor="mt")

# LED
lx, ly = PX + 170, 820
d.ellipse([lx - 45, ly - 45, lx + 45, ly + 45], fill=(235, 235, 240), outline=INK, width=3)
d.rectangle([lx - 45, ly, lx + 45, ly + 70], fill=(235, 235, 240))
d.line([(lx - 45, ly), (lx - 45, ly + 70)], fill=INK, width=3); d.line([(lx + 45, ly), (lx + 45, ly + 70)], fill=INK, width=3)
d.rectangle([lx - 55, ly + 70, lx + 55, ly + 84], fill=(235, 235, 240), outline=INK, width=3)
d.rectangle([lx - 55, ly + 70, lx - 46, ly + 84], fill=INK)          # flat spot on the rim = minus
d.rectangle([lx - 24, ly + 84, lx - 16, ly + 190], fill=(150, 150, 155))   # minus, short
d.rectangle([lx + 16, ly + 84, lx + 24, ly + 230], fill=(150, 150, 155))   # plus, long
d.text((lx - 20, ly + 200), "−", fill=INK, font=b34, anchor="mt")
d.text((lx + 20, ly + 240), "+", fill=INK, font=b34, anchor="mt")
d.text((lx - 160, ly + 100), "flat spot", fill=GREY, font=f22, anchor="lm")
d.text((lx - 160, ly + 126), "= −", fill=GREY, font=f22, anchor="lm")
d.text((lx + 50, ly + 150), "long leg = +", fill=GREY, font=f22, anchor="lm")
d.text((lx, ly - 75), "If the legs are cut: + is the", fill=INK, font=f22, anchor="mb")
d.text((lx, ly - 50), "pad marked \"+\" on the module", fill=INK, font=f22, anchor="mb")

cv.save(out); print(out, cv.size)
