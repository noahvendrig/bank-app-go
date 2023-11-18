import selenium
import os

cwd = os.getcwd()

FOLDER = "scraped_images"
os.makedirs(
    name=f"{cwd}/{FOLDER}",
    exist_ok=True
)

