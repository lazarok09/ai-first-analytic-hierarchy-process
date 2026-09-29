#!/usr/bin/env python3
"""Render the README loops in docs/assets/.

Numbers are the committed examples/vendor-selection workspace:
criteria weights 0.64 / 0.26 / 0.10, the Saaty criteria matrix, CR 0.03,
and the global ranking Acme 0.41, Initech 0.33, Globex 0.26.
"""

from __future__ import annotations

import math
from pathlib import Path

import numpy as np
from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "docs" / "assets"

W, H = 640, 400
SS = 2
FRAMES = 36
FRAME_MS = 90

FONT_BOLD = "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"
FONT_REG = "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"

# (name, weight, accent)
CRITERIA = (
    ("Cost", 0.64, (242, 176, 68)),
    ("Quality", 0.26, (64, 214, 196)),
    ("Risk", 0.10, (176, 156, 242)),
)
ALTS = (
    ("Acme", 0.41, (242, 176, 68)),
    ("Initech", 0.33, (64, 214, 196)),
    ("Globex", 0.26, (176, 156, 242)),
)
# a_ij: importance of row over column. Order is Cost, Quality, Risk.
MATRIX = (
    ("1", "3", "5"),
    ("1/3", "1", "3"),
    ("1/5", "1/3", "1"),
)
# Off-diagonal cells, judgments first, then the reciprocals they imply.
CELL_ORDER = ((0, 1), (0, 2), (1, 2), (1, 0), (2, 0), (2, 1))


def font(size: int, bold: bool = True) -> ImageFont.FreeTypeFont:
    return ImageFont.truetype(FONT_BOLD if bold else FONT_REG, size * SS)


def clamp(v: float, lo: float, hi: float) -> float:
    return lo if v < lo else hi if v > hi else v


def mix(a: tuple[int, int, int], b: tuple[int, int, int], t: float) -> tuple[int, int, int]:
    return tuple(int(a[i] + (b[i] - a[i]) * t) for i in range(3))


def shade(rgb: tuple[int, int, int], k: float) -> tuple[int, int, int]:
    return tuple(int(clamp(c * k, 0, 255)) for c in rgb)


class Camera:
    def __init__(self, yaw: float, pitch: float, dist: float, fov: float, vshift: float = 0) -> None:
        self.yaw = yaw
        self.pitch = pitch
        self.dist = dist
        self.fov = fov * SS
        self.vshift = vshift * SS
        self.cy, self.sy = math.cos(yaw), math.sin(yaw)
        self.cp, self.sp = math.cos(pitch), math.sin(pitch)

    def to_cam(self, p: tuple[float, float, float]) -> tuple[float, float, float]:
        x, y, z = p
        x, z = x * self.cy + z * self.sy, -x * self.sy + z * self.cy
        y, z = y * self.cp - z * self.sp, y * self.sp + z * self.cp
        return x, y, z + self.dist

    def project(self, p: tuple[float, float, float]) -> tuple[float, float, float]:
        x, y, z = self.to_cam(p)
        sx = (W * SS) / 2 + x / z * self.fov
        sy = (H * SS) / 2 - y / z * self.fov - 6 * SS + self.vshift
        return sx, sy, z


def facing_camera(cam_pts: list[tuple[float, float, float]]) -> bool:
    a, b, c = cam_pts[0], cam_pts[1], cam_pts[2]
    ux, uy = b[0] - a[0], b[1] - a[1]
    vx, vy = c[0] - a[0], c[1] - a[1]
    return (ux * vy - uy * vx) < 0


def box_faces(
    center: tuple[float, float, float],
    size: tuple[float, float, float],
    accent: tuple[int, int, int],
    lift: float = 1.0,
) -> list[tuple[list[tuple[float, float, float]], tuple[int, int, int]]]:
    cx, cy, cz = center
    w, h, d = size
    x0, x1 = cx - w / 2, cx + w / 2
    y0, y1 = cy - h / 2, cy + h / 2
    z0, z1 = cz - d / 2, cz + d / 2
    c = [
        (x0, y0, z0),
        (x1, y0, z0),
        (x1, y1, z0),
        (x0, y1, z0),
        (x0, y0, z1),
        (x1, y0, z1),
        (x1, y1, z1),
        (x0, y1, z1),
    ]
    body = mix((18, 26, 44), accent, 0.55)
    top = shade(mix((40, 52, 78), accent, 0.82), lift)
    faces = (
        ([c[0], c[1], c[5], c[4]], shade(body, 0.62)),
        ([c[3], c[2], c[6], c[7]], top),
        ([c[0], c[3], c[2], c[1]], shade(body, 0.78)),
        ([c[4], c[5], c[6], c[7]], shade(body, 1.08)),
        ([c[0], c[4], c[7], c[3]], shade(body, 0.9)),
        ([c[1], c[2], c[6], c[5]], shade(body, 0.74)),
    )
    return faces


