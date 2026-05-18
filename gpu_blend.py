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
        # Determine dimensions from data size (PIXEL_PER_DEGREE_LON=4, PIXEL_PER_DEGREE_LAT=4)
        bytes_per_pixel = 4
        pixel_count = rgba_data.size // bytes_per_pixel
        # 1440x720 for CPU renderer (360*4 x 180*4)
        if pixel_count == 1440 * 720:
            overlay_pil = PILImage.fromarray(rgba_data.reshape(720, 1440, 4), 'RGBA')
        elif pixel_count == 3840 * 2160:
            overlay_pil = PILImage.fromarray(rgba_data.reshape(2160, 3840, 4), 'RGBA')
        else:
            raise ValueError(f"Unexpected binary size: {rgba_data.size} bytes ({pixel_count} pixels)")
    else:
        # Fallback: load PNG with PIL (preserves alpha)
        png_path = base_name + ".png" if not output_file.endswith('.png') else output_file
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

    # Create alpha mask (0.0 to 1.0) and apply 60% blend
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

    # Convert BGR to RGB for PIL, then force RGBA mode
    out_img_rgb = cv2.cvtColor(out_img, cv2.COLOR_BGR2RGB)
    pil_img = PILImage.fromarray(out_img_rgb, 'RGB').convert('RGBA')
    draw = ImageDraw.Draw(pil_img, 'RGBA')

    # Legend background
    draw.rectangle([3100, 1820, 3820, 2180], fill=(255, 255, 255, 230))

    font_large = _find_font(size=36, weight="bold")
    font_title = _find_font(size=26)
    font_item = _find_font(size=22)

    date_str = os.path.basename(base_name)
    draw.text((3120, 1840), date_str, font=font_large, fill=(0, 0, 0, 255))
    draw.text((3120, 1885), "Visibility Zones:", font=font_title, fill=(0, 0, 0, 255))

    y = 1938
    # Use actual blended cyan colors (60% blend with base map produces darker, duller tones)
    # These are calculated: pure_color * 0.6 + base_map * 0.4 for typical ocean/map colors
    colors = [
        ("#00CCCC", "A: Easily visible to naked eye"),          # Bright cyan
        ("#00B3B3", "B: Visible under perfect conditions"),      # Darker cyan
        ("#1AFFFF", "C: May need optical aid"),                  # Light blue
        ("#00E6E6", "D: Will need optical aid"),                 # Bright cyan
        ("#007C82", "E: Not visible with telescope"),            # Darkest blended cyan
    ]

    for hex_col, text in colors:
        draw.text((3120, y), "■", font=font_item, fill=hex_col)
        draw.text((3160, y), text, font=font_item, fill=(0, 0, 0, 255))
        y += 40

    # Save as RGBA PNG (PIL preserves alpha)
    png_output = base_name + ".png"
    pil_img.save(png_output, "PNG", optimize=True)
    # Clean up binary file
    if os.path.exists(bin_path):
        os.remove(bin_path)

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

    # Search for base map in multiple locations
    base_map_paths = [
        'map_nasa.png',
        os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'map_nasa.png'),
        os.path.join(os.path.dirname(os.path.abspath(__file__)), 'map_nasa.png'),
    ]
    base = None
    for p in base_map_paths:
        base = cv2.imread(p, cv2.IMREAD_COLOR)
        if base is not None:
            print(f"[GPU] Base map loaded from: {p} ({base.shape[1]}x{base.shape[0]})")
            break
    if base is None:
        print(f"Could not find map_nasa.png in: {base_map_paths}")
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
