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
        for file_path, data in iter_json_files(cve_folder):
            for vuln in data.get('vulnerabilities', []):
                cve = vuln.get('cve', {})
                cve_id = cve.get('id')
                if not cve_id:
                    continue

                desc = None
                for d in cve.get('descriptions', []):
                    if d.get('lang') == 'en':
                        desc = d.get('value')
                        break

                severity = None
                metrics = cve.get('metrics', {})
                if 'cvssMetricV31' in metrics and metrics['cvssMetricV31']:
                    severity = metrics['cvssMetricV31'][0].get('cvssData', {}).get('baseSeverity')
                elif 'cvssMetricV30' in metrics and metrics['cvssMetricV30']:
                    severity = metrics['cvssMetricV30'][0].get('cvssData', {}).get('baseSeverity')
                elif 'cvssMetricV2' in metrics and metrics['cvssMetricV2']:
                    severity = metrics['cvssMetricV2'][0].get('baseSeverity')
                elif 'cvssMetricV40' in metrics and metrics['cvssMetricV40']:
                    severity = metrics['cvssMetricV40'][0].get('cvssData', {}).get('baseSeverity')

                refs = []
                for ref in cve.get('references', []):
                    if 'Broken Link' not in ref.get('tags', []):
                        url = ref.get('url')
                        if url:
                            refs.append(url)
                refs_str = '\n'.join(refs) if refs else None

                cur.execute("""
                    INSERT INTO cves (id, description, severity, references)
                    VALUES (%s, %s, %s, %s)
                    ON CONFLICT (id) DO UPDATE SET
                        description = EXCLUDED.description,
                        severity = EXCLUDED.severity,
                        references = EXCLUDED.references
                """, (cve_id, desc, severity, refs_str))
        conn.commit()
    conn.close()

if __name__ == '__main__':
    main()