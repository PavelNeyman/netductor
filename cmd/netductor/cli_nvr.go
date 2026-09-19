package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/audit"
	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/nvr"
)

func runNVR(args []string) {
	if len(args) < 1 {
		printNVRHelp()
		os.Exit(2)
	}
	switch args[0] {
	case "help", "-h", "--help":
		printNVRHelp()
	case "config":
		runNVRConfig(args[1:])
	case "cameras", "camera":
		runNVRCameras(args[1:])
	case "retention", "rotate":
		rep, err := nvr.RunRetention(nvr.LoadConfig())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rep)
	case "segments":
		cam := ""
		if len(args) > 1 {
			cam = args[1]
		}
		files, err := nvr.ListSegmentFiles(nvr.LoadConfig().Path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for _, f := range files {
			if cam != "" && f.Camera != cam {
				continue
			}
			fmt.Printf("%s\t%d\t%s\t%s\n", f.ModTime.Format("2006-01-02T15:04:05"), f.Size, f.Camera, f.Path)
		}
	case "leases":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor nvr leases <device_id> [--wait]")
			os.Exit(2)
		}
		id := edge.EnqueueCmd(args[1], "dhcp_leases", "")
		if id == "" {
			fmt.Fprintln(os.Stderr, "enqueue failed")
			os.Exit(1)
		}
		wait := true
		for _, a := range args[2:] {
			if a == "--no-wait" {
				wait = false
			}
		}
		if !wait {
			fmt.Println("cmd_id", id)
			return
		}
		fmt.Fprintln(os.Stderr, "waiting for", id, "…")
		res, err := edge.WaitCmdResult(id, 120*time.Second)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			fmt.Println("cmd_id", id)
			os.Exit(1)
		}
		if r, ok := res["result"].(string); ok {
			fmt.Println(r)
		} else {
			b, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println(string(b))
		}
	case "wifi-clients":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor nvr wifi-clients <device_id>")
			os.Exit(2)
		}
		id := edge.EnqueueCmd(args[1], "wifi_clients", "")
		if id == "" {
			fmt.Fprintln(os.Stderr, "enqueue failed")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "waiting for", id, "…")
		res, err := edge.WaitCmdResult(id, 120*time.Second)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if r, ok := res["result"].(string); ok {
			fmt.Println(r)
		} else {
			b, _ := json.MarshalIndent(res, "", "  ")
			fmt.Println(string(b))
		}
	case "dhcp-static":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: netductor nvr dhcp-static <device_id> <mac> <ip> [name]")
			os.Exit(2)
		}
		name := ""
		if len(args) > 4 {
			name = args[4]
		}
		arg := "mac=" + args[2] + "|ip=" + args[3]
		if name != "" {
			arg += "|name=" + name
		}
		id := edge.EnqueueCmd(args[1], "dhcp_static", arg)
		fmt.Println("cmd_id", id)
	case "record":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor nvr record start|stop <camera_id>")
			os.Exit(2)
		}
		c, ok := nvr.GetCamera(args[2])
		if !ok {
			fmt.Fprintln(os.Stderr, "camera not found")
			os.Exit(1)
		}
		switch args[1] {
		case "start":
			url := nvr.RTSPURL(c)
			if url == "" || c.SiteID == "" {
				fmt.Fprintln(os.Stderr, "need site_id + rtsp secret/ip")
				os.Exit(1)
			}
			seg := nvr.LoadConfig().SegmentSec
			if seg <= 0 {
				seg = 300
			}
			arg := c.ID + "|" + url + "|" + strconv.Itoa(seg)
			id := edge.EnqueueCmd(c.SiteID, "nvr_record_start", arg)
			if id == "" {
				fmt.Fprintln(os.Stderr, "enqueue failed")
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "waiting", id)
			res, err := edge.WaitCmdResult(id, 60*time.Second)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println(res["result"])
		case "stop":
			id := edge.EnqueueCmd(c.SiteID, "nvr_record_stop", c.ID)
			res, err := edge.WaitCmdResult(id, 60*time.Second)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println(res["result"])
		default:
			fmt.Fprintln(os.Stderr, "start|stop")
			os.Exit(2)
		}
	case "probe":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor nvr probe <camera_id>")
			os.Exit(2)
		}
		c, ok := nvr.GetCamera(args[1])
		if !ok {
			fmt.Fprintln(os.Stderr, "camera not found")
			os.Exit(1)
		}
		url := nvr.RTSPURL(c)
		if url == "" {
			// probe TCP only from site without password
			if c.LANIP == "" || c.SiteID == "" {
				fmt.Fprintln(os.Stderr, "need lan_ip+site or rtsp secret")
				os.Exit(1)
			}
			port := c.RTSPPort
			if port == 0 {
				port = 554
			}
			arg := c.LANIP + ":" + strconv.Itoa(port)
			id := edge.EnqueueCmd(c.SiteID, "rtsp_probe", arg)
			fmt.Fprintln(os.Stderr, "tcp probe", id)
			res, err := edge.WaitCmdResult(id, 60*time.Second)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println(res["result"])
			return
		}
		// full URL probe via agent (password in cmd — 0600 state)
		id := edge.EnqueueCmd(c.SiteID, "rtsp_probe", url)
		if id == "" {
			fmt.Fprintln(os.Stderr, "enqueue failed")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "waiting", id)
		res, err := edge.WaitCmdResult(id, 90*time.Second)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(res["result"])
	case "storage":
		b, _ := json.MarshalIndent(nvr.GetStorageStatus(), "", "  ")
		fmt.Println(string(b))
	case "prepare-storage":
		if err := nvr.PrepareStorage(len(args) > 1 && args[1] == "--print-only"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "recorder":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: netductor nvr recorder start|stop <camera_id>")
			os.Exit(2)
		}
		switch args[1] {
		case "start":
			c, ok := nvr.GetCamera(args[2])
			if !ok {
				fmt.Fprintln(os.Stderr, "camera not found")
				os.Exit(1)
			}
			if err := nvr.StartRecorder(c); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("started", args[2])
		case "stop":
			nvr.StopRecorder(args[2])
			fmt.Println("stopped", args[2])
		default:
			fmt.Fprintln(os.Stderr, "start|stop")
			os.Exit(2)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown nvr subcommand: %s\n", args[0])
		printNVRHelp()
		os.Exit(2)
	}
}

