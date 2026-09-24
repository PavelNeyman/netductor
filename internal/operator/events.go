package operator

import "fmt"

// Step is a stable progress ID for TUI/Web (not free-form log lines).
type Step struct {
	ID      string // e.g. primary, secondary, credentials
	Message string
	Done    bool
	Err     string
}

// Reporter receives structured steps. nil = no-op.
type Reporter func(Step)

func report(r Reporter, id, msg string) {
	if r != nil {
		r(Step{ID: id, Message: msg})
	}
}

func reportDone(r Reporter, id, msg string) {
	if r != nil {
		r(Step{ID: id, Message: msg, Done: true})
	}
}

func reportErr(r Reporter, id string, err error) {
	if r != nil && err != nil {
		r(Step{ID: id, Message: err.Error(), Err: err.Error()})
	}
}

// FleetDeployWithReport is FleetDeploy plus step callbacks.
func FleetDeployWithReport(f FleetSpec, r Reporter) error {
	if !f.DoPrimary && !f.DoSecondary {
		return fmt.Errorf("fleet: nothing selected (primary and/or secondary)")
	}
	var primaryHost, primaryUser, primaryKey, keyPass string
	if f.DoPrimary {
		report(r, "primary", "deploy primary")
		s := f.Primary
		ApplyDomainFlags(&s)
		if err := DeployPrimary(s); err != nil {
			reportErr(r, "primary", err)
			return fmt.Errorf("fleet primary: %w", err)
		}
		reportDone(r, "primary", "primary ok")
		primaryHost = s.Host
		primaryUser = s.User
		if primaryUser == "" {
			primaryUser = "root"
		}
		primaryKey = s.SSHPrivateKey
		keyPass = s.KeyPassphrase
		if primaryKey == "" {
			primaryKey = defaultOperatorKeyPath()
		}
		primaryKey = expandHome(primaryKey)
	}
	if f.DoSecondary {
		report(r, "secondary", "deploy secondary")
		sec := f.Secondary
		if sec.PrimaryHost == "" {
			sec.PrimaryHost = primaryHost
		}
		if sec.PrimaryUser == "" {
			sec.PrimaryUser = primaryUser
		}
		if sec.PrimaryKey == "" {
			sec.PrimaryKey = primaryKey
		}
		if sec.PrimaryKeyPassphrase == "" {
			sec.PrimaryKeyPassphrase = keyPass
		}
		if sec.SNI == "" && f.Primary.SNI != "" {
			sec.SNI = f.Primary.SNI
		}
		if sec.PrimaryHost == "" || sec.PrimaryKey == "" {
			err := fmt.Errorf("fleet secondary: primary host/key required")
			reportErr(r, "secondary", err)
			return err
		}
		if err := DeploySecondary(sec); err != nil {
			reportErr(r, "secondary", err)
			return fmt.Errorf("fleet secondary: %w", err)
		}
		reportDone(r, "secondary", "secondary ok")
	}
	reportDone(r, "done", "fleet complete")
	return nil
}