_BG: Image.Image | None = None


def background() -> Image.Image:
    global _BG
    if _BG is not None:
        return _BG.copy()
    yy, xx = np.mgrid[0 : H * SS, 0 : W * SS]
    nx = (xx - W * SS * 0.5) / (W * SS)
    ny = (yy - H * SS * 0.46) / (H * SS)
    r = np.sqrt(nx * nx * 0.85 + ny * ny)
    glow = np.exp(-((r / 0.72) ** 2))
    vig = np.clip(1.02 - r * 0.55, 0.55, 1)
    arr = np.zeros((H * SS, W * SS, 3), np.float32)
    arr[:, :, 0] = 9 + glow * 10
    arr[:, :, 1] = 13 + glow * 18
    arr[:, :, 2] = 24 + glow * 28
    arr *= vig[:, :, None]
    _BG = Image.fromarray(np.clip(arr, 0, 255).astype(np.uint8), "RGB").convert("RGBA")
    return _BG.copy()


def draw_grid(base: Image.Image, cam: Camera, y: float) -> None:
    layer = Image.new("RGBA", base.size, (0, 0, 0, 0))
    draw = ImageDraw.Draw(layer)
    span, step = 5.2, 1.3
    col = (38, 52, 78, 210)
    xs = [i * step for i in range(-4, 5)]
    for x in xs:
        a = cam.project((x, y, -span))
        b = cam.project((x, y, span))
        draw.line([(a[0], a[1]), (b[0], b[1])], fill=col, width=SS)
    for z in xs:
        a = cam.project((-span, y, z))
        b = cam.project((span, y, z))
        draw.line([(a[0], a[1]), (b[0], b[1])], fill=col, width=SS)
    base.alpha_composite(layer)


def paint(
    base: Image.Image,
    cam: Camera,
    solids: list[tuple[tuple[float, float, float], tuple[float, float, float], tuple[int, int, int], float]],
) -> None:
    faces: list[tuple[float, list[tuple[float, float]], tuple[int, int, int]]] = []
    for center, size, accent, lift in solids:
        for pts, col in box_faces(center, size, accent, lift):
            cam_pts = [cam.to_cam(p) for p in pts]
            if not facing_camera(cam_pts):
                continue
            proj = [(cam.project(p)[0], cam.project(p)[1]) for p in pts]
            depth = sum(p[2] for p in cam_pts) / len(cam_pts)
            faces.append((depth, proj, col))
    faces.sort(key=lambda item: -item[0])
    layer = Image.new("RGBA", base.size, (0, 0, 0, 0))
    draw = ImageDraw.Draw(layer)
    for _depth, poly, col in faces:
        draw.polygon(poly, fill=(*col, 255))
        draw.line(poly + [poly[0]], fill=(*shade(col, 1.45), 230), width=max(2, SS))
    base.alpha_composite(layer)


def blob(
    base: Image.Image,
    cam: Camera,
    center: tuple[float, float, float],
    width: float,
    y: float,
) -> None:
    p = cam.project((center[0], y, center[2]))
    rx = max(10 * SS, width * cam.fov / p[2] * 0.42)
    ry = rx * 0.32
    layer = Image.new("RGBA", base.size, (0, 0, 0, 0))
    ImageDraw.Draw(layer).ellipse(
        [p[0] - rx, p[1] - ry, p[0] + rx, p[1] + ry],
        fill=(0, 0, 0, 80),
    )
    base.alpha_composite(layer)


def text(
    base: Image.Image,
    xy: tuple[float, float],
    label: str,
    fnt: ImageFont.FreeTypeFont,
    fill: tuple[int, int, int] = (246, 248, 252),
) -> None:
    layer = Image.new("RGBA", base.size, (0, 0, 0, 0))
    ImageDraw.Draw(layer).text(xy, label, font=fnt, fill=(*fill, 255), anchor="mm")
    base.alpha_composite(layer)


