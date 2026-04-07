package main

// NimOS Storage — Startup, detection, disk scanning, pool dirs

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ─── Startup functions (called from main.go) ────────────────────────────────

func zfsAutoImportOnStartup() {
	if !hasZfs {
		return
	}
	// Import all known ZFS pools
	run("zpool import -a -N 2>/dev/null || true")

	conf := getStorageConfigFull()
	confPools, _ := conf["pools"].([]interface{})
	for _, poolRaw := range confPools {
		pm, _ := poolRaw.(map[string]interface{})
		poolType, _ := pm["type"].(string)
		if poolType != "zfs" {
			continue
		}
		zpoolName, _ := pm["zpoolName"].(string)
		mountPoint, _ := pm["mountPoint"].(string)
		if zpoolName == "" || mountPoint == "" {
			continue
		}
		// Check if pool is imported
		if _, ok := runSafe("zpool", "list", "-H", "-o", "name", zpoolName); !ok {
			runSafe("zpool", "import", zpoolName)
		}
		// Set mount point and mount
		runSafe("zfs", "set", "mountpoint="+mountPoint, zpoolName)
		run("zfs mount -a 2>/dev/null || true")
	}
	logMsg("ZFS auto-import completed")
}

func btrfsAutoMountOnStartup() {
	if !hasBtrfs {
		return
	}
	conf := getStorageConfigFull()
	confPools, _ := conf["pools"].([]interface{})
	for _, poolRaw := range confPools {
		pm, _ := poolRaw.(map[string]interface{})
		poolType, _ := pm["type"].(string)
		if poolType != "btrfs" {
			continue
		}
		mountPoint, _ := pm["mountPoint"].(string)
		if mountPoint == "" {
			continue
		}
		// Try mount from fstab
		runSafe("mount", mountPoint)
	}
	logMsg("Btrfs auto-mount completed")
}

func startupStorage() {
	logMsg("startup: Storage initialization...")
	conf := getStorageConfigFull()
	confPools, _ := conf["pools"].([]interface{})
	if len(confPools) == 0 {
		logMsg("startup: No pools configured")
		return
	}
	// Verify pools are mounted and create dirs if needed
	for _, poolRaw := range confPools {
		pm, _ := poolRaw.(map[string]interface{})
		mountPoint, _ := pm["mountPoint"].(string)
		poolName, _ := pm["name"].(string)
		if mountPoint == "" {
			continue
		}
		if isPathOnMountedPool(mountPoint) {
			logMsg("startup: Pool '%s' mounted at %s", poolName, mountPoint)
			createPoolDirs(mountPoint)
		} else {
			logMsg("startup: WARNING — Pool '%s' NOT mounted at %s", poolName, mountPoint)
		}
	}
	logMsg("startup: Storage initialization complete")
}

func startStorageMonitoring() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			checkStorageHealthGo()
		}
	}()
}

func startZfsScheduler() {
	// TODO: reimplement with new storage_health.go
	logMsg("ZFS scheduler: stub (pending rewrite)")
}

// ─── Detection (called from hardware.go) ─────────────────────────────────────

func detectBtrfs() {
	if _, ok := run("which mkfs.btrfs 2>/dev/null"); ok {
		hasBtrfs = true
		logMsg("Btrfs: available")
	} else {
		logMsg("Btrfs: not available")
	}
}

// ─── Disk detection ──────────────────────────────────────────────────────────

