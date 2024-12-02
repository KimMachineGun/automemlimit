//go:build linux
// +build linux

package memlimit

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// GetMemoryLimit retrieves the memory limit for the current cgroup, supporting:
// - cgroup v1
// - cgroup v2
// - Hybrid mode (fallback to v1 if v2 fails)
func FromCgroup() (uint64, error) {
	return fromCgroup(detectCgroupVersion)
}

func FromCgroupV1() (uint64, error) {
	return fromCgroup(func(_ []mountInfo) (bool, bool) {
		return true, false
	})
}

func FromCgroupHybrid() (uint64, error) {
	return FromCgroup()
}

func FromCgroupV2() (uint64, error) {
	return fromCgroup(func(_ []mountInfo) (bool, bool) {
		return false, true
	})
}

func fromCgroup(versionDetector func(mis []mountInfo) (bool, bool)) (uint64, error) {
	mf, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return 0, fmt.Errorf("failed to open /proc/self/mountinfo: %w", err)
	}
	defer mf.Close()

	mis, err := parseMountInfo(mf)
	if err != nil {
		return 0, fmt.Errorf("failed to parse mountinfo: %w", err)
	}

	v1, v2 := versionDetector(mis)
	if !(v1 || v2) {
		return 0, ErrNoCgroup
	}

	cf, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return 0, fmt.Errorf("failed to open /proc/self/cgroup: %w", err)
	}
	defer cf.Close()

	chs, err := parseCgroupFile(cf)
	if err != nil {
		return 0, fmt.Errorf("failed to parse cgroup file: %w", err)
	}

	if v2 {
		limit, err := getMemoryLimitV2(chs, mis)
		if err == nil {
			return limit, nil
		} else if !v1 {
			return 0, err
		}
	}

	return getMemoryLimitV1(chs, mis)
}

func detectCgroupVersion(mis []mountInfo) (bool, bool) {
	var v1, v2 bool
	for _, mi := range mis {
		switch mi.FilesystemType {
		case "cgroup":
			v1 = true
		case "cgroup2":
			v2 = true
		}
	}
	return v1, v2
}

// getMemoryLimitV2 retrieves the memory limit for cgroup v2.
func getMemoryLimitV2(chs []cgroupHierarchy, mis []mountInfo) (uint64, error) {
	idx := slices.IndexFunc(chs, func(ch cgroupHierarchy) bool {
		return ch.HierarchyID == "0" && ch.ControllerList == ""
	})
	if idx == -1 {
		return 0, errors.New("cgroup v2 path not found")
	}
	relPath := chs[idx].CgroupPath

	idx = slices.IndexFunc(mis, func(mi mountInfo) bool {
		return mi.FilesystemType == "cgroup2"
	})
	if idx == -1 {
		return 0, errors.New("cgroup v2 mountpoint not found")
	}
	root, mountPoint := mis[idx].Root, mis[idx].MountPoint

	// Resolve the actual cgroup path
	cgroupPath, err := resolveCgroupPath(mountPoint, root, relPath)
	if err != nil {
		return 0, err
	}

	// Construct the path to memory.max
	memoryMaxPath := filepath.Join(cgroupPath, "memory.max")

	// Read the memory limit from memory.max
	return readMemoryLimitV2FromPath(memoryMaxPath)
}

// getMemoryLimitV1 retrieves the memory limit for cgroup v1.
func getMemoryLimitV1(chs []cgroupHierarchy, mis []mountInfo) (uint64, error) {
	idx := slices.IndexFunc(chs, func(ch cgroupHierarchy) bool {
		return slices.Contains(strings.Split(ch.ControllerList, ","), "memory")
	})
	if idx == -1 {
		return 0, errors.New("cgroup v1 path for memory controller not found")
	}
	relPath := chs[idx].CgroupPath

	idx = slices.IndexFunc(mis, func(mi mountInfo) bool {
		return mi.FilesystemType == "cgroup" && slices.Contains(strings.Split(mi.SuperOptions, ","), "memory")
	})
	if idx == -1 {
		return 0, errors.New("cgroup v1 mountpoint for memory controller not found")
	}
	root, mountPoint := mis[idx].Root, mis[idx].MountPoint

	// Resolve the actual cgroup path
	cgroupPath, err := resolveCgroupPath(mountPoint, root, relPath)
	if err != nil {
		return 0, err
	}

	// Retrieve the memory limit
	return readMemoryLimitV1FromPath(cgroupPath)
}

