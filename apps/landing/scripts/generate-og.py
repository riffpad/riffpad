from PIL import Image, ImageDraw, ImageFont

# Canvas
WIDTH, HEIGHT = 1200, 630
BG = "#eeefe9"
INK = "#23251d"
BODY = "#4d4f46"
ACCENT = "#f7a501"

img = Image.new("RGB", (WIDTH, HEIGHT), BG)
draw = ImageDraw.Draw(img)

# Load fonts
try:
    font_brand = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 52)
    font_h1 = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 56)
    font_desc = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", 30)
except Exception:
    font_brand = ImageFont.load_default()
    font_h1 = ImageFont.load_default()
    font_desc = ImageFont.load_default()

# Load logo
logo = Image.open("public/rounded_rect_rotated_shapes_only_favicon.png").convert("RGBA")
logo_size = 100
logo = logo.resize((logo_size, logo_size), Image.LANCZOS)

# Layout
left_margin = 90
top_margin = 110

# Paste logo
img.paste(logo, (left_margin, top_margin), logo)

# Brand name next to logo (vertically centered)
brand_x = left_margin + logo_size + 28
brand_y = top_margin + (logo_size - 52) // 2 + 2
draw.text((brand_x, brand_y), "Riffpad", font=font_brand, fill=INK)

# Headline below logo/brand area
headline_x = left_margin
headline_y = top_margin + logo_size + 48
draw.text((headline_x, headline_y), "The AI-native workspace", font=font_h1, fill=INK)
draw.text((headline_x, headline_y + 76), "in seconds.", font=font_h1, fill=INK)

# Description (wrapped)
desc_line1 = "Brainstorm, capture ideas, and build light prototypes"
desc_line2 = "anywhere—on your phone, tablet, or desktop. Riffpad turns"
desc_line3 = "a sentence into a running workspace, no setup required."
desc_y = headline_y + 180
line_spacing = 44
draw.text((headline_x, desc_y), desc_line1, font=font_desc, fill=BODY)
draw.text((headline_x, desc_y + line_spacing), desc_line2, font=font_desc, fill=BODY)
draw.text((headline_x, desc_y + line_spacing * 2), desc_line3, font=font_desc, fill=BODY)

# Save
img.save("public/og.png", "PNG", optimize=True)
print("Saved public/og.png")
