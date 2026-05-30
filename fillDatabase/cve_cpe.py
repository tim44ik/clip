import psycopg
import os
from utils import iter_json_files

def main():
    conn = psycopg.connect(
        dbname=os.getenv('POSTGRES_DB', 'cve_db'),
        user=os.getenv('POSTGRES_USER', 'user'),
        password=os.getenv('POSTGRES_PASSWORD', 'pass'),
        host=os.getenv('POSTGRES_HOST', 'localhost'),
        port=os.getenv('POSTGRES_PORT', 5432)
    )
    conn.autocommit = False

    cve_folder = os.getenv('CVE_FOLDER', '/app/cve')
    with conn.cursor() as cur:
        cur.execute('SELECT cpe_name FROM cpe_products')
        existing_cpe_names = {row[0] for row in cur.fetchall()}

        for file_path, data in iter_json_files(cve_folder):
            for vuln in data.get('vulnerabilities', []):
                cve = vuln.get('cve', {})
                cve_id = cve.get('id')
                if not cve_id:
                    continue

                nodes = cve.get('configurations', {}).get('nodes', [])
                for node in nodes:
                    for cpem in node.get('cpeMatch', []):
                        criteria = cpem.get('criteria')
                        if not criteria or criteria not in existing_cpe_names:
                            continue
                        cur.execute("""
                            INSERT INTO cpe_cve (cpe_name, cve_id)
                            VALUES (%s, %s)
                            ON CONFLICT DO NOTHING
                        """, (criteria, cve_id))
        conn.commit()
    conn.close()

if __name__ == '__main__':
    main()