#!/bin/bash

# OpenEAAP Database Migration Script
# This script manages database migrations using golang-migrate tool
# Supports PostgreSQL database schema versioning

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default configuration
MIGRATIONS_DIR="${MIGRATIONS_DIR:-./migrations}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-openeeap}"
DB_PASSWORD="${DB_PASSWORD:-openeeap}"
DB_NAME="${DB_NAME:-openeeap}"
DB_SSLMODE="${DB_SSLMODE:-disable}"

# Construct database URL
DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"

# Check if migrate tool is installed
check_migrate_installed() {
   if ! command -v migrate &> /dev/null; then
       echo -e "${RED}Error: golang-migrate is not installed${NC}"
       echo "Install it with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
       exit 1
   fi
}

# Print usage information
usage() {
   cat << EOF
${GREEN}OpenEAAP Database Migration Script${NC}

Usage: $0 [command] [options]

Commands:
   up [N]              Apply all or N up migrations
   down [N]            Apply all or N down migrations
   goto VERSION        Migrate to a specific version
   drop                Drop everything inside database
   force VERSION       Set version but don't run migration (useful for fixing dirty state)
   version             Print current migration version
   create NAME         Create new migration files with given name
   status              Show migration status

Options:
   -h, --help          Show this help message
   -d, --dir DIR       Migrations directory (default: ./migrations)
   --host HOST         Database host (default: localhost)
   --port PORT         Database port (default: 5432)
   --user USER         Database user (default: openeeap)
   --password PASS     Database password
   --database DB       Database name (default: openeeap)
   --sslmode MODE      SSL mode (default: disable)

Environment Variables:
   DB_HOST             Database host
   DB_PORT             Database port
   DB_USER             Database user
   DB_PASSWORD         Database password
   DB_NAME             Database name
   DB_SSLMODE          SSL mode
   MIGRATIONS_DIR      Migrations directory

Examples:
   # Apply all pending migrations
   $0 up

   # Apply next 2 migrations
   $0 up 2

   # Rollback last migration
   $0 down 1

   # Check current version
   $0 version

   # Create new migration
   $0 create add_user_table

   # Migrate to specific version
   $0 goto 3

EOF
}

# Parse command line arguments
parse_args() {
   while [[ $# -gt 0 ]]; do
       case $1 in
           -h|--help)
               usage
               exit 0
               ;;
           -d|--dir)
               MIGRATIONS_DIR="$2"
               shift 2
               ;;
           --host)
               DB_HOST="$2"
               shift 2
               ;;
           --port)
               DB_PORT="$2"
               shift 2
               ;;
           --user)
               DB_USER="$2"
               shift 2
               ;;
           --password)
               DB_PASSWORD="$2"
               shift 2
               ;;
           --database)
               DB_NAME="$2"
               shift 2
               ;;
           --sslmode)
               DB_SSLMODE="$2"
               shift 2
               ;;
           *)
               break
               ;;
       esac
   done

   # Reconstruct DB_URL after parsing args
   DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"
}

# Execute migration command
run_migration() {
   local command=$1
   shift

   echo -e "${YELLOW}Running migration: ${command} $@${NC}"
   echo -e "${YELLOW}Database: ${DB_HOST}:${DB_PORT}/${DB_NAME}${NC}"
   echo -e "${YELLOW}Migrations directory: ${MIGRATIONS_DIR}${NC}"
   echo ""

   case $command in
       up)
           if [ -z "$1" ]; then
               migrate -path "${MIGRATIONS_DIR}" -database "${DB_URL}" up
           else
               migrate -path "${MIGRATIONS_DIR}" -database "${DB_URL}" up "$1"
           fi
           ;;
       down)
           if [ -z "$1" ]; then
               echo -e "${RED}Warning: This will rollback ALL migrations!${NC}"
               read -p "Are you sure? (yes/no): " confirm
               if [ "$confirm" != "yes" ]; then
                   echo "Aborted."
                   exit 0
               fi
               migrate -path "${MIGRATIONS_DIR}" -database "${DB_URL}" down -all
           else
               migrate -path "${MIGRATIONS_DIR}" -database "${DB_URL}" down "$1"
           fi
           ;;
       goto)
           if [ -z "$1" ]; then
               echo -e "${RED}Error: VERSION argument is required${NC}"
               exit 1
           fi
           migrate -path "${MIGRATIONS_DIR}" -database "${DB_URL}" goto "$1"
           ;;
       drop)
           echo -e "${RED}Warning: This will drop everything in the database!${NC}"
           read -p "Are you sure? (yes/no): " confirm
           if [ "$confirm" != "yes" ]; then
               echo "Aborted."
               exit 0
           fi
           migrate -path "${MIGRATIONS_DIR}" -database "${DB_URL}" drop -f
           ;;
       force)
           if [ -z "$1" ]; then
               echo -e "${RED}Error: VERSION argument is required${NC}"
               exit 1
           fi
           migrate -path "${MIGRATIONS_DIR}" -database "${DB_URL}" force "$1"
           ;;
       version)
           migrate -path "${MIGRATIONS_DIR}" -database "${DB_URL}" version
           ;;
       create)
           if [ -z "$1" ]; then
               echo -e "${RED}Error: NAME argument is required${NC}"
               exit 1
           fi
           migrate create -ext sql -dir "${MIGRATIONS_DIR}" -seq "$1"
           echo -e "${GREEN}Created migration files in ${MIGRATIONS_DIR}${NC}"
           ;;
       status)
           echo -e "${GREEN}Current migration status:${NC}"
           migrate -path "${MIGRATIONS_DIR}" -database "${DB_URL}" version
           echo ""
           echo -e "${GREEN}Available migrations:${NC}"
           ls -1 "${MIGRATIONS_DIR}"/*.sql 2>/dev/null | sort || echo "No migrations found"
           ;;
       *)
           echo -e "${RED}Error: Unknown command '${command}'${NC}"
           usage
           exit 1
           ;;
   esac

   if [ $? -eq 0 ]; then
       echo -e "${GREEN}✓ Migration completed successfully${NC}"
   else
       echo -e "${RED}✗ Migration failed${NC}"
       exit 1
   fi
}

# Main execution
main() {
   # Check if migrate is installed
   check_migrate_installed

   # Parse arguments
   parse_args "$@"

   # Get command
   if [ $# -eq 0 ]; then
       usage
       exit 1
   fi

   local command=$1
   shift

   # Create migrations directory if it doesn't exist
   if [ ! -d "${MIGRATIONS_DIR}" ]; then
       echo -e "${YELLOW}Creating migrations directory: ${MIGRATIONS_DIR}${NC}"
       mkdir -p "${MIGRATIONS_DIR}"
   fi

   # Run migration
   run_migration "$command" "$@"
}

# Run main function
main "$@"

# Personal.AI order the ending
