import sqlite3
import json

conn = sqlite3.connect(r'E:\workspace\com.alaikis.wisehoof\data\opentether.db')
cursor = conn.cursor()

output = []
output.append("=== Enabled Skills ===")
cursor.execute('SELECT id, name, skill_type, category, description, keywords FROM skills WHERE enabled = 1')
rows = cursor.fetchall()
output.append(f'Total: {len(rows)}')
for r in rows:
    name = r[1]
    stype = r[2]
    cat = r[3]
    desc = r[4] or ''
    kw = r[5] or ''
    output.append(f'\nName: {name}')
    output.append(f'Type: {stype}')
    output.append(f'Category: {cat}')
    output.append(f'Desc: {desc}')
    output.append(f'Keywords: {kw}')

output.append("\n=== Data Sources ===")
cursor.execute('SELECT id, name, source_type, host, port, database, enabled FROM data_sources WHERE enabled = 1')
ds_rows = cursor.fetchall()
output.append(f'Total: {len(ds_rows)}')
for r in ds_rows:
    output.append(f'ID={r[0]}, Name={r[1]}, Type={r[2]}, Host={r[3]}, Port={r[4]}, DB={r[5]}, Enabled={r[6]}')

conn.close()

with open(r'E:\workspace\com.alaikis.wisehoof\script\skills_analysis.txt', 'w', encoding='utf-8') as f:
    f.write('\n'.join(output))

print("Analysis written to skills_analysis.txt")