def edge(
    base: Image.Image,
    cam: Camera,
    a: tuple[float, float, float],
    b: tuple[float, float, float],
    color: tuple[int, int, int],
    pulse: float,
) -> None:
    pa, pb = cam.project(a), cam.project(b)
    layer = Image.new("RGBA", base.size, (0, 0, 0, 0))
    draw = ImageDraw.Draw(layer)
    draw.line([(pa[0], pa[1]), (pb[0], pb[1])], fill=(*color, 170), width=SS * 2)
    u = pulse % 1.0
    x = pa[0] + (pb[0] - pa[0]) * u
    y = pa[1] + (pb[1] - pa[1]) * u
    r = 3.4 * SS
    draw.ellipse([x - r, y - r, x + r, y + r], fill=(255, 248, 230, 240))
    base.alpha_composite(layer)


def banner(base: Image.Image, label: str) -> None:
    text(base, (W * SS / 2, 20 * SS), label, font(14, bold=False), (186, 198, 220))


def foot(base: Image.Image, label: str) -> None:
    text(base, (W * SS / 2, H * SS - 16 * SS), label, font(14, bold=False), (186, 198, 220))


def finish(frame: Image.Image) -> Image.Image:
    return frame.resize((W, H), Image.Resampling.LANCZOS).convert("RGB")


def save_gif(frames: list[Image.Image], path: Path) -> None:
    pal = frames[0].quantize(colors=80, method=Image.Quantize.MEDIANCUT)
    images = [f.quantize(palette=pal, dither=Image.Dither.NONE) for f in frames]
    images[0].save(
        path,
        save_all=True,
        append_images=images[1:],
        duration=FRAME_MS,
        loop=0,
        optimize=True,
        disposal=1,
    )


def scene_hierarchy(i: int) -> Image.Image:
    """Goal branches to every criterion. Alternatives are one shared shortlist."""
    cam = Camera(0.52, 0.50, 13.2, 620)
    img = background()
    floor = -2.05
    draw_grid(img, cam, floor)

    goal = (0.0, 2.05, 0.0)
    crits = []
    for k, (name, weight, accent) in enumerate(CRITERIA):
        x = (k - 1) * 2.45
        width = 1.05 + weight * 1.7
        crits.append((name, weight, accent, (x, 0.35, 0.0), (width, 0.62, 0.95)))
    rail_y = -1.05
    alts = []
    for k, (name, weight, accent) in enumerate(ALTS):
        x = (k - 1) * 2.15
        alts.append((name, weight, accent, (x, -1.62, 1.15), (1.55, 0.40, 0.72)))

    solids: list = [((goal[0], goal[1], goal[2]), (3.35, 0.52, 1.15), (214, 222, 236), 1.0)]
    for _n, _w, accent, center, size in crits:
        solids.append((center, size, accent, 1.0))
    solids.append(((0.0, rail_y, 0.15), (7.4, 0.16, 0.55), (150, 168, 196), 1.0))
    for _n, _w, accent, center, size in alts:
        solids.append((center, size, accent, 1.0))

    for center, size, _a, _l in solids:
        blob(img, cam, center, size[0], floor)
    paint(img, cam, solids)

    pulse = (i + 0.5) / FRAMES
    for k, (_n, _w, accent, center, size) in enumerate(crits):
        edge(
            img,
            cam,
            (goal[0], goal[1] - 0.28, goal[2]),
            (center[0], center[1] + size[1] / 2, center[2]),
            accent,
            pulse + k * 0.08,
        )
        edge(
            img,
            cam,
            (center[0], center[1] - size[1] / 2, center[2]),
            (center[0], rail_y + 0.1, 0.15),
            accent,
            pulse + 0.45 + k * 0.08,
        )

    text(img, cam.project((goal[0], goal[1] + 0.08, goal[2]))[:2], "Choose a vendor", font(16))
    for name, weight, _accent, center, _size in crits:
        p = cam.project((center[0], center[1] + 0.08, center[2]))
        text(img, (p[0], p[1] - 14 * SS), name, font(14))
        text(img, (p[0], p[1] + 16 * SS), f"{weight:.2f}", font(12, bold=False), (232, 238, 248))
    for name, _w, _a, center, _s in alts:
        text(img, cam.project((center[0], center[1] + 0.02, center[2]))[:2], name, font(13))
    banner(img, "goal  →  criteria  →  shortlist")
    return finish(img)


