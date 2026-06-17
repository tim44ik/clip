import logging
import os
from pathlib import Path

import psycopg

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(message)s",
)

CVE_FOLDER = Path(os.environ["CVE_FOLDER"])

conn = psycopg.connect(
    host=os.environ["POSTGRES_HOST"],
    dbname=os.environ["POSTGRES_DB"],
    user=os.environ["POSTGRES_USER"],
    password=os.environ["POSTGRES_PASSWORD"],
)

BATCH_SIZE = 5000


def extract_description(cve: dict) -> str:
    for item in cve.get("descriptions", []):
        if item.get("lang") == "en":
            return item.get("value", "")

    return ""


def extract_severity(cve: dict):
    metrics = cve.get("metrics", {})

    if "cvssMetricV40" in metrics and metrics["cvssMetricV40"]:
        return (
            metrics["cvssMetricV40"][0]
            .get("cvssData", {})
            .get("baseSeverity")
        )

    if "cvssMetricV31" in metrics and metrics["cvssMetricV31"]:
        return (
            metrics["cvssMetricV31"][0]
            .get("cvssData", {})
            .get("baseSeverity")
        )

    if "cvssMetricV30" in metrics and metrics["cvssMetricV30"]:
        return (
            metrics["cvssMetricV30"][0]
            .get("cvssData", {})
            .get("baseSeverity")
        )

    if "cvssMetricV2" in metrics and metrics["cvssMetricV2"]:
        return metrics["cvssMetricV2"][0].get("baseSeverity")

    return None


processed = 0
buffer = []

with conn:
    with conn.cursor() as cur:

        for file in sorted(CVE_FOLDER.rglob("*.json")):

            logging.info("Processing %s", file)

            try:
                import json

                with open(file, "r", encoding="utf-8") as f:
                    data = json.load(f)

            except Exception:
                logging.exception("Failed to parse %s", file)
                continue

            for item in data.get("vulnerabilities", []):

                cve = item.get("cve", {})

                cve_id = cve.get("id")

                if not cve_id:
                    continue

                refs = "\n".join(
                    sorted(
                        {
                            ref["url"]
                            for ref in cve.get("references", [])
                            if ref.get("url")
                        }
                    )
                )

                buffer.append(
                    (
                        cve_id,
                        extract_description(cve),
                        extract_severity(cve),
                        refs,
                    )
                )

                processed += 1

                if len(buffer) >= BATCH_SIZE:

                    cur.executemany(
                        """
                        INSERT INTO cve (
                            id,
                            descr,
                            severity,
                            refs
                        )
                        VALUES (%s, %s, %s, %s)
                        ON CONFLICT (id) DO NOTHING
                        """,
                        buffer,
                    )

                    conn.commit()

                    logging.info(
                        "Inserted %d CVE (processed=%d)",
                        len(buffer),
                        processed,
                    )

                    buffer.clear()

        if buffer:

            cur.executemany(
                """
                INSERT INTO cve (
                    id,
                    descr,
                    severity,
                    refs
                )
                VALUES (%s, %s, %s, %s)
                ON CONFLICT (id) DO NOTHING
                """,
                buffer,
            )

            conn.commit()

            logging.info(
                "Inserted final %d CVE",
                len(buffer),
            )

logging.info(
    "Finished CVE import. Total processed=%d",
    processed,
)

conn.close()