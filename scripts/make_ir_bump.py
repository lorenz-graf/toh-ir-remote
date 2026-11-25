# IR cover, bump and lid: the generic wide-bump cover, open-back bump and lid (../toh-devkit/model/wide-bump/, made by
# ../toh-devkit/scripts/make_wide_bump.py) + what the IR parts need, so model/ has everything this project needs
# (user): model/ir-cover, model/ir-bump, model/ir-lid (STLs in model/, STEPs in model/step/).
# Layout (user, 2026-11-25, worked out in Blender): the 5 mm IR LED (8.6 long, rim flat turned to the outside) and the
# IR receiver lying flat (5.4 x 2.5 block + half cylinder d 3.8 on top along its whole 6.8 length) sit in the
# compartment, the LED on the phone's back through a cutout in the lid, the receiver on the lid (user); both look out
# of the phone's top edge. What sticks out above
# the roof is covered by a small cap (13.8 wide, 1.4 above the roof, centred on the phone's width). A ridge across the
# bump just below the lower camera window makes the phone lying on its back rest on the cover's bottom edge and the
# ridge (full width, can't rock). RIDGE_H is computed so the cap would just touch the same plane (3 legs), plus
# RIDGE_MARGIN for print tolerance, so the cap floats slightly above the table.
# - LED tip and receiver front OUT_BY past the bump's front face (user); where they run over the cover, the cover
#   gets notches in its outside face; no stops: the legs (and a drop of glue) hold them
# - LED: U-slot open to the phone side (drops in); receiver: bay = its profile + 0.2, out through the front
# Print: cap side on the bed; the roof is 1.4 up -> supports from the build plate.
# Run from this folder: freecadcmd -c "exec(open('make_ir_bump.py').read())"
import os, math, shutil, Part, MeshPart, Mesh, FreeCAD as App
REPO = os.path.dirname(os.getcwd()); OUT = os.path.join(REPO, "model"); os.makedirs(os.path.join(OUT, "step"), exist_ok=True)   # STLs; STEPs in step/
GEN = os.path.join(REPO, "..", "toh-devkit", "model", "wide-bump")
P = lambda u, v, z: App.Vector(-37.0 + u, -78.8 + v, z)

V1, ROOF, BUMP_IN = 151.35, -6.2, -1.8                   # generic bump: top edge, roof, inner face (= phone's back)
BOTTOM_V, FACE = -0.15, -3.0                             # cover's outside face reaches the phone's bottom edge here
FLASH_U, FLASH_V, FLASH_R, CONE = 44.6, 134.0, 3.0, math.radians(25)
LED_U, LED_Z, LED_L = 33.6, BUMP_IN - 2.9, 8.6        # axis; rim (r 2.9) on the phone's back
LED_R, RIM_R, RIM_T = 2.5, 2.9, 1.0
RIM_ROOF = 0.2                            # cap roof over the rim pocket: room for a rim flat up to 2.7 from the axis (assumed 2.5)
REC_U, REC_Z0, REC_L = 40.6, BUMP_IN - 0.8, 6.8       # receiver on the lid (0.8)
OUT_BY = 2.5                               # LED tip and receiver front this far past the bump's front face (user)
NOTCH_CLEAR = 0.1                          # around them in the cover's notches
REC_BW, REC_BH, REC_CR = 5.4, 2.5, 1.9
RIDGE = (8.8, 65.2, 116.4, 118.2)                        # strip between the roof's lower edge and the lower camera window
RIDGE_TOP_R = 0.25
RIDGE_MARGIN = 0.2                       # print tolerance (user): the ridge is this much higher than 3 legs need, so the cap floats

def box(u0, u1, v0, v1, z0, z1, r=0.0):
    f = Part.Face(Part.makePolygon([P(u0, v0, z0), P(u1, v0, z0), P(u1, v1, z0), P(u0, v1, z0), P(u0, v0, z0)]))
    t = f.extrude(App.Vector(0, 0, z1 - z0))
    return t.makeFillet(r, [e for e in t.Edges if abs(e.tangentAt(e.FirstParameter).normalize().z) > 0.99]) if r else t
