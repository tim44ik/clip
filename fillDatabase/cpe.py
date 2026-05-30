import psycopg
import os
from utils import iter_json_files, extract_product_version

def main():
    conn = psycopg.connect(
        dbname=os.getenv('POSTGRES_DB', 'cve_db'),
        user=os.getenv('POSTGRES_USER', 'user'),
        password=os.getenv('POSTGRES_PASSWORD', 'pass'),
        host=os.getenv('POSTGRES_HOST', 'localhost'),
        port=os.getenv('POSTGRES_PORT', 5432)
    )
    conn.autocommit = False

    cpe_folder = os.getenv('CPE_FOLDER', '/app/cpe')
    with conn.cursor() as cur:
        for file_path, data in iter_json_files(cpe_folder):
            for ms in data.get('matchStrings', []):
                matches = ms.get('matchString', {}).get('matches', [])
                for m in matches:
                    cpe_name = m.get('cpeName')
                    if not cpe_name:
                        continue
                    product, version = extract_product_version(cpe_name)
                    cur.execute("""
                        INSERT INTO cpe_products (cpe_name, product, version)
                        VALUES (%s, %s, %s)
                        ON CONFLICT (cpe_name) DO NOTHING
                    """, (cpe_name, product, version))
        conn.commit()
    conn.close()

if __name__ == '__main__':
    main()