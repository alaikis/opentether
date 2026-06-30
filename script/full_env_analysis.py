import sqlite3
import json

conn = sqlite3.connect(r'E:\workspace\com.alaikis.wisehoof\data\opentether.db')
cursor = conn.cursor()

# Get data source details
cursor.execute('SELECT id, name, source_type, host, port, database, user, password FROM data_sources WHERE enabled = 1')
ds = cursor.fetchone()
if ds:
    print(f'Data Source: {ds[1]}')
    print(f'Host: {ds[2]}://{ds[3]}:{ds[4]}')
    print(f'Database: {ds[5]}')
    print(f'User: {ds[6]}')
    print(f'Password: {ds[7]}')

# Get all skills with full config
print("\n=== Skills Full Config ===")
cursor.execute('SELECT id, name, skill_type, category, data_scope, allowed_groups, config, keywords FROM skills WHERE enabled = 1')
for r in cursor.fetchall():
    print(f'\n--- {r[1]} ---')
    print(f'ID: {r[0]}')
    print(f'Type: {r[2]}, Category: {r[3]}, Scope: {r[4]}')
    print(f'AllowedGroups: {r[5]}')
    print(f'Keywords: {r[7]}')
    print(f'Config: {r[6]}')

# Get users with groups
print("\n=== Users with Groups ===")
cursor.execute('''
    SELECT u.id, u.name, u.role, u.global_user_id, u.external_employee_id, ug.group_name 
    FROM users u
    LEFT JOIN user_group_members ugm ON u.id = ugm.user_id
    LEFT JOIN user_groups ug ON ugm.user_group_id = ug.id
    WHERE u.status = "active"
''')
for r in cursor.fetchall():
    print(f'User: {r[1]}, Role: {r[2]}, GlobalID: {r[3]}, ExternalID: {r[4]}, Group: {r[5]}')

conn.close()
