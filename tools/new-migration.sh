#!/usr/bin/env bash
# Scaffolds a new internal/db migration file with a timestamp-based name
# and prints the registry-slice line to add to internal/db/migrations
# registration point by hand.
set -euo pipefail

if [[ $# -ne 1 ]]; then
	echo "usage: $(basename "$0") <slug>" >&2
	echo "  slug: snake_case description, e.g. create_notifications" >&2
	exit 1
fi

slug="$1"

if [[ ! "$slug" =~ ^[a-z][a-z0-9_]*$ ]]; then
	echo "error: slug must be snake_case, e.g. create_notifications" >&2
	exit 1
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
migrations_dir="$repo_root/internal/db/migrations"

timestamp="$(date -u +%Y%m%d%H%M%S%3N)"
name="${timestamp}_${slug}"
file="$migrations_dir/${name}.go"

# PascalCase the slug, e.g. create_notifications -> CreateNotifications.
pascal_slug="$(echo "$slug" | awk -F_ '{for (i=1;i<=NF;i++) printf "%s%s", toupper(substr($i,1,1)), substr($i,2)}')"
struct_name="m${timestamp}${pascal_slug}"

mkdir -p "$migrations_dir"

if [[ -e "$file" ]]; then
	echo "error: $file already exists" >&2
	exit 1
fi

cat > "$file" <<EOF
package migrations

import (
	"database/sql"
	"ntx/internal/db"
)

var _ db.Migration = ${struct_name}{}

type ${struct_name} struct{}

func (${struct_name}) Name() string {
	return "${name}"
}

func (${struct_name}) Up(tx *sql.Tx) error {
	_, err := tx.Exec(\`
		-- TODO: forward schema change
	\`)

	return err
}

func (${struct_name}) Down(tx *sql.Tx) error {
	_, err := tx.Exec(\`
		-- TODO: reverse schema change
	\`)

	return err
}
EOF

echo "created $file"
echo
echo "add this line to the registry slice in internal/db/migrations/migrations.go:"
echo "	${struct_name}{},"
