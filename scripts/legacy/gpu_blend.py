# gpu_blend.py — Legacy OpenCV + PIL blending script
#
# NOTE: This file is retained only for advanced users who specifically want
# the original OpenCV T-API GPU blending behavior.
# The default pipeline (since 2026) uses the pure-Go implementation in
# `internal/blend`, which removes all Python runtime dependencies.

import sys
import os
import cv2
import numpy as np
from PIL import Image as PILImage, ImageDraw, ImageFont

# Cross-platform font paths (bold and regular variants)
_FONT_BOLD_PATHS = [
    "/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf",
    "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
    "/usr/share/fonts/TTF/LiberationSans-Bold.ttf",
    "/System/Library/Fonts/Helvetica.ttc",  # macOS
    os.path.expanduser("~/.fonts/LiberationSans-Bold.ttf"),
    os.path.expanduser("~/.local/share/fonts/LiberationSans-Bold.ttf"),
]
_FONT_REG_PATHS = [p.replace("Bold.ttf", "Regular.ttf") for p in _FONT_BOLD_PATHS]
_FONT_REG_PATHS = [p for p in _FONT_REG_PATHS if "Bold" not in p]

def _find_font(size: int, weight: str = "normal") -> ImageFont.FreeTypeFont:
    """Try loading a TrueType font from multiple platform paths, fall back to default."""
    paths = _FONT_BOLD_PATHS if weight == "bold" else _FONT_REG_PATHS
    for p in paths:
        try:
            return ImageFont.truetype(p, size)
        except (IOError, OSError):
            continue
    return ImageFont.load_default()

