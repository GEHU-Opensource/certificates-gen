import csv
import json

INPUT_CSV = "certificates.csv"
OUTPUT_JSON = "payload.json"

students = []

with open(INPUT_CSV, newline="", encoding="utf-8") as f:
    reader = csv.reader(f)
    for row in reader:
        student = {
            "name": row[2],
            "email": row[4],
            "course": row[3],
            "event": "Hack The Winter, 2026",
            "club": "WeCode",
            "date": "2026-01-22/23",
            "student_id": row[0]
        }
        students.append(student)


with open(OUTPUT_JSON, "w", encoding="utf-8") as f:
    json.dump(students, f, indent=2)

print("JSON generated:", OUTPUT_JSON)