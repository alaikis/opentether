import sqlite3
import json

conn = sqlite3.connect(r'E:\workspace\com.alaikis.wisehoof\data\opentether.db')
cursor = conn.cursor()

output = []

# 1. Skills
output.append("=== Enabled Skills ===")
cursor.execute('SELECT id, name, skill_type, category, description, keywords FROM skills WHERE enabled = 1')
rows = cursor.fetchall()
output.append(f'Total: {len(rows)}')
for r in rows:
    output.append(f'\nName: {r[1]}')
    output.append(f'Type: {r[2]}')
    output.append(f'Category: {r[3]}')
    output.append(f'Desc: {r[4]}')
    output.append(f'Keywords: {r[5]}')

# 2. Data Sources
output.append("\n=== Data Sources ===")
cursor.execute('SELECT id, name, source_type, host, port, database, enabled FROM data_sources WHERE enabled = 1')
ds_rows = cursor.fetchall()
output.append(f'Total: {len(ds_rows)}')
for r in ds_rows:
    output.append(f'ID={r[0]}, Name={r[1]}, Type={r[2]}, Host={r[3]}, Port={r[4]}, DB={r[5]}, Enabled={r[6]}')

# 3. Users
output.append("\n=== Users ===")
cursor.execute('SELECT id, name, email, role, status, global_user_id, external_employee_id FROM users LIMIT 20')
user_rows = cursor.fetchall()
output.append(f'Total (sample): {len(user_rows)}')
for r in user_rows:
    output.append(f'ID={r[0]}, Name={r[1]}, Email={r[2]}, Role={r[3]}, Status={r[4]}, GlobalUserID={r[5]}, ExternalEmployeeID={r[6]}')

# 4. User Groups
output.append("\n=== User Groups ===")
cursor.execute('SELECT id, group_name, group_code, description, data_access_scope, parent_group_id FROM user_groups')
group_rows = cursor.fetchall()
output.append(f'Total: {len(group_rows)}')
for r in group_rows:
    output.append(f'ID={r[0]}, Name={r[1]}, Code={r[2]}, Desc={r[3]}, Scope={r[4]}, Parent={r[5]}')

# 5. Group Members
output.append("\n=== Group Members ===")
cursor.execute('SELECT user_group_id, user_id FROM user_group_members')
gm_rows = cursor.fetchall()
output.append(f'Total: {len(gm_rows)}')
for r in gm_rows:
    output.append(f'GroupID={r[0]}, UserID={r[1]}')

output.append("\n=== Skill Configs ===")
cursor.execute('SELECT id, name, skill_type, category, data_scope, allowed_groups, config FROM skills WHERE enabled = 1')
skill_rows = cursor.fetchall()
for r in skill_rows:
    output.append(f'\nName: {r[1]}')
    output.append(f'Type: {r[2]}, Category: {r[3]}, Scope: {r[4]}')
    output.append(f'AllowedGroups: {r[5]}')
    config = r[6]
    if config:
        output.append(f'Config: {config[:300]}')

conn.close()

with open(r'E:\workspace\com.alaikis.wisehoof\script\env_analysis.txt', 'w', encoding='utf-8') as f:
    f.write('\n'.join(output))

print("Analysis written to env_analysis.txt")