def round_top(s, z_top, r):
    try: return s.makeFillet(r, [e for e in s.Edges if abs(e.BoundBox.ZMax - z_top) < 1e-6 and abs(e.BoundBox.ZMin - z_top) < 1e-6])
    except Exception as ex: print("top fillet skipped:", ex); return s
def slope(shape, ref=None):
    """Steepest line from the cover's bottom edge to the shape's points (the resting plane's dz/dv)."""
    pts = shape.tessellate(0.02)[0]
    return min(((p.z - FACE) / (p.y + 78.8 - BOTTOM_V)) for p in pts if p.y + 78.8 > BOTTOM_V + 5)

b = Part.read(os.path.join(GEN, "step", "ctohm-bump.step"))           # phone coords
cover = Part.read(os.path.join(GEN, "step", "ctohm-cover.step"))
lid = Part.read(os.path.join(GEN, "step", "ctohm-lid.step"))
def rec_profile(off, v0, v1, z_bottom):
    """Receiver's front profile grown by off (block + half cylinder on top), extruded from v0 to v1; bottom at z_bottom."""
    blk = box(REC_U - REC_BW / 2 - off, REC_U + REC_BW / 2 + off, v0, v1, REC_Z0 - REC_BH - min(off, 0.1), z_bottom)
    return blk.fuse(Part.makeCylinder(REC_CR + off, v1 - v0, P(REC_U, v0, REC_Z0 - REC_BH), App.Vector(0, 1, 0)))
def led_solid(back, g=0.0):
    """5 mm LED grown by g: rim (flat side out, flush with the body), body, dome; back face at v back."""
    rim = Part.makeCylinder(RIM_R + g, RIM_T, P(LED_U, back, LED_Z), App.Vector(0, 1, 0))
    body = Part.makeCylinder(LED_R + g, LED_L - RIM_T - LED_R, P(LED_U, back + RIM_T, LED_Z), App.Vector(0, 1, 0))
    return rim.fuse([body, Part.makeSphere(LED_R + g, P(LED_U, back + LED_L - LED_R, LED_Z))])
LED_BACK = V1 + OUT_BY - LED_L
REC_FRONT = V1 + OUT_BY
REC_V = (REC_FRONT - REC_L, REC_FRONT)
# cover: notches in its outside face where the LED and the receiver run over it (user)
cover = cover.cut([led_solid(LED_BACK, NOTCH_CLEAR), rec_profile(NOTCH_CLEAR, REC_V[0], REC_FRONT + NOTCH_CLEAR, REC_Z0 + NOTCH_CLEAR)]).removeSplitter()
assert cover.isValid() and len(cover.Solids) == 1, (cover.isValid(), len(cover.Solids))
print("CHECK LED tip / receiver front at v %.2f (bump front %.2f); cover left under the notches: LED %.2f, receiver %.2f" % (
    REC_FRONT, V1, -1.6 - (LED_Z + LED_R + NOTCH_CLEAR), -1.6 - (REC_Z0 + NOTCH_CLEAR)))
