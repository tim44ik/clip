import json
from pathlib import Path

def extract_product_version(cpe_name: str):
    parts = cpe_name.split(':')
    if len(parts) < 5:
        return None, None
    product = parts[3] if parts[3] != '*' else None
    version = parts[4] if parts[4] not in ('*', '-') else None
    return product, version

def iter_json_files(folder_path: str):
    folder = Path(folder_path)
    for file_path in folder.glob('**/*.json'):
        with open(file_path, 'r', encoding='utf-8') as f:
            data = json.load(f)
        yield file_path, data