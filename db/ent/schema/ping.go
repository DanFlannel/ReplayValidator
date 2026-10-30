// Package schema holds ent entity schemas. No real domain model exists yet —
// this repo is bootstrap scaffolding.
package schema

import (
	"entgo.io/ent"
)

// Ping is a placeholder entity: ent requires at least one schema to generate
// a client, and this repo has no domain model yet. It exists only to prove
// the generate → migrate → ready-check pipeline works end to end. Delete it
// once a real entity exists.
type Ping struct {
	ent.Schema
}