CAP = (37.0 - 6.9, 37.0 + 6.9, min(LED_BACK, REC_V[0]) - 0.6, V1, ROOF + 0.2, LED_Z - LED_R - 0.4)   # u0 u1 v0 v1 z(in roof) z top (-7.6)
cap = round_top(box(*CAP, r=1.0), CAP[5], 0.5)
cone = Part.makeCone(FLASH_R, FLASH_R + (BUMP_IN - CAP[5] + 0.5) * math.tan(CONE), BUMP_IN - CAP[5] + 0.5, P(FLASH_U, FLASH_V, BUMP_IN + 0.1), App.Vector(0, 0, -1))
print("CHECK cap overlap with the flash cone %.3f mm3" % cap.common(cone).Volume)
# ridge height: the plane through the cover's bottom edge and the cap must touch the ridge's top (3 legs)
s_cap = slope(cap)
v_c = RIDGE[2] + RIDGE_TOP_R                               # the ridge top's lower edge touches first
RIDGE_H = -(FACE + s_cap * (v_c - BOTTOM_V)) + ROOF + RIDGE_MARGIN
ridge = round_top(box(RIDGE[0], RIDGE[1], RIDGE[2], RIDGE[3], ROOF + 0.2, ROOF - RIDGE_H, r=0.6), ROOF - RIDGE_H, RIDGE_TOP_R)
b = b.fuse([cap, ridge])
# LED: U-slot (body r 2.55) + rim pocket (flat side out, flush with the body), open to the phone side, through the front
led = Part.makeCylinder(LED_R + 0.05, 160 - LED_BACK, P(LED_U, LED_BACK - 0.2, LED_Z), App.Vector(0, 1, 0))
led = led.fuse([box(LED_U - LED_R - 0.05, LED_U + LED_R + 0.05, LED_BACK - 0.2, 160, LED_Z, BUMP_IN + 0.1),
                box(LED_U - RIM_R - 0.1, LED_U + RIM_R + 0.1, LED_BACK - 0.2, LED_BACK + RIM_T + 0.2, CAP[5] + RIM_ROOF, BUMP_IN + 0.1)])
b = b.cut(led)
b = b.cut(rec_profile(0.2, REC_V[0] - 0.2, 160, BUMP_IN + 0.1))           # bay, open to the phone side, out through the front
b = b.cut(cone)
b = b.removeSplitter()
assert b.isValid() and len(b.Solids) == 1, (b.isValid(), len(b.Solids))
# lid: cutout under the LED (it sits on the phone's back; the receiver sits on the lid, user)
lid = lid.cut(box(LED_U - RIM_R - 0.2, LED_U + RIM_R + 0.2, LED_BACK - 0.35, 160, BUMP_IN - 2, BUMP_IN + 1)).removeSplitter()
assert lid.isValid() and len(lid.Solids) == 1, (lid.isValid(), len(lid.Solids))

# checks: the resting plane touches the cover's bottom edge, the ridge and the cap; the LED and receiver stay inside
s_all = slope(b.fuse(cover))
z_cap = min(p.z - (FACE + s_all * (p.y + 78.8 - BOTTOM_V)) for p in cap.tessellate(0.02)[0])
print("CHECK ridge %.3f high (3 legs + %.2f); resting plane %.2f deg; ridge on the table: %s; cap clears the table by %.3f mm" % (
    RIDGE_H, RIDGE_MARGIN, math.degrees(math.atan(-s_all)), abs(slope(ridge) - s_all) < 1e-6, z_cap))
print("CHECK overlap bump/cover %.3f, bump/lid %.3f mm3" % (b.common(cover).Volume, b.common(lid).Volume))
b.exportStep(os.path.join(OUT, "step", "ir-bump.step")); lid.exportStep(os.path.join(OUT, "step", "ir-lid.step"))
for shape, name in ((b, "ir-bump"), (lid, "ir-lid")):
    p = shape.copy(); p.translate(App.Vector(0, 0, -p.BoundBox.ZMin))     # bump: cap on the bed; lid: flat
    f = os.path.join(OUT, name + ".stl"); MeshPart.meshFromShape(Shape=p, LinearDeflection=0.02, AngularDeflection=0.15).write(f)
    m = Mesh.Mesh(f); print("CHECK %s.stl solid %s nonmanifold %s components %d" % (name, m.isSolid(), m.hasNonManifolds(), m.countComponents()))
cover.exportStep(os.path.join(OUT, "step", "ir-cover.step"))             # outside face at Z -3.0, as the generic cover
f = os.path.join(OUT, "ir-cover.stl"); MeshPart.meshFromShape(Shape=cover, LinearDeflection=0.01, AngularDeflection=0.1).write(f)
m = Mesh.Mesh(f); print("CHECK ir-cover.stl solid %s nonmanifold %s components %d" % (m.isSolid(), m.hasNonManifolds(), m.countComponents()))
print("CHECK model/:", sorted(os.listdir(OUT)))
