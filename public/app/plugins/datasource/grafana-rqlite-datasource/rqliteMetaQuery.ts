export function showTables(): string {
  return "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name";
}

export function getSchema(table: string): string {
  // Quote the table name to prevent SQL injection and handle special characters
  return `PRAGMA table_info("${table.replace(/"/g, '""')}")`;
}

export function getVersion(): string {
  return 'SELECT sqlite_version() AS version';
}
