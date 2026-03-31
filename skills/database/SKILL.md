---
name: database
description: Interact with databases using persistent config files or environment variables. Supports multiple named databases (auto-detected from env, project config, and global config), PostgreSQL, MySQL, SQLite, and MongoDB. Manage database connections with --add-db, --edit-db, --remove-db. Select which database to use with --db.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *), Bash(printenv *)
---

When the user asks to query a database, use `${CLAUDE_SKILL_DIR}` to reference the Go file.
**Never print raw passwords. Never save credentials to memory files or auto-memory.**

## Configuration

Databases discovered from (highest priority first):
1. Project config: `.claude/db-config.yaml`
2. Global config: `~/.claude/db-config.yaml`
3. Environment variables (see patterns below)

```yaml
# .claude/db-config.yaml
databases:
  prod:
    dsn: "postgres://user:pass@host:5432/mydb?sslmode=require"
  staging:
    driver: postgres
    host: staging.example.com
    port: "5432"
    user: admin
    password: secret
    dbname: myapp
  mongo-db:
    driver: mongodb
    dsn: "mongodb://admin:secret@host:27017/mydb?authSource=admin"
```

### Environment Variable Patterns

- Component-based: `<PREFIX>_DB_HOST`, `<PREFIX>_DB_PORT`, `<PREFIX>_DB_USER`, `<PREFIX>_DB_PASSWORD`, `<PREFIX>_DB_NAME`, `<PREFIX>_DB_DRIVER`, `<PREFIX>_DB_SSL_MODE`
- DSN-based: `<NAME>_DSN=postgres://...`, `DATABASE_URL=postgres://...`, `MONGODB_URI=mongodb://...`

## Config Management

```bash
go run ${CLAUDE_SKILL_DIR}/db_query.go [flag]
```

| Flag | Description |
|---|---|
| `--add-db <name>` | Add database (with `--dsn` or `--host --port --user --password --dbname`) |
| `--edit-db <name>` | Edit database (only updates provided fields) |
| `--remove-db <name>` | Remove database |
| `--global` | Target global config instead of project |
| `--list` | List all detected databases (masks passwords) |

Connection flags for add/edit: `--dsn`, `--host`, `--port`, `--user`, `--password`, `--dbname`, `--driver` (postgres/mysql/mongodb), `--ssl-mode`.

## Query Flags

| Flag | Description |
|---|---|
| `--db <name>` | Select database |
| `--query <sql/json>` | Run SQL query or MongoDB JSON filter |
| `--tables` | List all tables/collections |
| `--describe <table>` | Show columns/fields, types, nullability |
| `--rows <n>` | Max result rows (default 20) |
| `--format <table\|csv\|json>` | Output format (default: table) |
| `--no-header` | Suppress column headers |

### MongoDB-Specific Flags

| Flag | Description |
|---|---|
| `--collection <name>` | Collection name (required for `--query`) |
| `--sort <json>` | Sort specification |
| `--project <json>` | Field projection |
| `--skip <n>` | Documents to skip |
| `--aggregate <json>` | Aggregation pipeline as JSON array |
| `--count` | Count matching documents |
| `--distinct <field>` | Get distinct values |

## Supported Drivers

| Driver | DSN prefix / port heuristic |
|---|---|
| PostgreSQL | `postgres://`, `postgresql://`, port 5432 |
| MySQL | `mysql://`, port 3306 |
| SQLite | `sqlite://`, `file:`, `.db`/`.sqlite` |
| MongoDB | `mongodb://`, `mongodb+srv://`, port 27017 |

## Security

- Passwords always shown as `***`. Config files created with `0600` permissions.
- Prefer read-only DB users. Gitignore `.claude/db-config.yaml`.
