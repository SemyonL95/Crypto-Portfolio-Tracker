#!/bin/bash

# Migration helper script for SQLite database
# Usage: ./scripts/migrate.sh [command] [options]

set -e

# Default values
DB_PATH="${DB_PATH:-./data/portfolio.db}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-migrations}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Ensure data directory exists
ensure_data_dir() {
    local db_dir=$(dirname "$DB_PATH")
    if [ ! -d "$db_dir" ]; then
        mkdir -p "$db_dir"
        echo -e "${GREEN}Created directory: $db_dir${NC}"
    fi
}

# Check if sqlite3 is available
check_sqlite3() {
    if ! command -v sqlite3 &> /dev/null; then
        echo -e "${RED}Error: sqlite3 is not installed${NC}" >&2
        echo "Please install sqlite3 to run migrations" >&2
        exit 1
    fi
}

# Run up migrations
migrate_up() {
    ensure_data_dir
    check_sqlite3
    
    cd "$PROJECT_ROOT"
    
    echo -e "${YELLOW}Running migrations...${NC}"
    
    local count=0
    for file in $(ls -1 "$MIGRATIONS_DIR"/*.up.sql 2>/dev/null | sort); do
        if [ -f "$file" ]; then
            echo -e "${GREEN}Applying $(basename "$file")...${NC}"
            sqlite3 "$DB_PATH" < "$file" || {
                echo -e "${RED}Error applying migration: $(basename "$file")${NC}" >&2
                exit 1
            }
            ((count++))
        fi
    done
    
    if [ $count -eq 0 ]; then
        echo -e "${YELLOW}No migrations found${NC}"
    else
        echo -e "${GREEN}Migrations completed successfully ($count migration(s) applied)${NC}"
    fi
}

# Run down migrations
migrate_down() {
    ensure_data_dir
    check_sqlite3
    
    cd "$PROJECT_ROOT"
    
    echo -e "${YELLOW}Rolling back migrations...${NC}"
    
    local count=0
    for file in $(ls -1 "$MIGRATIONS_DIR"/*.down.sql 2>/dev/null | sort -r); do
        if [ -f "$file" ]; then
            echo -e "${GREEN}Rolling back $(basename "$file")...${NC}"
            sqlite3 "$DB_PATH" < "$file" || {
                echo -e "${RED}Error rolling back migration: $(basename "$file")${NC}" >&2
                exit 1
            }
            ((count++))
        fi
    done
    
    if [ $count -eq 0 ]; then
        echo -e "${YELLOW}No migrations to rollback${NC}"
    else
        echo -e "${GREEN}Rollback completed successfully ($count migration(s) rolled back)${NC}"
    fi
}

# Reset database
migrate_reset() {
    ensure_data_dir
    check_sqlite3
    
    cd "$PROJECT_ROOT"
    
    echo -e "${YELLOW}Resetting database...${NC}"
    
    if [ -f "$DB_PATH" ]; then
        rm -f "$DB_PATH"
        echo -e "${GREEN}Database file removed${NC}"
    fi
    
    migrate_up
    echo -e "${GREEN}Database reset completed${NC}"
}

# Show migration status
migrate_status() {
    ensure_data_dir
    
    cd "$PROJECT_ROOT"
    
    echo -e "${YELLOW}Migration status:${NC}"
    echo "Database path: $DB_PATH"
    echo "Migrations directory: $MIGRATIONS_DIR"
    echo ""
    
    if [ -f "$DB_PATH" ]; then
        echo -e "${GREEN}Database exists${NC}"
        echo ""
        echo "Tables in database:"
        check_sqlite3
        sqlite3 "$DB_PATH" "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;" 2>/dev/null || echo -e "${RED}Database is empty or not accessible${NC}"
        
        echo ""
        echo "Migration files:"
        local up_count=$(ls -1 "$MIGRATIONS_DIR"/*.up.sql 2>/dev/null | wc -l | tr -d ' ')
        local down_count=$(ls -1 "$MIGRATIONS_DIR"/*.down.sql 2>/dev/null | wc -l | tr -d ' ')
        echo "  Up migrations: $up_count"
        echo "  Down migrations: $down_count"
    else
        echo -e "${YELLOW}Database does not exist${NC}"
    fi
}

# Run migrations in Docker
migrate_up_docker() {
    echo -e "${YELLOW}Running migrations in Docker container...${NC}"
    
    docker-compose exec -T app sh -c "
        for file in \$(ls -1 /root/migrations/*.up.sql 2>/dev/null | sort); do
            if [ -f \"\$file\" ]; then
                echo \"Applying \$(basename \$file)...\"
                sqlite3 /data/portfolio.db < \"\$file\" || exit 1
            fi
        done
    " || {
        echo -e "${RED}Error running migrations in Docker${NC}" >&2
        exit 1
    }
    
    echo -e "${GREEN}Migrations completed successfully${NC}"
}

# Run down migrations in Docker
migrate_down_docker() {
    echo -e "${YELLOW}Rolling back migrations in Docker container...${NC}"
    
    docker-compose exec -T app sh -c "
        for file in \$(ls -1 /root/migrations/*.down.sql 2>/dev/null | sort -r); do
            if [ -f \"\$file\" ]; then
                echo \"Rolling back \$(basename \$file)...\"
                sqlite3 /data/portfolio.db < \"\$file\" || exit 1
            fi
        done
    " || {
        echo -e "${RED}Error rolling back migrations in Docker${NC}" >&2
        exit 1
    }
    
    echo -e "${GREEN}Rollback completed successfully${NC}"
}

# Show usage
show_usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  up              Run all up migrations"
    echo "  down            Run all down migrations (rollback)"
    echo "  reset           Drop database and reapply all migrations"
    echo "  status          Show migration status"
    echo ""
    echo "Environment variables:"
    echo "  DB_PATH         Database file path (default: ./data/portfolio.db)"
    echo "  MIGRATIONS_DIR  Migrations directory (default: migrations)"
    echo ""
    echo "Examples:"
    echo "  $0 up"
    echo "  DB_PATH=/custom/path.db $0 up"
    echo "  $0 status"
}

# Main command handler
main() {
    case "${1:-}" in
        up)
            migrate_up
            ;;
        down)
            migrate_down
            ;;
        reset)
            migrate_reset
            ;;
        status)
            migrate_status
            ;;
        *)
            show_usage
            exit 1
            ;;
    esac
}

main "$@"

