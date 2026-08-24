// Package migrations holds the ordered, manually registered list of
// internal/db.Migration implementations applied to the ntx-server
// database. New migrations are scaffolded with tools/new-migration.sh and
// must be added to All by hand, in execution order.
package migrations

import "ntx/internal/db"

// All is the ordered list of migrations applied by internal/db.DB.
var All = []db.Migration{
	m20260824083247487CreateNotifications{},
}
