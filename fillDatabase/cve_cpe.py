import json
import logging
import os
import sys
import psycopg
from pathlib import Path

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

CVE_FOLDER = Path(os.environ.get("CVE_FOLDER", "/app/cve"))
BATCH_SIZE = 10000

def parse_cpe(criteria: str):
    parts = criteria.split(':')
    vendor = parts[3] if len(parts) > 3 and parts[3] != '*' else None
    product = parts[4] if len(parts) > 4 and parts[4] != '*' else None
    version = parts[5] if len(parts) > 5 and parts[5] not in ('*', '-') else None
    return vendor, product, version

def collect_cpe_matches(node):
    """Рекурсивно собирает все criteria из cpeMatch с vulnerable=True."""
    result = []
    for match in node.get("cpeMatch", []):
        if match.get("vulnerable") in (True, "true"):
            criteria = match.get("criteria")
            if criteria:
                result.append(criteria)
    for child in node.get("children", []):
        result.extend(collect_cpe_matches(child))
    return result

def main():
    conn = psycopg.connect(
        host=os.environ.get("POSTGRES_HOST", "localhost"),
        dbname=os.environ.get("POSTGRES_DB", "cve_db"),
        user=os.environ.get("POSTGRES_USER", "postgres"),
        password=os.environ.get("POSTGRES_PASSWORD", "postgres"),
    )
    conn.autocommit = False

    files = list(CVE_FOLDER.rglob("*.json"))
    if not files:
        logging.error("No JSON files found in %s", CVE_FOLDER)
        sys.exit(1)
    logging.info("Found %d JSON files", len(files))

    cpe_buffer = []
    cpe_cve_buffer = []
    inserted_cpe = 0
    inserted_links = 0

    try:
        with conn.cursor() as cur:
            for file in files:
                logging.info("Processing %s", file.name)
                with open(file, "r", encoding="utf-8") as f:
                    data = json.load(f)
                for item in data.get("vulnerabilities", []):
                    cve = item.get("cve", {})
                    cve_id = cve.get("id")
                    if not cve_id:
                        continue
                    for config in cve.get("configurations", []):
                        for node in config.get("nodes", []):
                            for criteria in collect_cpe_matches(node):
                                vendor, product, ver = parse_cpe(criteria)
                                cpe_buffer.append((criteria, vendor, product, ver))
                                cpe_cve_buffer.append((criteria, cve_id))

                                if len(cpe_buffer) >= BATCH_SIZE:
                                    cur.executemany(
                                        "INSERT INTO cpe (cpe_name, vendor, product, ver) VALUES (%s, %s, %s, %s) ON CONFLICT (cpe_name) DO NOTHING",
                                        cpe_buffer,
                                    )
                                    conn.commit()
                                    inserted_cpe += len(cpe_buffer)
                                    cpe_buffer.clear()

                                    cur.executemany(
                                        "INSERT INTO cpe_cve (cpe_name, cve_id) VALUES (%s, %s) ON CONFLICT (cpe_name, cve_id) DO NOTHING",
                                        cpe_cve_buffer,
                                    )
                                    conn.commit()
                                    inserted_links += len(cpe_cve_buffer)
                                    cpe_cve_buffer.clear()
                                    logging.info("Inserted batch: CPEs=%d, links=%d", inserted_cpe, inserted_links)

            if cpe_buffer:
                cur.executemany(
                    "INSERT INTO cpe (cpe_name, vendor, product, ver) VALUES (%s, %s, %s, %s) ON CONFLICT (cpe_name) DO NOTHING",
                    cpe_buffer,
                )
                conn.commit()
                inserted_cpe += len(cpe_buffer)
                cpe_buffer.clear()
            if cpe_cve_buffer:
                cur.executemany(
                    "INSERT INTO cpe_cve (cpe_name, cve_id) VALUES (%s, %s) ON CONFLICT (cpe_name, cve_id) DO NOTHING",
                    cpe_cve_buffer,
                )
                conn.commit()
                inserted_links += len(cpe_cve_buffer)
                cpe_cve_buffer.clear()

            cur.execute("SELECT COUNT(*) FROM cpe")
            cpe_count = cur.fetchone()[0]
            cur.execute("SELECT COUNT(*) FROM cpe_cve")
            link_count = cur.fetchone()[0]
            logging.info("Final counts: cpe=%d, cpe_cve=%d", cpe_count, link_count)

    except Exception as e:
        logging.error("Error: %s", e)
        conn.rollback()
        sys.exit(1)
    finally:
        conn.close()

if __name__ == "__main__":
    main()