def process_file(output_file, base_umat, moon_age, ones_umat):
    # Load overlay: try raw RGBA binary first, then PNG
    rgba_data = None
    # Handle both "YYYY-MM-DD.png" and "YYYY-MM-DD" formats
    base_name = output_file.replace('.png', '') if output_file.endswith('.png') else output_file
    bin_path = base_name + ".bin"

    if os.path.exists(bin_path):
        # Load raw RGBA binary data from C++ renderer
        with open(bin_path, 'rb') as f:
            rgba_data = np.frombuffer(f.read(), dtype=np.uint8)
        # Determine dimensions from data size (PIXEL_PER_DEGREE_LON=10, PIXEL_PER_DEGREE_LAT=12)
        bytes_per_pixel = 4
        pixel_count = rgba_data.size // bytes_per_pixel
        # 3600x2160 for CPU renderer (360*10 x 180*12)
        if pixel_count == 3600 * 2160:
            overlay_pil = PILImage.fromarray(rgba_data.reshape(2160, 3600, 4), 'RGBA')
        elif pixel_count == 1440 * 720:
            overlay_pil = PILImage.fromarray(rgba_data.reshape(720, 1440, 4), 'RGBA')
        elif pixel_count == 3840 * 2160:
            overlay_pil = PILImage.fromarray(rgba_data.reshape(2160, 3840, 4), 'RGBA')
        else:
            raise ValueError(f"Unexpected binary size: {rgba_data.size} bytes ({pixel_count} pixels)")
    else:
        # Fallback: load PNG with PIL (preserves alpha)
        # GPU renderer writes PNG without .png extension; CPU writes .bin
        png_path = base_name + ".png" if not output_file.endswith('.png') else output_file
        if not os.path.exists(png_path) and os.path.exists(base_name):
            png_path = base_name
        try:
            overlay_pil = PILImage.open(png_path).convert('RGBA')
        except Exception:
            overlay_pil = PILImage.open(png_path)
            if overlay_pil.mode != 'RGBA':
                overlay_pil = overlay_pil.convert('RGBA')

    width, height = overlay_pil.size

    # Convert to numpy + split into BGR and A
    rgba_arr = np.array(overlay_pil)
    r = rgba_arr[:, :, 0]
    g = rgba_arr[:, :, 1]
    b = rgba_arr[:, :, 2]
    a = rgba_arr[:, :, 3]

    # Skip blending if the overlay has no visibility classifications at all.
    # Some young-crescent dates pass the orchestrator's pre-render illumination
    # threshold (because the moon has reached 0.2% illumination somewhere on
    # Earth at sunset) but still render with zero Yallop A-E pixels because the
    # geometry (declination, observer latitudes) gives every visible-side
    # observer either an 'F' (value < -0.293) or an 'I' (moonset before sunset).
    # Writing a "blank" map with just the base photo and a legend is misleading.
    A, B, C, D, E = (0, 204, 204), (0, 179, 179), (255, 255, 26), (230, 230, 0), (179, 179, 0)
    visible_px = (
        ((r == A[0]) & (g == A[1]) & (b == A[2]) & (a == 255)).sum() +
        ((r == B[0]) & (g == B[1]) & (b == B[2]) & (a == 255)).sum() +
        ((r == C[0]) & (g == C[1]) & (b == C[2]) & (a == 255)).sum() +
        ((r == D[0]) & (g == D[1]) & (b == D[2]) & (a == 255)).sum() +
        ((r == E[0]) & (g == E[1]) & (b == E[2]) & (a == 255)).sum()
    )
    if visible_px < 100:
        print(f"[skip] {os.path.basename(base_name)}: no visibility zones ({visible_px} px) — overlay too young to render")
        if os.path.exists(bin_path):
            os.remove(bin_path)
        elif os.path.exists(base_name) and base_name != base_name + ".webp":
            os.remove(base_name)
        return

    # Resize on CPU then upload to GPU (use LANCZOS for smooth edges)
    overlay_resized = overlay_pil.resize((3840, 2160), PILImage.LANCZOS)
    overlay_arr = np.array(overlay_resized)
    # Convert RGBA to BGRA for OpenCV
    bgr_a = cv2.merge([overlay_arr[:,:,2], overlay_arr[:,:,1], overlay_arr[:,:,0], overlay_arr[:,:,3]])
    overlay_umat = cv2.UMat(bgr_a)

    # Split into BGR and A
    channels = cv2.split(overlay_umat)
    fg_umat = cv2.merge([channels[0], channels[1], channels[2]])
    a_umat = channels[3]

    # Create alpha mask (0.0 to 1.0) and apply 60% blend so the NASA base map
    # remains clearly visible through the visibility-zone overlay.
    alpha_f = cv2.multiply(a_umat, 1.0 / 255.0, dtype=cv2.CV_32F)
    alpha_blend = cv2.multiply(alpha_f, 0.6, dtype=cv2.CV_32F)

    # Prepare 3-channel alpha
    alpha_3c = cv2.merge([alpha_blend, alpha_blend, alpha_blend])
    inv_alpha_3c = cv2.subtract(ones_umat, alpha_3c)

    fg_f = cv2.multiply(fg_umat, 1.0, dtype=cv2.CV_32F)
    bg_f = cv2.multiply(base_umat, 1.0, dtype=cv2.CV_32F)

    out_f = cv2.add(cv2.multiply(fg_f, alpha_3c), cv2.multiply(bg_f, inv_alpha_3c))

    # Convert back to 8-bit
    out_umat = cv2.multiply(out_f, 1.0, dtype=cv2.CV_8U)

    # Download from GPU to CPU
    out_img = out_umat.get()

    # Convert BGR to RGB for PIL
    out_img_rgb = cv2.cvtColor(out_img, cv2.COLOR_BGR2RGB)
    pil_img = PILImage.fromarray(out_img_rgb, 'RGB')
    draw = ImageDraw.Draw(pil_img, 'RGB')

    # Legend background
    draw.rectangle([3080, 1780, 3820, 2150], fill=(255, 255, 255))

    font_large = _find_font(size=36, weight="bold")
    font_title = _find_font(size=22)
    font_item  = _find_font(size=20)

    date_str = os.path.basename(base_name)
    draw.text((3100, 1790), date_str, font=font_large, fill=(0, 0, 0))
    draw.text((3100, 1835), "Yallop visibility (Q):", font=font_title, fill=(0, 0, 0))

    # Colors here mirror the renderer's actual output (visibility.cc:223-242).
    # 'C', 'D', 'E' are yellow/olive (not cyan as legacy comments suggested).
    y = 1870
    entries = [
        ("#00CCCC", "A: Easily visible (naked eye)"),
        ("#00B3B3", "B: Visible, perfect conditions"),
        ("#FFFF1A", "C: May need optical aid"),
        ("#E6E600", "D: Will need optical aid"),
        ("#B3B300", "E: Telescope only"),
        ("#FF0000", "● First naked-eye visibility"),
        ("#0000FF", "● First telescope visibility"),
    ]
    # Draw markers as native vector geometry rather than Unicode glyphs so they
    # render identically regardless of the available font's coverage.
    for hex_col, text in entries:
        is_circle = text.startswith("●")
        label = text[2:] if is_circle else text
        if is_circle:
            draw.ellipse([3100, y + 2, 3118, y + 20], fill=hex_col)
        else:
            draw.rectangle([3100, y + 2, 3118, y + 20], fill=hex_col)
        draw.text((3135, y), label, font=font_item, fill=(0, 0, 0))
        y += 32

    # Save as WEBP with high quality for better compression
    webp_output = base_name + ".webp"
    pil_img.save(webp_output, "WEBP", quality=98, method=6)
    # Clean up source overlay (CPU .bin or GPU no-extension PNG)
    if os.path.exists(bin_path):
        os.remove(bin_path)
    elif os.path.exists(base_name) and base_name != webp_output:
        os.remove(base_name)