func printNVRHelp() {
	fmt.Print(`netductor nvr — cameras / record / retention

  config [show|set key=value ...]
  cameras list|add|delete
  leases <device_id>
  wifi-clients <device_id>
  dhcp-static <device_id> <mac> <ip> [name]
  retention|rotate
  segments [camera_id]
  record start|stop <camera_id>
  probe <camera_id>
  storage
  prepare-storage [--print-only]
  recorder start|stop <camera_id>

Config keys: path, segment_sec, retention_days, max_gb, min_free_gb,
  rotate_interval_sec, record_enabled, storage_backend
`)
}

func runNVRConfig(args []string) {
	if len(args) == 0 || args[0] == "show" {
		b, _ := json.MarshalIndent(nvr.LoadConfig(), "", "  ")
		fmt.Println(string(b))
		if r, ok := nvr.LastRetentionReport(); ok {
			fmt.Println("--- last retention ---")
			b, _ = json.MarshalIndent(r, "", "  ")
			fmt.Println(string(b))
		}
		return
	}
	if args[0] != "set" {
		fmt.Fprintln(os.Stderr, "usage: netductor nvr config show|set k=v ...")
		os.Exit(2)
	}
	cfg := nvr.LoadConfig()
	for _, a := range args[1:] {
		k, v, ok := strings.Cut(a, "=")
		if !ok {
			continue
		}
		switch strings.ToLower(k) {
		case "path":
			cfg.Path = v
		case "storage_backend":
			cfg.StorageBackend = v
		case "segment_sec":
			cfg.SegmentSec, _ = strconv.Atoi(v)
		case "retention_days":
			cfg.RetentionDays, _ = strconv.Atoi(v)
		case "max_gb":
			fmt.Sscanf(v, "%f", &cfg.MaxGB)
		case "min_free_gb":
			fmt.Sscanf(v, "%f", &cfg.MinFreeGB)
		case "rotate_interval_sec":
			cfg.RotateIntervalSec, _ = strconv.Atoi(v)
		case "record_enabled":
			cfg.RecordEnabled = v == "1" || strings.EqualFold(v, "true") || v == "yes"
		}
	}
	if err := nvr.SaveConfig(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	audit.Log("cli", "nvr.config", "", "")
	b, _ := json.MarshalIndent(nvr.LoadConfig(), "", "  ")
	fmt.Println(string(b))
}

func runNVRCameras(args []string) {
	if len(args) < 1 {
		args = []string{"list"}
	}
	switch args[0] {
	case "list":
		for _, c := range nvr.ListCameras() {
			fmt.Printf("%s\tsite=%s\t%s\t%s\t%s\trecord=%v\n", c.ID, c.SiteID, c.Name, c.MAC, c.LANIP, c.Record)
		}
	case "delete":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor nvr cameras delete <id>")
			os.Exit(2)
		}
		fmt.Println(nvr.DeleteCamera(args[1]))
	case "add":
		// name= site= mac= ip= user= password= path=
		c := nvr.Camera{Enabled: true, Record: true, RTSPPath: "/stream1", RTSPPort: 554, Features: map[string]bool{"ptz": true}}
		pass := ""
		for _, a := range args[1:] {
			k, v, ok := strings.Cut(a, "=")
			if !ok {
				continue
			}
			switch k {
			case "name":
				c.Name = v
			case "site", "site_id":
				c.SiteID = v
			case "mac":
				c.MAC = v
			case "ip", "lan_ip":
				c.LANIP = v
			case "user", "rtsp_user":
				c.RTSPUser = v
			case "password", "rtsp_password":
				pass = v
			case "path", "rtsp_path":
				c.RTSPPath = v
			}
		}
		if c.Name == "" || c.SiteID == "" {
			fmt.Fprintln(os.Stderr, "need name= and site=")
			os.Exit(2)
		}
		out, err := nvr.UpsertCamera(c)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if pass != "" {
			ref := out.ID
			out.SecretRef = ref
			out, _ = nvr.UpsertCamera(out)
			_ = nvr.SetSecret(ref, pass)
		}
		audit.Log("cli", "nvr.camera.add", out.ID, out.Name)
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
	default:
		fmt.Fprintln(os.Stderr, "list|add|delete")
		os.Exit(2)
	}
}
