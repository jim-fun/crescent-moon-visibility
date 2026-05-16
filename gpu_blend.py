import sys
import os
import cv2
import numpy as np
from PIL import Image as PILImage, ImageDraw, ImageFont

def process_file(output_file, base_umat):
    # Load overlay
    overlay = cv2.imread(output_file, cv2.IMREAD_UNCHANGED)
    if overlay is None:
        print(f"Failed to load {output_file}")
        return

    # Upload to GPU memory (OpenCL T-API)
    overlay_umat = cv2.UMat(overlay)

    # Resize overlay on GPU
    overlay_umat = cv2.resize(overlay_umat, (3840, 2160), interpolation=cv2.INTER_NEAREST)

    # Extract Alpha and BGR channels
    # In OpenCV, channels can be split on GPU
    channels = cv2.split(overlay_umat)
    
    if len(channels) == 4:
        # We have an alpha channel
        b, g, r, a = channels
        
        # Merge BGR back
        fg_umat = cv2.merge([b, g, r])
        
        # Create alpha mask (0.0 to 1.0) and apply 60% blend
        # Since T-API has limited operations, we can do it via convertScaleAbs or multiply
        # It's faster to do: out = fg * (alpha * 0.6) + bg * (1.0 - alpha * 0.6)
        alpha_f = cv2.multiply(a, 1.0 / 255.0, dtype=cv2.CV_32F)
        alpha_blend = cv2.multiply(alpha_f, 0.6, dtype=cv2.CV_32F)
        
        # Prepare 3-channel alpha
        alpha_3c = cv2.merge([alpha_blend, alpha_blend, alpha_blend])
        ones_umat = cv2.UMat(np.ones((2160, 3840, 3), dtype=np.float32))
        inv_alpha_3c = cv2.subtract(ones_umat, alpha_3c)
        
        fg_f = cv2.multiply(fg_umat, 1.0, dtype=cv2.CV_32F)
        bg_f = cv2.multiply(base_umat, 1.0, dtype=cv2.CV_32F)
        
        out_f = cv2.add(cv2.multiply(fg_f, alpha_3c), cv2.multiply(bg_f, inv_alpha_3c))
        
        # Convert back to 8-bit
        out_umat = cv2.multiply(out_f, 1.0, dtype=cv2.CV_8U)
    else:
        out_umat = base_umat
        
    # Download from GPU to CPU
    out_img = out_umat.get()
    
    # Convert BGR to RGB for PIL
    out_img_rgb = cv2.cvtColor(out_img, cv2.COLOR_BGR2RGB)
    pil_img = PILImage.fromarray(out_img_rgb)
    draw = ImageDraw.Draw(pil_img, 'RGBA')
    
    # Legend background
    draw.rectangle([3100, 1820, 3820, 2140], fill=(255, 255, 255, 230))
    
    try:
        font_large = ImageFont.truetype("/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf", 36)
        font_title = ImageFont.truetype("/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf", 26)
        font_item = ImageFont.truetype("/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf", 22)
    except:
        font_large = ImageFont.load_default()
        font_title = font_large
        font_item = font_large
        
    date_str = os.path.basename(output_file).replace('.png', '')
    draw.text((3120, 1840), date_str, font=font_large, fill=(0, 0, 0, 255))
    draw.text((3120, 1885), "Visibility Zones:", font=font_title, fill=(0, 0, 0, 255))
    
    y = 1938
    colors = [
        ("#0080A0", "Not visible (even with telescope)"),
        ("#FFCC00", "Visible only with optical aid"),
        ("#00E6E6", "Visible to experienced observer"),
        ("#B3B300", "Easily visible to naked eye")
    ]
    
    for hex_col, text in colors:
        draw.text((3120, y), "●", font=font_item, fill=hex_col)
        draw.text((3160, y), text, font=font_item, fill=(0, 0, 0, 255))
        y += 40
        
    pil_img.save(output_file, "PNG", optimize=True)

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
        
    base = cv2.imread('map_nasa.png', cv2.IMREAD_COLOR)
    if base is None:
        print("Could not load map_nasa.png")
        sys.exit(1)
        
    # Pre-load base map to GPU memory
    base_umat = cv2.UMat(base)
        
    for f in sys.argv[1:]:
        process_file(f, base_umat)
