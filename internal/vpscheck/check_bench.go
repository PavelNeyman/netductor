package vpscheck

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Heavier bench closer to YABS/bench.sh intent (still local, no remote scripts).

func checkBenchExtra(ctx context.Context) []Result {
	var out []Result
	out = append(out, checkSysbenchMemory(ctx))
	out = append(out, checkCPUInfo())
	out = append(out, checkDiskFIO(ctx))
	return out
}

func checkSysbenchMemory(ctx context.Context) Result {
	start := time.Now()
	r := Result{ID: "sysbench_mem", TitleEN: "Memory sysbench", TitleRU: "Память sysbench"}
	if _, err := exec.LookPath("sysbench"); err != nil {
		r.Status = StatusSkip
		r.Detail = "sysbench missing"
		return r
	}
	cmd := exec.CommandContext(ctx, "sysbench", "memory", "run", "--threads=1")
	out, err := cmd.CombinedOutput()
	r.DurationMs = time.Since(start).Milliseconds()
	s := string(out)
	line := ""
	for _, ln := range strings.Split(s, "\n") {
		if strings.Contains(ln, "transferred") || strings.Contains(ln, "ops/sec") || strings.Contains(ln, "Total operations") {
			line = strings.TrimSpace(ln)
			if strings.Contains(ln, "transferred") {
				break
			}
		}
	}
	if err != nil {
		r.Status = StatusWarn
		r.Detail = truncate(s, 250)
		return r
	}
	r.Status = StatusOK
	if line != "" {
		r.Detail = line
	} else {
		r.Detail = truncate(s, 200)
	}
	return r
}

func checkCPUInfo() Result {
	r := Result{ID: "cpu_info", TitleEN: "CPU info", TitleRU: "CPU info"}
	r.Detail = fmt.Sprintf("GOMAXPROCS=%d NumCPU=%d", runtime.GOMAXPROCS(0), runtime.NumCPU())
	// /proc/cpuinfo model
	b, err := os.ReadFile("/proc/cpuinfo")
	if err == nil {
		for _, ln := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(ln, "model name") || strings.HasPrefix(ln, "Hardware") {
				r.Detail += " | " + strings.TrimSpace(ln)
				break
			}
		}
		// count processors
		n := strings.Count(string(b), "processor")
		if n > 0 {
			r.Detail += fmt.Sprintf(" | processors≈%d", n)
		}
	}
	r.Status = StatusInfo
	return r
}

func checkDiskFIO(ctx context.Context) Result {
	start := time.Now()
	r := Result{ID: "disk_fio", TitleEN: "Disk fio seq", TitleRU: "Диск fio"}
	if _, err := exec.LookPath("fio"); err != nil {
		r.Status = StatusSkip
		r.Detail = "fio not installed"
		return r
	}
	tmp := "/tmp/nd-fio-test"
	cmd := exec.CommandContext(ctx, "fio",
		"--name=ndseq", "--directory=/tmp", "--filename="+tmp,
		"--rw=write", "--bs=1m", "--size=256m", "--iodepth=1",
		"--direct=1", "--runtime=15", "--time_based=0",
		"--group_reporting", "--output-format=terse",
	)
	out, err := cmd.CombinedOutput()
	_ = os.Remove(tmp)
	r.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		r.Status = StatusWarn
		r.Detail = truncate(string(out)+" "+err.Error(), 300)
		return r
	}
	r.Status = StatusOK
	r.Detail = truncate(strings.TrimSpace(string(out)), 300)
	return r
}
