package update

// JobSnapshot is a JSON-friendly view of an update job for UI.
type JobSnapshot struct {
	Component     string `json:"component"`
	InstanceID    string `json:"instance_id"`
	State         string `json:"state"`
	TargetVersion string `json:"target_version"`
	DisplayLabel  string `json:"display_label"` // from UserFacingVersionLabel
	DisplayKind   string `json:"display_kind"`  // running | target | ...
	LastError     string `json:"last_error,omitempty"`
	JobID         string `json:"job_id,omitempty"`
}

// HostJobsSnapshot returns known jobs for node/tg/agent on instance "local".
func HostJobsSnapshot() []JobSnapshot {
	var out []JobSnapshot
	for _, c := range []ComponentID{ComponentNode, ComponentTG, ComponentAgent} {
		j, ok := DefaultHostRegistry.Get(c, "local")
		if !ok {
			continue
		}
		confirmed := j.State == StateSucceeded
		kind, label := UserFacingVersionLabel(j.TargetVersion, confirmed, j.TargetVersion)
		out = append(out, JobSnapshot{
			Component:     string(c),
			InstanceID:    string(j.InstanceID),
			State:         string(j.State),
			TargetVersion: j.TargetVersion,
			DisplayLabel:  label,
			DisplayKind:   kind,
			LastError:     j.LastError,
			JobID:         j.JobID,
		})
	}
	return out
}