// readMemoryLimitV2FromPath reads the memory limit from the memory.max file for cgroup v2.
func readMemoryLimitV2FromPath(path string) (uint64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, ErrNoLimit
		}
		return 0, fmt.Errorf("failed to read memory.max: %w", err)
	}

	slimit := strings.TrimSpace(string(b))
	if slimit == "max" {
		return 0, ErrNoLimit
	}

	limit, err := strconv.ParseUint(slimit, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse memory.max value: %w", err)
	}

	return limit, nil
}

func getCgroupV1NoLimit() uint64 {
	ps := uint64(os.Getpagesize())
	return math.MaxInt64 / ps * ps
}

// readMemoryLimitV1FromPath reads the memory limit for cgroup v1 from the given path.
func readMemoryLimitV1FromPath(cgroupPath string) (uint64, error) {
	hml, err := readHierarchicalMemoryLimit(filepath.Join(cgroupPath, "memory.stats"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, fmt.Errorf("failed to read hierarchical_memory_limit: %w", err)
	} else if hml == 0 {
		hml = math.MaxUint64
	}

	b, err := os.ReadFile(filepath.Join(cgroupPath, "memory.limit_in_bytes"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return 0, fmt.Errorf("failed to read memory.limit_in_bytes: %w", err)
	}
	lib, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse memory.limit_in_bytes value: %w", err)
	} else if lib == 0 {
		hml = math.MaxUint64
	}

	limit := min(hml, lib)
	if limit >= getCgroupV1NoLimit() {
		return 0, ErrNoLimit
	}

	return limit, nil
}

// readHierarchicalMemoryLimit extracts hierarchical_memory_limit from memory.stats for cgroup v1.
func readHierarchicalMemoryLimit(statPath string) (uint64, error) {
	file, err := os.Open(statPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Split(line, " ")
		if len(fields) < 2 {
			return 0, fmt.Errorf("failed to parse memory.stat %q: not enough fields", line)
		}

		if fields[0] == "hierarchical_memory_limit" {
			if len(fields) > 2 {
				return 0, fmt.Errorf("failed to parse memory.stat %q: too many fields for hierarchical_memory_limit", line)
			}
			return strconv.ParseUint(fields[1], 10, 64)
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return 0, nil
}

// https://www.man7.org/linux/man-pages/man5/proc_pid_mountinfo.5.html
// 731 771 0:59 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw
//
// 36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue
// (1)(2)(3)   (4)   (5)      (6)      (7)   (8) (9)   (10)         (11)
//
// (1)  mount ID: a unique ID for the mount (may be reused after umount(2)).
// (2)  parent ID: the ID of the parent mount (or of self for the root of this mount namespace's mount tree).
// (3)  major:minor: the value of st_dev for files on this filesystem (see stat(2)).
// (4)  root: the pathname of the directory in the filesystem which forms the root of this mount.
// (5)  mount point: the pathname of the mount point relative to the process's root directory.
// (6)  mount options: per-mount options (see mount(2)).
// (7)  optional fields: zero or more fields of the form "tag[:value]"; see below.
// (8)  separator: the end of the optional fields is marked by a single hyphen.
// (9)  filesystem type: the filesystem type in the form "type[.subtype]".
// (10) mount source: filesystem-specific information or "none".
// (11) super options: per-superblock options (see mount(2)).
type mountInfo struct {
	Root           string
	MountPoint     string
	FilesystemType string
	SuperOptions   string
}

func parseMountInfo(r io.Reader) ([]mountInfo, error) {
	var (
		s   = bufio.NewScanner(r)
		mis []mountInfo
	)
	for s.Scan() {
		line := s.Text()

		fieldss := strings.SplitN(line, " - ", 2)
		if len(fieldss) != 2 {
			return nil, fmt.Errorf("failed to parse mountinfo %q: invalid separator", line)
		}

		fields1 := strings.Split(fieldss[0], " ")
		if len(fields1) < 6 {
			return nil, fmt.Errorf("failed to parse mountinfo %q: not enough fields1 %v", line, fields1)
		} else if len(fields1) > 7 {
			return nil, fmt.Errorf("failed to parse mountinfo %q: too many fields", line)
		} else if len(fields1) == 6 {
			fields1 = append(fields1, "")
		}

		fields2 := strings.Split(fieldss[1], " ")
		if len(fields2) < 3 {
			return nil, fmt.Errorf("failed to parse mountinfo %q: not enough fields2 %v", line, fields2)
		} else if len(fields2) > 3 {
			return nil, fmt.Errorf("failed to parse mountinfo %q: too many fields", line)
		}

		mis = append(mis, mountInfo{
			Root:           fields1[3],
			MountPoint:     fields1[4],
			FilesystemType: fields2[0],
			SuperOptions:   fields2[2],
		})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	return mis, nil
}

// https://www.man7.org/linux/man-pages/man7/cgroups.7.html
//
// 5:cpuacct,cpu,cpuset:/daemons
// (1)       (2)           (3)
//
// (1) hierarchy ID:
//
//	cgroups version 1 hierarchies, this field
//	contains a unique hierarchy ID number that can be
//	matched to a hierarchy ID in /proc/cgroups.  For the
//	cgroups version 2 hierarchy, this field contains the
//	value 0.
//
// (2) controller list:
//
//	For cgroups version 1 hierarchies, this field
//	contains a comma-separated list of the controllers
//	bound to the hierarchy.  For the cgroups version 2
//	hierarchy, this field is empty.
//
// (3) cgroup path:
//
//	This field contains the pathname of the control group
//	in the hierarchy to which the process belongs.  This
//	pathname is relative to the mount point of the
//	hierarchy.
type cgroupHierarchy struct {
	HierarchyID    string
	ControllerList string
	CgroupPath     string
}

func parseCgroupFile(r io.Reader) ([]cgroupHierarchy, error) {
	var (
		s   = bufio.NewScanner(r)
		chs []cgroupHierarchy
	)
	for s.Scan() {
		line := s.Text()

		fields := strings.Split(line, ":")
		if len(fields) != 3 {
			return nil, fmt.Errorf("failed to parse cgroup file %q: invalid separator", line)
		}

		chs = append(chs, cgroupHierarchy{
			HierarchyID:    fields[0],
			ControllerList: fields[1],
			CgroupPath:     fields[2],
		})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	return chs, nil
}

func resolveCgroupPath(mountpoint, root, cgroupRelPath string) (string, error) {
	root = filepath.Clean(strings.TrimPrefix(root, "/"))
	cgroupRelPath = filepath.Clean(strings.TrimPrefix(cgroupRelPath, "/"))

	if root == cgroupRelPath || (root == "." && cgroupRelPath == ".") {
		return mountpoint, nil
	}

	if strings.HasPrefix(cgroupRelPath, root) {
		relativePath := strings.TrimPrefix(cgroupRelPath, root)
		finalPath := filepath.Join(mountpoint, relativePath)

		if _, err := os.Stat(finalPath); os.IsNotExist(err) {
			return "", fmt.Errorf("resolved cgroup path does not exist: %s", finalPath)
		}

		return finalPath, nil
	}

	return "", fmt.Errorf("invalid cgroup path: %s is not under root %s", cgroupRelPath, root)
}