def scene_pairwise(i: int) -> Image.Image:
    """Committed criteria matrix. Gold cells are judgments; teal cells are reciprocals."""
    cam = Camera(0.36, 1.02, 12.4, 820, vshift=28)
    img = background()
    draw_grid(img, cam, y=-1.35)
    active = CELL_ORDER[(i * len(CELL_ORDER)) // FRAMES]
    labels = [c[0] for c in CRITERIA]
    accents = [c[2] for c in CRITERIA]
    gap = 1.48
    solids = []
    cells = []
    for r in range(3):
        for c in range(3):
            hot = (r, c) == active
            if r == c:
                accent = (198, 208, 222)
            elif c > r:
                accent = (242, 176, 68)
            else:
                accent = (64, 214, 196)
            lift = 1.18 if hot else 1.0
            y = 0.28 if hot else 0.0
            center = ((c - 1) * gap + 0.35, y, (1 - r) * gap)
            solids.append((center, (1.22, 0.26, 1.22), accent, lift))
            cells.append((MATRIX[r][c], center, hot))
    for center, size, _a, _l in solids:
        blob(img, cam, center, size[0], -1.35)
    paint(img, cam, solids)

    projected: list[list[tuple[float, float]]] = [[(0.0, 0.0)] * 3 for _ in range(3)]
    for r in range(3):
        for c in range(3):
            value, center, hot = cells[r * 3 + c]
            p = cam.project((center[0], center[1] + 0.32, center[2]))
            projected[r][c] = (p[0], p[1])
            text(img, (p[0], p[1]), value, font(20 if hot else 16))

    # Axis titles in screen space so perspective cannot pile them on one corner.
    for c, name in enumerate(labels):
        xs = [projected[r][c][0] for r in range(3)]
        ys = [projected[r][c][1] for r in range(3)]
        text(img, (sum(xs) / 3, min(ys) - 58 * SS), name, font(14), accents[c])
    for r, name in enumerate(labels):
        xs = [projected[r][c][0] for c in range(3)]
        ys = [projected[r][c][1] for c in range(3)]
        text(img, (min(xs) - 96 * SS, sum(ys) / 3), name, font(14), accents[r])
    foot(img, "CR 0.03")
    return finish(img)


def scene_ranking(i: int) -> Image.Image:
    cam = Camera(0.48, 0.40, 12.2, 700)
    img = background()
    floor = -2.15
    draw_grid(img, cam, floor)
    # Sheen travels up the winning bar and wraps.
    u = (i + 0.5) / FRAMES
    solids = [((0.0, floor + 0.08, 0.1), (7.3, 0.16, 2.15), (160, 174, 198), 1.0)]
    metas = []
    winner_h = 0.0
    winner_x = 0.0
    for k, (name, weight, accent) in enumerate(ALTS):
        h = 0.7 + weight * 6.6
        x = (k - 1) * 2.2
        center = (x, floor + 0.16 + h / 2, 0.15)
        solids.append((center, (1.38, h, 1.05), accent, 1.0))
        metas.append((name, weight, accent, center, h, k == 0))
        if k == 0:
            winner_h = h
            winner_x = x
    for center, size, _a, _l in solids:
        blob(img, cam, center, max(size[0], 0.4), floor)
    paint(img, cam, solids)
    # Screen-space bead. A 3D cube inside the bar was covered on every frame,
    # so GIF optimization collapsed the loop to one picture.
    # Same x as Acme, on its front face, nudged left so perspective doesn't slide it onto Initech.
    face_z = 0.15 + 0.55
    lo = cam.project((winner_x - 0.28, floor + 0.55, face_z))
    hi = cam.project((winner_x - 0.28, floor + winner_h - 0.08, face_z))
    bx = lo[0] + (hi[0] - lo[0]) * u
    by = lo[1] + (hi[1] - lo[1]) * u
    bead = Image.new("RGBA", img.size, (0, 0, 0, 0))
    r = 6 * SS
    ImageDraw.Draw(bead).ellipse([bx - r, by - r, bx + r, by + r], fill=(255, 236, 190, 255))
    img.alpha_composite(bead)

    for name, weight, accent, center, h, winner in metas:
        top = cam.project((center[0], center[1] + h / 2 + 0.38, center[2]))
        text(img, (top[0], top[1]), f"{weight:.2f}", font(16))
        foot = cam.project((center[0], floor + 0.02, center[2] + 1.15))
        text(img, (foot[0], foot[1]), f"{name}  #1" if winner else name, font(14), accent if winner else (230, 236, 244))
    banner(img, "global weights")
    return finish(img)


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    scenes = (
        ("hierarchy.gif", scene_hierarchy),
        ("pairwise.gif", scene_pairwise),
        ("ranking.gif", scene_ranking),
    )
    for name, fn in scenes:
        frames = [fn(i) for i in range(FRAMES)]
        path = OUT / name
        save_gif(frames, path)
        print(f"{path.relative_to(ROOT)}  {path.stat().st_size // 1024} KiB")


if __name__ == "__main__":
    main()
