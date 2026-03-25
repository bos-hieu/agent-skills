---
name: database
description: Interact with databases using persistent config files or environment variables. Supports multiple named databases (auto-detected from env, project config, and global config), PostgreSQL, MySQL, SQLite, and MongoDB. Manage database connections with --add-db, --edit-db, --remove-db. Select which database to use with --db.
allowed-tools: Bash(go run *), Bash(cat *), Bash(ls *), Bash(printenv *)
---

When the user asks to query a database, explore tables/collections, or run SQL/MongoDB queries:

1. Auto-detect available databases from config files and environment variables.
2. Let the user pick which database with `--db <name>`.
3. Run queries, list tables/collections, describe schemas, or export results.
4. Never print raw passwords — mask credentials in output.
5. Use `${CLAUDE_SKILL_DIR}` to reference the Go file.
6. **Never save database credentials to memory files or auto-memory.**

## Database Configuration

Databases are discovered from three sources (highest priority first):

1. **Project config** (`.claude/db-config.yaml`) — project-specific databases
2. **Global config** (`~/.claude/db-config.yaml`) — shared across all projects
3. **Environment variables** — traditional env-based configuration

When the same database name exists in multiple sources, the higher-priority source wins.

### Managing Databases via CLI

```bash
# Add a database to project config (default)
go run ${CLAUDE_SKILL_DIR}/db_query.go --add-db prod --dsn "postgres://user:pass@host:5432/dbname"

# Add with individual fields
go run ${CLAUDE_SKILL_DIR}/db_query.go --add-db staging \
  --host staging.example.com --port 5432 \
  --user admin --password secret --dbname myapp

# Add a MongoDB database
go run ${CLAUDE_SKILL_DIR}/db_query.go --add-db mongo-prod \
  --dsn "mongodb://user:pass@host:27017/mydb"

# Add MongoDB with individual fields
go run ${CLAUDE_SKILL_DIR}/db_query.go --add-db mongo-dev \
  --driver mongodb --host localhost --port 27017 \
  --user admin --password secret --dbname myapp

# Add to global config (shared across projects)
go run ${CLAUDE_SKILL_DIR}/db_query.go --add-db shared-db --global \
  --dsn "postgres://user:pass@shared-host:5432/db"

# Edit an existing database (only updates provided fields)
go run ${CLAUDE_SKILL_DIR}/db_query.go --edit-db prod --password newpassword
go run ${CLAUDE_SKILL_DIR}/db_query.go --edit-db prod --global --host new-host.example.com

# Remove a database
go run ${CLAUDE_SKILL_DIR}/db_query.go --remove-db staging
go run ${CLAUDE_SKILL_DIR}/db_query.go --remove-db shared-db --global

# List all databases (shows source: env/global/project)
go run ${CLAUDE_SKILL_DIR}/db_query.go --list
```

### Config File Format

Both config files use the same YAML format:

```yaml
# .claude/db-config.yaml (project) or ~/.claude/db-config.yaml (global)
databases:
  prod:
    dsn: "postgres://user:pass@prod-host:5432/mydb?sslmode=require"
  staging:
    driver: postgres
    host: staging.example.com
    port: "5432"
    user: admin
    password: secret
    dbname: myapp
    ssl_mode: disable
  mysql-db:
    driver: mysql
    host: localhost
    port: "3306"
    user: root
    password: root
    dbname: testdb
  mongo-db:
    driver: mongodb
    dsn: "mongodb://admin:secret@mongo-host:27017/mydb?authSource=admin"
  mongo-local:
    driver: mongodb
    host: localhost
    port: "27017"
    user: admin
    password: secret
    dbname: myapp
```

Config files can be edited manually or via the CLI commands above. The config files are stored with `0600` permissions (readable only by the owner).

### Environment Variables (Legacy)

Still fully supported. Databases from env vars have the lowest priority.

#### Pattern A — Component-based (matches this project's config.yaml style)
```
<PREFIX>_DB_HOST=localhost
<PREFIX>_DB_PORT=5432
<PREFIX>_DB_USER=postgres
<PREFIX>_DB_PASSWORD=secret
<PREFIX>_DB_NAME=mydb
<PREFIX>_DB_SSL_MODE=disable       # optional, default: disable
<PREFIX>_DB_DRIVER=postgres        # optional, default: auto-detect from port
```

#### Pattern B — DSN-based
```
<NAME>_DSN=postgres://user:pass@host:5432/dbname?sslmode=disable
DATABASE_URL=postgres://...              # -> name "default"
DATABASE_URL_PROD=postgres://...         # -> name "prod"
MONGODB_URI=mongodb://...               # -> name "mongodb"
<NAME>_DSN=mongodb://...                # -> name "<name>"
```