func detectStorageDisksGo() map[string]interface{} {
	// TODO: rewrite with storage_disks.go from plan v2
	// For now: minimal implementation that works
	result := map[string]interface{}{
		"eligible":    []interface{}{},
		"nvme":        []interface{}{},
		"usb":         []interface{}{},
		"provisioned": []interface{}{},
	}

	lsblkRaw, ok := run("lsblk -J -b -o NAME,SIZE,TYPE,ROTA,MOUNTPOINT,MODEL,SERIAL,TRAN,RM,FSTYPE,LABEL,PKNAME 2>/dev/null")
	if !ok || lsblkRaw == "" {
		return result
	}

	var data struct {
		BlockDevices []json.RawMessage `json:"blockdevices"`
	}
	if json.Unmarshal([]byte(lsblkRaw), &data) != nil {
		return result
	}

	rootDisk := findRootDiskGo(lsblkRaw)
	confPools := getStorageConfigFull()
	poolDisks := map[string]bool{}
	if pools, ok := confPools["pools"].([]interface{}); ok {
		for _, p := range pools {
			pm, _ := p.(map[string]interface{})
			if disks, ok := pm["disks"].([]interface{}); ok {
				for _, d := range disks {
					if ds, _ := d.(string); ds != "" {
						poolDisks[ds] = true
					}
				}
			}
		}
	}

	var eligible, nvme, usb, provisioned []interface{}

	for _, raw := range data.BlockDevices {
		var dev map[string]interface{}
		json.Unmarshal(raw, &dev)

		devType, _ := dev["type"].(string)
		if devType != "disk" {
			continue
		}
		devName, _ := dev["name"].(string)

		// Whitelist: only sd*, nvme*, vd*
		validPrefix := false
		for _, prefix := range []string{"sd", "nvme", "vd"} {
			if strings.HasPrefix(devName, prefix) {
				validPrefix = true
				break
			}
		}
		if !validPrefix {
			continue
		}

		size := jsonToInt64(dev["size"])
		if size < 1024*1024*1024 { // < 1GB
			continue
		}

		transport, _ := dev["tran"].(string)
		model, _ := dev["model"].(string)
		serial, _ := dev["serial"].(string)
		rotaBool := jsonToBool(dev["rota"])
		removableBool := jsonToBool(dev["rm"])

		diskInfo := map[string]interface{}{
			"name":          devName,
			"path":          "/dev/" + devName,
			"model":         strings.TrimSpace(model),
			"serial":        strings.TrimSpace(serial),
			"size":          size,
			"sizeFormatted": formatBytes(size),
			"transport":     transport,
			"rotational":    rotaBool,
			"removable":     removableBool,
			"isBoot":        devName == rootDisk,
			"partitions":    []interface{}{},
		}

		// Parse partitions
		var partitions []interface{}
		if children, ok := dev["children"].([]interface{}); ok {
			for _, child := range children {
				cm, ok := child.(map[string]interface{})
				if !ok {
					continue
				}
				partSize := jsonToInt64(cm["size"])
				partitions = append(partitions, map[string]interface{}{
					"name":       cm["name"],
					"path":       "/dev/" + fmt.Sprintf("%v", cm["name"]),
					"size":       partSize,
					"fstype":     cm["fstype"],
					"label":      cm["label"],
					"mountpoint": cm["mountpoint"],
				})
			}
		}
		if partitions == nil {
			partitions = []interface{}{}
		}
		diskInfo["partitions"] = partitions
		diskInfo["hasExistingData"] = len(partitions) > 0

		// Classify
		if devName == rootDisk {
			continue // boot disk — never show
		}

		if poolDisks["/dev/"+devName] {
			diskInfo["classification"] = "provisioned"
			provisioned = append(provisioned, diskInfo)
			continue
		}

		// USB pendrive: USB + removable + < 10GB
		if transport == "usb" && removableBool && size < 10*1024*1024*1024 {
			diskInfo["classification"] = "usb"
			usb = append(usb, diskInfo)
			continue
		}

		// NVMe that isn't boot
		if strings.HasPrefix(devName, "nvme") {
			diskInfo["classification"] = "nvme"
			nvme = append(nvme, diskInfo)
			continue
		}

		// Everything else is eligible
		diskInfo["classification"] = "eligible"
		eligible = append(eligible, diskInfo)
	}

	if eligible == nil { eligible = []interface{}{} }
	if nvme == nil { nvme = []interface{}{} }
	if usb == nil { usb = []interface{}{} }
	if provisioned == nil { provisioned = []interface{}{} }

	result["eligible"] = eligible
	result["nvme"] = nvme
	result["usb"] = usb
	result["provisioned"] = provisioned
	return result
}