if __name__ == "__main__":
    if not sys.argv[1:]:
        print("No files to process.")
        sys.exit(0)

    # Enable OpenCV OpenCL T-API for Mac, AMD, Nvidia GPU Acceleration
    cv2.ocl.setUseOpenCL(True)
    if cv2.ocl.useOpenCL():
        print(f"GPU Acceleration Enabled via OpenCL: {cv2.ocl.Device.getDefault().name()}")
    else:
        print("GPU Acceleration not available, falling back to highly optimized CPU.")

    # Search for base map in multiple locations (supports both old root location
    # and the new organized data/ directory).
    base_map_paths = [
        'map_nasa.png',
        'data/map_nasa.png',
        os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'map_nasa.png'),
        os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'data', 'map_nasa.png'),
        os.path.join(os.path.dirname(os.path.abspath(__file__)), 'map_nasa.png'),
        os.path.join(os.path.dirname(os.path.abspath(__file__)), 'data', 'map_nasa.png'),
    ]
    base = None
    for p in base_map_paths:
        base = cv2.imread(p, cv2.IMREAD_COLOR)
        if base is not None:
            print(f"[GPU] Base map loaded from: {p} ({base.shape[1]}x{base.shape[0]})")
            break
    if base is None:
        print(f"Could not find map_nasa.png in any of: {base_map_paths}")
        sys.exit(1)

    # Pre-load base map to GPU memory
    base_umat = cv2.UMat(base)

    # Pre-allocate ones mask once (shared across all files)
    ones_umat = cv2.UMat(np.ones((2160, 3840, 3), dtype=np.float32))

    gpu_dev = None
    if cv2.ocl.useOpenCL():
        gpu_dev = cv2.ocl.Device.getDefault()
        print(f"[GPU] Acceleration enabled via OpenCL: {gpu_dev.name()}")
    else:
        print("[GPU] Acceleration not available — falling back to optimized CPU.")

    for arg in sys.argv[1:]:
        parts = arg.split('|')
        f = parts[0]
        moon_age = parts[1] if len(parts) > 1 else "?"
        process_file(f, base_umat, moon_age, ones_umat)

    if gpu_dev:
        print(f"[GPU] Device: {gpu_dev.name()} (vendorID={gpu_dev.vendorID()})")