## Query Flags

| Flag | Description | Example |
|---|---|---|
| `--list` | List all detected databases (masks passwords) | `--list` |
| `--db <name>` | Select which database to use | `--db alert` |
| `--query <sql/json>` | Run a SQL query or MongoDB JSON filter | `--query "SELECT count(*) FROM users"` |
| `--tables` | List all tables/collections | `--tables` |
| `--describe <table>` | Show columns/fields, types, and nullability | `--describe users` |
| `--collection <name>` | MongoDB collection name (required for `--query` with MongoDB) | `--collection users` |
| `--rows <n>` | Max result rows to display (default 20) | `--rows 50` |
| `--format <table\|csv\|json>` | Output format (default: table) | `--format csv` |
| `--no-header` | Suppress column headers in output | `--no-header` |

## Config Management Flags

| Flag | Description | Example |
|---|---|---|
| `--add-db <name>` | Add a database to config file | `--add-db prod --dsn "..."` |
| `--edit-db <name>` | Edit a database in config file | `--edit-db prod --password new` |
| `--remove-db <name>` | Remove a database from config file | `--remove-db staging` |
| `--global` | Target global config instead of project | `--add-db x --global --dsn "..."` |
| `--dsn <dsn>` | Connection string (for add/edit) | `--dsn "postgres://..."` |
| `--host <host>` | Database host (for add/edit) | `--host localhost` |
| `--port <port>` | Database port (for add/edit) | `--port 5432` |
| `--user <user>` | Database user (for add/edit) | `--user postgres` |
| `--password <pass>` | Database password (for add/edit) | `--password secret` |
| `--dbname <name>` | Database name (for add/edit) | `--dbname mydb` |
| `--driver <driver>` | Driver: postgres, mysql, mongodb (for add/edit) | `--driver mongodb` |
| `--ssl-mode <mode>` | SSL mode (for add/edit) | `--ssl-mode require` |

## Examples

### SQL Databases (PostgreSQL, MySQL)

```bash
# List all databases
go run ${CLAUDE_SKILL_DIR}/db_query.go --list

# List tables
go run ${CLAUDE_SKILL_DIR}/db_query.go --db main --tables

# Describe a table
go run ${CLAUDE_SKILL_DIR}/db_query.go --db main --describe users

# Run a query
go run ${CLAUDE_SKILL_DIR}/db_query.go --db main --query "SELECT id, email FROM users LIMIT 5"

# Export to CSV
go run ${CLAUDE_SKILL_DIR}/db_query.go --db main --query "SELECT * FROM orders" --format csv --rows 1000

# JSON output
go run ${CLAUDE_SKILL_DIR}/db_query.go --db main --query "SELECT * FROM wallets WHERE balance > 0" --format json
```

### MongoDB

```bash
# List all collections
go run ${CLAUDE_SKILL_DIR}/db_query.go --db mongo-dev --tables

# Describe a collection (samples documents to infer schema)
go run ${CLAUDE_SKILL_DIR}/db_query.go --db mongo-dev --describe users

# Query with JSON filter
go run ${CLAUDE_SKILL_DIR}/db_query.go --db mongo-dev --collection users --query '{"age": {"$gt": 25}}'

# Get all documents from a collection
go run ${CLAUDE_SKILL_DIR}/db_query.go --db mongo-dev --collection orders --query '{}'

# Export to JSON
go run ${CLAUDE_SKILL_DIR}/db_query.go --db mongo-dev --collection users --query '{}' --format json --rows 100

# Export to CSV
go run ${CLAUDE_SKILL_DIR}/db_query.go --db mongo-dev --collection logs --query '{"level": "error"}' --format csv
```

## Supported Drivers

| Driver | DSN prefix / port heuristic |
|---|---|
| PostgreSQL | `postgres://`, `postgresql://`, port 5432 |
| MySQL | `mysql://`, port 3306 |
| SQLite | `sqlite://`, `file:`, path ending in `.db`/`.sqlite` |
| MongoDB | `mongodb://`, `mongodb+srv://`, port 27017 |

## Security Notes

- Passwords are **never printed** — always shown as `***`.
- Config files are created with `0600` permissions (owner-only read/write).
- **Never save database credentials to auto-memory or memory files.**
- Always prefer read-only DB users for this skill.
- Do not commit config files with real credentials — `.claude/db-config.yaml` should be gitignored.