func findRootDiskGo(lsblkJSON string) string {
	var data struct {
		BlockDevices []struct {
			Name     string `json:"name"`
			Children []struct {
				Mountpoint interface{} `json:"mountpoint"`
			} `json:"children"`
			Mountpoint interface{} `json:"mountpoint"`
		} `json:"blockdevices"`
	}
	json.Unmarshal([]byte(lsblkJSON), &data)
	for _, dev := range data.BlockDevices {
		for _, child := range dev.Children {
			if mp, _ := child.Mountpoint.(string); mp == "/" {
				return dev.Name
			}
		}
		if mp, _ := dev.Mountpoint.(string); mp == "/" {
			return dev.Name
		}
	}
	return ""
}

// ─── Pool dirs ───────────────────────────────────────────────────────────────

func createPoolDirs(mountPoint string) {
	dirs := []string{"shares", "system-backup/config", "system-backup/snapshots"}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(mountPoint, d), 0755)
	}
}

// ─── Health ──────────────────────────────────────────────────────────────────

func checkStorageHealthGo() []map[string]interface{} {
	var alerts []map[string]interface{}
	pools := getStoragePoolsGo()
	for _, pool := range pools {
		pct, _ := pool["usagePercent"].(int)
		name, _ := pool["name"].(string)
		if pct >= 95 {
			alerts = append(alerts, map[string]interface{}{"severity": "critical", "pool": name, "message": fmt.Sprintf("Pool %s is %d%% full", name, pct)})
		} else if pct >= 85 {
			alerts = append(alerts, map[string]interface{}{"severity": "warning", "pool": name, "message": fmt.Sprintf("Pool %s is %d%% full", name, pct)})
		}
	}
	if alerts == nil { alerts = []map[string]interface{}{} }
	storageAlertsMu.Lock()
	storageAlertsGo = alerts
	storageAlertsMu.Unlock()
	return alerts
}

// ─── Wipe (implemented in storage_wipe.go) ──────────────────────────────────

// ─── Scan / Restore (stubs) ──────────────────────────────────────────────────

func rescanSCSIBuses() {
	entries, err := os.ReadDir("/sys/class/scsi_host")
	if err != nil {
		return
	}
	for _, e := range entries {
		scanPath := filepath.Join("/sys/class/scsi_host", e.Name(), "scan")
		os.WriteFile(scanPath, []byte("- - -"), 0200)
	}
	run("udevadm settle --timeout=5 2>/dev/null || true")
}

func scanForRestorablePoolsGo() []map[string]interface{} {
	return []map[string]interface{}{}
}

func backupConfigToPoolGo() {
	// TODO: reimplement
}

func appendFstab(uuid, mountPoint, filesystem string) {
	existing, _ := os.ReadFile("/etc/fstab")
	if strings.Contains(string(existing), mountPoint) {
		return
	}
	opts := "defaults,nofail,noatime"
	if filesystem == "btrfs" {
		opts = "defaults,nofail,noatime,compress=zstd"
	}
	entry := fmt.Sprintf("UUID=%s %s %s %s 0 2\n", uuid, mountPoint, filesystem, opts)
	f, err := os.OpenFile("/etc/fstab", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(entry)
	log.Printf("appendFstab: added %s", mountPoint)
}

// ─── ZFS Pool Info (needed by getStoragePoolsGo) ────────────────────────────

// enrichDisksWithSmart takes a flat disk name list and returns enriched objects
// with SMART status from the cached monitor data. Does NOT run smartctl — only
// reads from smartHistory to avoid false positives from stale or slow queries.
// The pool-level status/health is NEVER modified by this function.
