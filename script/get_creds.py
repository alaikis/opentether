import sqlite3

conn = sqlite3.connect(r'E:\workspace\com.alaikis.wisehoof\data\opentether.db')
cursor = conn.cursor()

# Get password for data source
cursor.execute('SELECT password FROM data_sources WHERE id = ?', ('1cce62ad-7c32-4179-8b2b-581f0edd1e87',))
r = cursor.fetchone()
if r:
    print(f'Password: {r[0]}')

# Get skill configs in detail
cursor.execute('SELECT name, config FROM skills WHERE enabled = 1')
for r in cursor.fetchall():
    print(f'\n=== {r[0]} ===')
    print(r[1])

conn.close()
