package update

import (
	"fmt"
	"strings"
	"time"
)

// BeginNodeApply registers a node-component job so observers share JobRegistry with tg/agent.
// Primary concurrency remains stack.TryAcquireApplyLock.
func BeginNodeApply(tag, jobID string) (*UpdateJob, error) {
	tag = strings.TrimSpace(tag)
	if tag != "" && !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	if jobID == "" {
		jobID = fmt.Sprintf("node-local-%d", time.Now().UnixNano())
	}
	j := &UpdateJob{
		JobID:         jobID,
		Component:     ComponentNode,
		InstanceID:    "local",
		TargetVersion: strings.TrimPrefix(tag, "v"),
		Asset:         AssetName(ComponentNode, PlatformFromRuntime()),
		State:         StateQueued,
	}
	return DefaultHostRegistry.Begin(j)
}

// FinishNodeApply walks legal transitions to succeeded or failed.
func FinishNodeApply(ok bool, errMsg string) (*UpdateJob, error) {
	j, found := DefaultHostRegistry.Get(ComponentNode, "local")
	if !found {
		return nil, fmt.Errorf("update: no node job")
	}
	if isTerminal(j.State) {
		return j, nil
	}
	if !ok {
		// From any non-terminal mid-state, only some allow direct → failed.
		return forceFailed(ComponentNode, "local", errMsg)
	}
	return forceSucceeded(ComponentNode, "local")
}

func forceSucceeded(c ComponentID, id InstanceID) (*UpdateJob, error) {
	path := []JobState{StateDownloading, StateVerifying, StateInstalling, StateHealthChecking, StateSucceeded}
	j, _ := DefaultHostRegistry.Get(c, id)
	cur := j.State
	for _, next := range path {
		if cur == StateSucceeded {
			j, ok := DefaultHostRegistry.Get(c, id)
			if !ok {
				return nil, fmt.Errorf("update: no job")
			}
			return j, nil
		}
		if CanTransition(cur, next) {
			jj, err := DefaultHostRegistry.Transition(c, id, next, "")
			if err != nil {
				return jj, err
			}
			cur = next
		}
	}
	j, ok := DefaultHostRegistry.Get(c, id)
	if !ok {
		return nil, fmt.Errorf("update: no job")
	}
	if j.State != StateSucceeded {
		return j, fmt.Errorf("update: could not reach succeeded from %s", j.State)
	}
	return j, nil
}

func forceFailed(c ComponentID, id InstanceID, msg string) (*UpdateJob, error) {
	j, _ := DefaultHostRegistry.Get(c, id)
	cur := j.State
	if CanTransition(cur, StateFailed) {
		return DefaultHostRegistry.Transition(c, id, StateFailed, msg)
	}
	// Step forward until failed is legal.
	for _, next := range []JobState{StateDownloading, StateVerifying, StateInstalling, StateHealthChecking} {
		if CanTransition(cur, next) {
			_, _ = DefaultHostRegistry.Transition(c, id, next, "")
			cur = next
		}
		if CanTransition(cur, StateFailed) {
			return DefaultHostRegistry.Transition(c, id, StateFailed, msg)
		}
	}
	return DefaultHostRegistry.Transition(c, id, StateFailed, msg)
}
