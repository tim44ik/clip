import psycopg
import os
import time
from utils import iter_json_files, extract_vendor_product_version, log

def main():
    conn = psycopg.connect(
        dbname=os.getenv('POSTGRES_DB', 'cve_db'),
        user=os.getenv('POSTGRES_USER', 'user'),
        password=os.getenv('POSTGRES_PASSWORD', 'pass'),
        host=os.getenv('POSTGRES_HOST', 'localhost'),
        port=os.getenv('POSTGRES_PORT', 5432)
    )
    conn.autocommit = False
    log("Connected to PostgreSQL")

    cpe_folder = os.getenv('CPE_FOLDER', '/app/cpe')
    total_inserts = 0
    start_time = time.time()

    with conn.cursor() as cur:
        for file_path, data in iter_json_files(cpe_folder):
            file_inserts = 0
            for ms in data.get('matchStrings', []):
                matches = ms.get('matchString', {}).get('matches', [])
                for m in matches:
                    cpe_name = m.get('cpeName')
                    if not cpe_name:
                        continue
                    vendor, product, version = extract_vendor_product_version(cpe_name)
                    if vendor is None:
                        continue
                    cur.execute("""
                        INSERT INTO cpe (cpe_name, vendor, product, ver)
                        VALUES (%s, %s, %s, %s)
                        ON CONFLICT (cpe_name) DO NOTHING
                    """, (cpe_name, vendor, product, version))
                    file_inserts += 1
                    total_inserts += 1
                    if file_inserts % 10000 == 0:
                        log(f"  {file_path.name}: inserted {file_inserts} CPEs so far")
            conn.commit()
            log(f"Committed {file_inserts} CPEs from {file_path.name}")

    elapsed = time.time() - start_time
    log(f"Total CPEs inserted: {total_inserts} in {elapsed:.2f} seconds")
    conn.close()

if __name__ == '__main__':
    main()