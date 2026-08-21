package memlimit

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestParseMountInfoLine(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    mountInfo
		wantErr string
	}{
		{
			name:  "valid_line_with_optional_field",
			input: "36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue",
			want: mountInfo{
				Root:           "/mnt1",
				MountPoint:     "/mnt2",
				FilesystemType: "ext3",
				SuperOptions:   "rw,errors=continue",
			},
		},
		{
			name:  "valid_line_without_optional_field",
			input: "731 771 0:59 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw",
			want: mountInfo{
				Root:           "/sysrq-trigger",
				MountPoint:     "/proc/sysrq-trigger",
				FilesystemType: "proc",
				SuperOptions:   "rw",
			},
		},
		{
			name:  "valid_line_with_minimal_fields",
			input: "25 1 0:22 / /dev rw - devtmpfs udev rw",
			want: mountInfo{
				Root:           "/",
				MountPoint:     "/dev",
				FilesystemType: "devtmpfs",
				SuperOptions:   "rw",
			},
		},
		{
			name:  "escaped_path_fields",
			input: `25 1 0:22 /root\040dir\011tab\012line\134slash /mount\040dir\011tab\012line\134slash rw - cgroup2 cgroup rw`,
			want: mountInfo{
				Root:           "/root dir\ttab\nline\\slash",
				MountPoint:     "/mount dir\ttab\nline\\slash",
				FilesystemType: "cgroup2",
				SuperOptions:   "rw",
			},
		},
		{
			name:  "escaped_backslash_is_not_redecoded",
			input: `25 1 0:22 /literal\134040 /literal\134134 rw - cgroup2 cgroup rw`,
			want: mountInfo{
				Root:           `/literal\040`,
				MountPoint:     `/literal\134`,
				FilesystemType: "cgroup2",
				SuperOptions:   "rw",
			},
		},
		{
			name:  "unknown_path_escape_is_preserved",
			input: `25 1 0:22 /literal\043 /literal\777 rw - cgroup2 cgroup rw`,
			want: mountInfo{
				Root:           `/literal\043`,
				MountPoint:     `/literal\777`,
				FilesystemType: "cgroup2",
				SuperOptions:   "rw",
			},
		},
		{
			name:    "no_separator",
			input:   "36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 ext3 /dev/root rw,errors=continue",
			wantErr: "invalid separator",
		},
		{
			name:    "not_enough_fields_on_left_side",
			input:   "36 35 98:0 /mnt1 /mnt2 - ext3 /dev/root rw,errors=continue",
			wantErr: "not enough fields before separator: [36 35 98:0 /mnt1 /mnt2]",
		},
		{
			name:    "not_enough_fields_on_right_side",
			input:   "36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3",
			wantErr: "not enough fields after separator: [ext3]",
		},
		{
			name:    "empty_line",
			input:   "",
			wantErr: "empty line",
		},
		{
			name:  "six_fields_on_left_side",
			input: "100 1 8:2 / /data rw - ext4 /dev/sda2 rw,relatime",
			want: mountInfo{
				Root:           "/",
				MountPoint:     "/data",
				FilesystemType: "ext4",
				SuperOptions:   "rw,relatime",
			},
		},
		{
			name:  "multiple_optional_fields_issue_26",
			input: "465 34 253:0 / / rw,relatime shared:409 master:1 - xfs /dev/mapper/fedora-root rw,seclabel,attr2,inode64,logbufs=8,logbsize=32k,noquota",
			want: mountInfo{
				Root:           "/",
				MountPoint:     "/",
				FilesystemType: "xfs",
				SuperOptions:   "rw,seclabel,attr2,inode64,logbufs=8,logbsize=32k,noquota",
			},
		},
		{
			name:  "super_options_have_spaces_issue_28",
			input: `1391 1160 0:151 / /Docker/host rw,noatime - 9p C:\134Program\040Files\134Docker\134Docker\134resources rw,dirsync,aname=drvfs;path=C:\Program Files\Docker\Docker\resources;symlinkroot=/mnt/,mmap,access=client,msize=65536,trans=fd,rfd=3,wfd=3`,
			want: mountInfo{
				Root:           "/",
				MountPoint:     "/Docker/host",
				FilesystemType: "9p",
				SuperOptions:   `rw,dirsync,aname=drvfs;path=C:\Program Files\Docker\Docker\resources;symlinkroot=/mnt/,mmap,access=client,msize=65536,trans=fd,rfd=3,wfd=3`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMountInfoLine(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("parseMountInfoLine() error = %v, want nil", err)
				}
			} else if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("parseMountInfoLine() error = %v, want error %q", err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseMountInfoLine() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseCgroupHierarchyLine(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    cgroupHierarchy
		wantErr string
	}{
		{
			name:  "valid_line_with_multiple_controllers",
			input: "5:cpuacct,cpu,cpuset:/daemons",
			want: cgroupHierarchy{
				HierarchyID:    "5",
				ControllerList: "cpuacct,cpu,cpuset",
				CgroupPath:     "/daemons",
			},
		},
		{
			name:  "valid_line_cgroup_v2",
			input: "0::/system.slice/docker.service",
			want: cgroupHierarchy{
				HierarchyID:    "0",
				ControllerList: "",
				CgroupPath:     "/system.slice/docker.service",
			},
		},
		{
			name:    "invalid_only_two_fields",
			input:   "5:cpuacct,cpu,cpuset",
			wantErr: "not enough fields: [5 cpuacct,cpu,cpuset]",
		},
		{
			name:  "valid_line_with_colon_in_cgroup_path",
			input: "5:cpuacct,cpu,cpuset:/daemons:extra",
			want: cgroupHierarchy{
				HierarchyID:    "5",
				ControllerList: "cpuacct,cpu,cpuset",
				CgroupPath:     "/daemons:extra",
			},
		},
		{
			name:    "empty_line",
			input:   "",
			wantErr: "empty line",
		},
		{
			name:  "empty_controller_list",
			input: "2::/my_cgroup",
			want: cgroupHierarchy{
				HierarchyID:    "2",
				ControllerList: "",
				CgroupPath:     "/my_cgroup",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCgroupHierarchyLine(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("parseCgroupHierarchyLine() error = %v, want nil", err)
				}
			} else if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("parseCgroupHierarchyLine() error = %v, want error %q", err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseCgroupHierarchyLine() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestResolveCgroupPath(t *testing.T) {
	tests := []struct {
		name          string
		mountPoint    string
		root          string
		cgroupRelPath string
		want          string
		wantErr       string
	}{
		{
			name:          "exact_match_root_paths",
			mountPoint:    "/fake/mount",
			root:          "/",
			cgroupRelPath: "/",
			want:          "/fake/mount",
		},
		{
			name:          "valid_subpath_under_root",
			mountPoint:    "/fake/mount",
			root:          "/container0",
			cgroupRelPath: "/container0/group1",
			want:          filepath.Join("/fake/mount", "group1"),
		},
		{
			name:          "valid_subpath_with_dotdot_prefix",
			mountPoint:    "/fake/mount",
			root:          "/",
			cgroupRelPath: "/..foo",
			want:          filepath.Join("/fake/mount", "..foo"),
		},
		{
			name:          "aligned_namespace_escape",
			mountPoint:    "/fake/mount",
			root:          "/../..",
			cgroupRelPath: "/../../app",
			want:          filepath.Join("/fake/mount", "app"),
		},
		{
			name:          "namespace_escape_above_mount_root",
			mountPoint:    "/fake/mount",
			root:          "/..",
			cgroupRelPath: "/../../app",
			want:          "",
		},
		{
			name:          "relative_path_escaping_mount_root",
			mountPoint:    "/fake/mount",
			root:          "/a",
			cgroupRelPath: "/a/b/../../..",
			want:          "",
		},
		{
			name:          "invalid_relative_resolution",
			mountPoint:    "/fake/mount",
			root:          "",
			cgroupRelPath: "/container0",
			wantErr:       "cgroup root and path must be absolute",
		},
		{
			name:          "non_descendant_path_returns_empty_path",
			mountPoint:    "/fake/mount",
			root:          "/container0",
			cgroupRelPath: "/other_container",
			want:          "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveCgroupPath(tt.mountPoint, tt.root, tt.cgroupRelPath)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("resolveCgroupPath() error = %v, want nil", err)
				}
			} else if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("resolveCgroupPath() error = %v, want error %q", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("resolveCgroupPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSelectMemoryController(t *testing.T) {
	tests := []struct {
		name      string
		chs       []cgroupHierarchy
		mis       []mountInfo
		want      memoryController
		wantErrIs error
		wantErr   string
	}{
		{
			name: "legacy_selects_v1",
			chs: []cgroupHierarchy{
				{HierarchyID: "1", ControllerList: "memory", CgroupPath: "/pod/abc"},
			},
			mis: []mountInfo{
				{FilesystemType: "cgroup", MountPoint: "/sys/fs/cgroup/memory", SuperOptions: "rw,memory"},
			},
			want: memoryController{path: "/pod/abc"},
		},
		{
			name: "unified_selects_v2",
			chs: []cgroupHierarchy{
				{HierarchyID: "0", ControllerList: "", CgroupPath: "/pod/abc"},
			},
			mis: []mountInfo{
				{FilesystemType: "cgroup2", MountPoint: "/sys/fs/cgroup"},
			},
			want: memoryController{path: "/pod/abc", isV2: true},
		},
		{
			name: "hybrid_prefers_v1_memory_entry",
			chs: []cgroupHierarchy{
				{HierarchyID: "0", ControllerList: "", CgroupPath: "/pod/abc"},
				{HierarchyID: "1", ControllerList: "memory", CgroupPath: "/pod/abc"},
			},
			mis: []mountInfo{
				{FilesystemType: "cgroup2", MountPoint: "/sys/fs/cgroup/unified"},
				{FilesystemType: "cgroup", MountPoint: "/sys/fs/cgroup/memory", SuperOptions: "rw,memory"},
			},
			want: memoryController{path: "/pod/abc"},
		},
		{
			name: "hybrid_v1_mount_without_v1_entry_selects_v2",
			chs: []cgroupHierarchy{
				{HierarchyID: "0", ControllerList: "", CgroupPath: "/pod/abc"},
			},
			mis: []mountInfo{
				{FilesystemType: "cgroup2", MountPoint: "/sys/fs/cgroup/unified"},
				{FilesystemType: "cgroup", MountPoint: "/sys/fs/cgroup/memory", SuperOptions: "rw,memory"},
			},
			want: memoryController{path: "/pod/abc", isV2: true},
		},
		{
			name: "zero_hierarchy_memory_entry_is_not_v1",
			chs: []cgroupHierarchy{
				{HierarchyID: "0", ControllerList: "memory", CgroupPath: "/pod/v1"},
				{HierarchyID: "0", ControllerList: "", CgroupPath: "/pod/v2"},
			},
			mis: []mountInfo{
				{FilesystemType: "cgroup2", MountPoint: "/sys/fs/cgroup"},
			},
			want: memoryController{path: "/pod/v2", isV2: true},
		},
		{
			name: "v1_entry_without_mount_blocks_v2_fallback",
			chs: []cgroupHierarchy{
				{HierarchyID: "0", ControllerList: "", CgroupPath: "/pod/v2"},
				{HierarchyID: "1", ControllerList: "memory", CgroupPath: "/pod/v1"},
			},
			mis: []mountInfo{
				{FilesystemType: "cgroup2", MountPoint: "/sys/fs/cgroup"},
			},
			wantErr: "memory controller found in /proc/self/cgroup but no cgroup v1 memory mount found",
		},
		{
			name: "v2_entry_without_mount_blocks_v1_fallback",
			chs: []cgroupHierarchy{
				{HierarchyID: "0", ControllerList: "", CgroupPath: "/pod/abc"},
			},
			mis: []mountInfo{
				{FilesystemType: "cgroup", MountPoint: "/sys/fs/cgroup/memory", SuperOptions: "rw,memory"},
			},
			wantErr: "cgroup v2 hierarchy found in /proc/self/cgroup but no cgroup2 mount found",
		},
		{
			name: "irrelevant_entries_return_err_no_cgroup",
			chs: []cgroupHierarchy{
				{HierarchyID: "2", ControllerList: "cpu", CgroupPath: "/pod/abc"},
			},
			mis: []mountInfo{
				{FilesystemType: "cgroup", MountPoint: "/sys/fs/cgroup/memory", SuperOptions: "rw,memory"},
			},
			wantErrIs: ErrNoCgroup,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectMemoryController(tt.chs, tt.mis)
			if tt.wantErrIs == nil && tt.wantErr == "" {
				if err != nil {
					t.Fatalf("selectMemoryController() error = %v, want nil", err)
				}
			} else if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("selectMemoryController() error = %v, want %v", err, tt.wantErrIs)
				}
			} else if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("selectMemoryController() error = %v, want error %q", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("selectMemoryController() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

type fileSpec struct {
	path    string
	content string
}

type mountSpec struct {
	root           string
	filesystemType string
	superOptions   string
	files          []fileSpec
}

func buildMountInfos(t *testing.T, mounts []mountSpec) []mountInfo {
	t.Helper()

	mis := make([]mountInfo, 0, len(mounts))
	for _, mount := range mounts {
		mountPoint := t.TempDir()
		for _, file := range mount.files {
			path := filepath.Join(mountPoint, file.path)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("MkdirAll(%s) error = %v, want nil", file.path, err)
			}

			if err := os.WriteFile(path, []byte(file.content), 0o644); err != nil {
				t.Fatalf("WriteFile(%s) error = %v, want nil", file.path, err)
			}
		}

		mis = append(mis, mountInfo{
			Root:           mount.root,
			MountPoint:     mountPoint,
			FilesystemType: mount.filesystemType,
			SuperOptions:   mount.superOptions,
		})
	}

	return mis
}

func TestGetMemoryLimitV2FromControllerPath(t *testing.T) {
	tests := []struct {
		name      string
		relPath   string
		mounts    []mountSpec
		want      uint64
		wantErrIs error
		wantErr   string
	}{
		{
			name:    "returns_limit_from_valid_candidate",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "pod/memory.max", content: "4096\n"},
						{path: "pod/abc/memory.max", content: "2048\n"},
					},
				},
			},
			want: 2048,
		},
		{
			name:    "mount_closest_to_root_wins",
			relPath: "/org/tenant/app",
			mounts: []mountSpec{
				{
					root:           "/org/tenant",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "memory.max", content: "2048\n"},
						{path: "app/memory.max", content: "4096\n"},
					},
				},
				{
					root:           "/",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "org/memory.max", content: "1024\n"},
						{path: "org/tenant/memory.max", content: "2048\n"},
						{path: "org/tenant/app/memory.max", content: "4096\n"},
					},
				},
			},
			want: 1024,
		},
		{
			name:    "parent_limit_smaller_than_leaf_wins",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "pod/memory.max", content: "2048\n"},
						{path: "pod/abc/memory.max", content: "4096\n"},
					},
				},
			},
			want: 2048,
		},
		{
			name:    "returns_err_no_limit_for_explicit_max",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "pod/abc/memory.max", content: "max\n"},
					},
				},
			},
			wantErrIs: ErrNoLimit,
		},
		{
			name:    "leaf_max_uses_parent_concrete_limit",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "pod/memory.max", content: "536870912\n"},
						{path: "pod/abc/memory.max", content: "max\n"},
					},
				},
			},
			want: 536870912,
		},
		{
			name:    "missing_leaf_file_uses_parent_limit",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "pod/memory.max", content: "4096\n"},
						{path: "pod/abc/cgroup.procs", content: ""},
					},
				},
			},
			want: 4096,
		},
		{
			name:    "missing_memory_max_returns_err_no_limit",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "pod/abc/cgroup.procs", content: ""},
					},
				},
			},
			wantErrIs: ErrNoLimit,
		},
		{
			name:    "concrete_limit_wins_over_no_limit",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "pod/abc/memory.max", content: "max\n"},
					},
				},
				{
					root:           "/",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "pod/abc/memory.max", content: "1024\n"},
					},
				},
			},
			want: 1024,
		},
		{
			name:    "parse_error_blocks_concrete_limits",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "pod/abc/memory.max", content: "1024\n"},
					},
				},
				{
					root:           "/",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "pod/abc/memory.max", content: "2048\n"},
					},
				},
				{
					root:           "/",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "pod/abc/memory.max", content: "invalid\n"},
					},
				},
			},
			wantErr: "failed to parse memory.max value: strconv.ParseUint: parsing \"invalid\": invalid syntax",
		},
		{
			name:    "conflicting_concrete_limits_return_error",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "pod/abc/memory.max", content: "1024\n"},
					},
				},
				{
					root:           "/",
					filesystemType: "cgroup2",
					files: []fileSpec{
						{path: "pod/abc/memory.max", content: "2048\n"},
					},
				},
			},
			wantErr: "conflicting memory limits from cgroup mount candidates: 1024 and 2048",
		},
		{
			name:    "all_skipped_candidates_return_error",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/other",
					filesystemType: "cgroup2",
				},
			},
			wantErr: "no usable cgroup v2 memory mount found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getMemoryLimitV2FromControllerPath(tt.relPath, buildMountInfos(t, tt.mounts))
			if tt.wantErrIs == nil && tt.wantErr == "" {
				if err != nil {
					t.Fatalf("getMemoryLimitV2FromControllerPath() error = %v, want nil", err)
				}
			} else if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("getMemoryLimitV2FromControllerPath() error = %v, want %v", err, tt.wantErrIs)
				}
			} else if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("getMemoryLimitV2FromControllerPath() error = %v, want error %q", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("getMemoryLimitV2FromControllerPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMemoryLimitV1FromControllerPath(t *testing.T) {
	noLimit32 := strconv.FormatUint(uint64(math.MaxInt32)*uint64(os.Getpagesize()), 10)

	tests := []struct {
		name      string
		relPath   string
		mounts    []mountSpec
		want      uint64
		wantErrIs error
		wantErr   string
	}{
		{
			name:    "returns_limit_from_valid_candidate",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.stat", content: "hierarchical_memory_limit 2048\n"},
						{path: "pod/abc/memory.limit_in_bytes", content: "4096\n"},
					},
				},
			},
			want: 2048,
		},
		{
			name:    "hierarchical_32bit_sentinel_uses_limit",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.stat", content: "hierarchical_memory_limit " + noLimit32 + "\n"},
						{path: "pod/abc/memory.limit_in_bytes", content: "4096\n"},
					},
				},
			},
			want: 4096,
		},
		{
			name:    "limit_32bit_sentinel_uses_hierarchical",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.stat", content: "hierarchical_memory_limit 4096\n"},
						{path: "pod/abc/memory.limit_in_bytes", content: noLimit32 + "\n"},
					},
				},
			},
			want: 4096,
		},
		{
			name:    "memory_stat_only_returns_hierarchical_limit",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.stat", content: "hierarchical_memory_limit 2048\n"},
					},
				},
			},
			want: 2048,
		},
		{
			name:    "memory_limit_in_bytes_only_returns_limit",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.limit_in_bytes", content: "4096\n"},
					},
				},
			},
			want: 4096,
		},
		{
			name:    "stat_without_target_uses_limit",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.stat", content: "cache 1\n"},
						{path: "pod/abc/memory.limit_in_bytes", content: "4096\n"},
					},
				},
			},
			want: 4096,
		},
		{
			name:    "missing_v1_limit_files_is_skipped",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.stat", content: "cache 1\n"},
					},
				},
			},
			wantErr: "no usable cgroup v1 memory mount found",
		},
		{
			name:    "incomplete_mount_is_skipped",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.stat", content: "cache 1\n"},
					},
				},
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.limit_in_bytes", content: "4096\n"},
					},
				},
			},
			want: 4096,
		},
		{
			name:    "malformed_unrelated_stat_line_is_ignored",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.stat", content: "cache\nhierarchical_memory_limit 2048\n"},
						{path: "pod/abc/memory.limit_in_bytes", content: "4096\n"},
					},
				},
			},
			want: 2048,
		},
		{
			name:    "zero_memory_limit_in_bytes_is_not_ignored",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.stat", content: "hierarchical_memory_limit 4096\n"},
						{path: "pod/abc/memory.limit_in_bytes", content: "0\n"},
					},
				},
			},
			want: 0,
		},
		{
			name:    "zero_hierarchical_limit_is_not_ignored",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.stat", content: "hierarchical_memory_limit 0\n"},
					},
				},
			},
			want: 0,
		},
		{
			name:    "returns_err_no_limit_for_explicit_unlimited",
			relPath: "/",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "memory.limit_in_bytes", content: "9223372036854771712\n"},
					},
				},
			},
			wantErrIs: ErrNoLimit,
		},
		{
			name:    "hierarchical_limit_missing_value_error",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.stat", content: "hierarchical_memory_limit\n"},
						{path: "pod/abc/memory.limit_in_bytes", content: "4096\n"},
					},
				},
			},
			wantErr: "failed to read hierarchical_memory_limit: failed to parse memory.stat \"hierarchical_memory_limit\": not enough fields",
		},
		{
			name:    "hierarchical_memory_limit_parse_error",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.stat", content: "cache\nhierarchical_memory_limit invalid\n"},
						{path: "pod/abc/memory.limit_in_bytes", content: "4096\n"},
					},
				},
			},
			wantErr: "failed to read hierarchical_memory_limit: failed to parse hierarchical_memory_limit value: strconv.ParseUint: parsing \"invalid\": invalid syntax",
		},
		{
			name:    "memory_limit_in_bytes_parse_error",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.limit_in_bytes", content: "invalid\n"},
					},
				},
			},
			wantErr: "failed to parse memory.limit_in_bytes value: strconv.ParseUint: parsing \"invalid\": invalid syntax",
		},
		{
			name:    "conflicting_concrete_limits_return_error",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.stat", content: "hierarchical_memory_limit 1024\n"},
						{path: "pod/abc/memory.limit_in_bytes", content: "1024\n"},
					},
				},
				{
					root:           "/",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
					files: []fileSpec{
						{path: "pod/abc/memory.stat", content: "hierarchical_memory_limit 2048\n"},
						{path: "pod/abc/memory.limit_in_bytes", content: "2048\n"},
					},
				},
			},
			wantErr: "conflicting memory limits from cgroup mount candidates: 1024 and 2048",
		},
		{
			name:    "all_skipped_candidates_return_error",
			relPath: "/pod/abc",
			mounts: []mountSpec{
				{
					root:           "/other",
					filesystemType: "cgroup",
					superOptions:   "rw,memory",
				},
			},
			wantErr: "no usable cgroup v1 memory mount found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getMemoryLimitV1FromControllerPath(tt.relPath, buildMountInfos(t, tt.mounts))
			if tt.wantErrIs == nil && tt.wantErr == "" {
				if err != nil {
					t.Fatalf("getMemoryLimitV1FromControllerPath() error = %v, want nil", err)
				}
			} else if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("getMemoryLimitV1FromControllerPath() error = %v, want %v", err, tt.wantErrIs)
				}
			} else if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("getMemoryLimitV1FromControllerPath() error = %v, want error %q", err, tt.wantErr)
			}

			if got != tt.want {
				t.Fatalf("getMemoryLimitV1FromControllerPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
