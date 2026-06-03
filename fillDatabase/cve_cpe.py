import psycopg
import os
import re
import time
from utils import iter_json_files, extract_vendor_product_version, log

def parse_version(ver_str):
    if not ver_str or ver_str == '*':
        return ()
    parts = re.split(r'\.', ver_str)
    result = []
    for p in parts:
        if p.isdigit():
            result.append(int(p))
        else:
            result.append(p)
    return tuple(result)

def version_in_range(ver_tuple, start_incl, start_excl, end_incl, end_excl):
    if start_incl is not None and ver_tuple < start_incl:
        return False
    if start_excl is not None and ver_tuple <= start_excl:
        return False
    if end_incl is not None and ver_tuple > end_incl:
        return False
    if end_excl is not None and ver_tuple >= end_excl:
        return False
    return True

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

    # Загружаем все CPE в память
    with conn.cursor() as cur:
        cur.execute("SELECT cpe_name, vendor, product, ver FROM cpe")
        cpe_map = {}
        for cpe_name, vendor, product, version in cur.fetchall():
            if vendor is None or product is None:
                continue
            key = (vendor, product)
            ver_tuple = parse_version(version) if version else ()
            cpe_map.setdefault(key, []).append((cpe_name, ver_tuple))
    log(f"Loaded {sum(len(v) for v in cpe_map.values())} CPE entries into memory")

    cve_folder = os.getenv('CVE_FOLDER', '/app/cve')
    total_relations = 0
    start_time = time.time()

    with conn.cursor() as cur:
        processed_criteria = {}
        for file_path, data in iter_json_files(cve_folder):
            file_relations = 0
            for vuln in data.get('vulnerabilities', []):
                cve_data = vuln.get('cve')
                if not isinstance(cve_data, dict):
                    continue
                cve_id = cve_data.get('id')
                if not cve_id:
                    continue

                configurations = cve_data.get('configurations', {})
                if isinstance(configurations, dict):
                    nodes = configurations.get('nodes', [])
                elif isinstance(configurations, list):
                    nodes = configurations
                else:
                    nodes = []

                for node in nodes:
                    for cpem in node.get('cpeMatch', []):
                        criteria = cpem.get('criteria')
                        if not criteria:
                            continue
                        if not cpem.get('vulnerable', False):
                            continue
                        vendor, product, criteria_version = extract_vendor_product_version(criteria)
                        if not vendor or not product:
                            continue
                        start_incl = cpem.get('versionStartIncluding')
                        start_excl = cpem.get('versionStartExcluding')
                        end_incl = cpem.get('versionEndIncluding')
                        end_excl = cpem.get('versionEndExcluding')

                        if any([start_incl, start_excl, end_incl, end_excl]):
                            key = (vendor, product)
                            cpe_list = cpe_map.get(key, [])
                            for cpe_name, ver_tuple in cpe_list:
                                if version_in_range(ver_tuple,
                                                   parse_version(start_incl) if start_incl else None,
                                                   parse_version(start_excl) if start_excl else None,
                                                   parse_version(end_incl) if end_incl else None,
                                                   parse_version(end_excl) if end_excl else None):
                                    cur.execute("""
                                        INSERT INTO cpe_cve (cpe_name, cve_id) VALUES (%s, %s) ON CONFLICT DO NOTHING
                                    """, (cpe_name, cve_id))
                                    file_relations += 1
                        else:
                            if criteria_version == '*' or criteria_version is None:
                                key = (vendor, product)
                                cpe_list = cpe_map.get(key, [])
                                for cpe_name, _ in cpe_list:
                                    cur.execute("""
                                        INSERT INTO cpe_cve (cpe_name, cve_id) VALUES (%s, %s) ON CONFLICT DO NOTHING
                                    """, (cpe_name, cve_id))
                                    file_relations += 1
                            else:
                                if criteria in processed_criteria:
                                    cpe_names = processed_criteria[criteria]
                                else:
                                    cur.execute("SELECT cpe_name FROM cpe WHERE cpe_name = %s", (criteria,))
                                    cpe_names = [row[0] for row in cur.fetchall()]
                                    processed_criteria[criteria] = cpe_names
                                for cpe_name in cpe_names:
                                    cur.execute("""
                                        INSERT INTO cpe_cve (cpe_name, cve_id) VALUES (%s, %s) ON CONFLICT DO NOTHING
                                    """, (cpe_name, cve_id))
                                    file_relations += 1
            conn.commit()
            total_relations += file_relations
            log(f"Committed {file_relations} relations from {file_path.name} (total {total_relations})")

    elapsed = time.time() - start_time
    log(f"Total CPE-CVE relations inserted: {total_relations} in {elapsed:.2f} seconds")
    conn.close()

if __name__ == '__main__':
    main()