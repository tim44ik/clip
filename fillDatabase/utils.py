import json
import time
from pathlib import Path

def log(msg, level="INFO"):
    print(f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] [{level}] {msg}")

def extract_product_version(cpe_name: str):
    parts = cpe_name.split(':')
    if len(parts) < 5:
        return None, None
    product = parts[3] if parts[3] != '*' else None
    version = parts[4] if parts[4] not in ('*', '-') else None
    return product, version

def extract_vendor_product_version(cpe_name: str):
    parts = cpe_name.split(':')
    if len(parts) < 6:
        return None, None, None
    vendor = parts[3] if parts[3] != '*' else None
    product = parts[4] if parts[4] != '*' else None
    version = parts[5] if parts[5] not in ('*', '-') else None
    return vendor, product, version

def iter_json_files(folder_path: str):
    folder = Path(folder_path)
    files = list(folder.glob('**/*.json'))
    total = len(files)
    log(f"Found {total} JSON files in {folder_path}")
    for idx, file_path in enumerate(files, 1):
        log(f"Processing file {idx}/{total}: {file_path.name}")
        start = time.time()
        with open(file_path, 'r', encoding='utf-8') as f:
            data = json.load(f)
        elapsed = time.time() - start
        log(f"Loaded {file_path.name} in {elapsed:.2f} sec")
        yield file_path, data