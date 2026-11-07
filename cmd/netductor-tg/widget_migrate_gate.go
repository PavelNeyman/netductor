package main

import "github.com/PavelNeyman/netductor/internal/migrate"

func migrateNeedsAttention() bool {
	return migrate.NeedsOperatorAttention()
}
