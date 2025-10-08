package memlimit

import (
	"reflect"
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
			name:  "valid line with optional field",
			input: "36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3 /dev/root rw,errors=continue",
			want: mountInfo{
				Root:           "/mnt1",
				MountPoint:     "/mnt2",
				FilesystemType: "ext3",
				SuperOptions:   "rw,errors=continue",
			},
		},
		{
			name:  "valid line without optional field",
			input: "731 771 0:59 /sysrq-trigger /proc/sysrq-trigger ro,nosuid,nodev,noexec,relatime - proc proc rw",
			want: mountInfo{
				Root:           "/sysrq-trigger",
				MountPoint:     "/proc/sysrq-trigger",
				FilesystemType: "proc",
				SuperOptions:   "rw",
			},
		},
		{
			name:  "valid line with minimal fields (no optional fields)",
			input: "25 1 0:22 / /dev rw - devtmpfs udev rw",
			want: mountInfo{
				Root:           "/",
				MountPoint:     "/dev",
				FilesystemType: "devtmpfs",
				SuperOptions:   "rw",
			},
		},
		{
			name:    "no separator",
			input:   "36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 ext3 /dev/root rw,errors=continue",
			wantErr: `invalid separator`,
		},
		{
			name:    "not enough fields on left side",
			input:   "36 35 98:0 /mnt1 /mnt2 - ext3 /dev/root rw,errors=continue",
			wantErr: `not enough fields before separator: [36 35 98:0 /mnt1 /mnt2]`,
		},
		{
			name:    "not enough fields on right side",
			input:   "36 35 98:0 /mnt1 /mnt2 rw,noatime master:1 - ext3",
			wantErr: `not enough fields after separator: [ext3]`,
		},
		{
			name:    "empty line",
			input:   "",
			wantErr: `empty line`,
		},
		{
			name:  "6 fields on left side (no optional field), should add empty optional field",
			input: "100 1 8:2 / /data rw - ext4 /dev/sda2 rw,relatime",
			want: mountInfo{
				Root:           "/",
				MountPoint:     "/data",
				FilesystemType: "ext4",
				SuperOptions:   "rw,relatime",
			},
		},
		{
			name:  "multiple optional fields on left side (issue #26)",
			input: "465 34 253:0 / / rw,relatime shared:409 master:1 - xfs /dev/mapper/fedora-root rw,seclabel,attr2,inode64,logbufs=8,logbsize=32k,noquota",
			want: mountInfo{
				Root:           "/",
				MountPoint:     "/",
				FilesystemType: "xfs",
				SuperOptions:   "rw,seclabel,attr2,inode64,logbufs=8,logbsize=32k,noquota",
			},
		},
		{
			name:  "super options have spaces (issue #28)",
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
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %+v, got %+v", tt.want, got)
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
			name:  "valid line with multiple controllers",
			input: "5:cpuacct,cpu,cpuset:/daemons",
			want: cgroupHierarchy{
				HierarchyID:    "5",
				ControllerList: "cpuacct,cpu,cpuset",
				CgroupPath:     "/daemons",
			},
		},
		{
			name:  "valid line with no controllers (cgroup v2)",
			input: "0::/system.slice/docker.service",
			want: cgroupHierarchy{
				HierarchyID:    "0",
				ControllerList: "",
				CgroupPath:     "/system.slice/docker.service",
			},
		},
		{
			name:    "invalid line - only two fields",
			input:   "5:cpuacct,cpu,cpuset",
			wantErr: "not enough fields: [5 cpuacct,cpu,cpuset]",
		},
		{
			name:    "invalid line - too many fields",
			input:   "5:cpuacct,cpu:cpuset:/daemons:extra",
			wantErr: "too many fields: [5 cpuacct,cpu cpuset /daemons extra]",
		},
		{
			name:    "empty line",
			input:   "",
			wantErr: "empty line",
		},
		{
			name:  "line with empty controller list but valid fields",
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
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestResolveCgroupPath(t *testing.T) {
	tests := []struct {
		name          string
		mountpoint    string
		root          string
		cgroupRelPath string
		want          string
		wantErr       string
	}{
		{
			name:          "exact match with both root and cgroupRelPath as '/'",
			mountpoint:    "/fake/mount",
			root:          "/",
			cgroupRelPath: "/",
			want:          "/fake/mount",
		},
		{
			name:          "exact match with a non-root path",
			mountpoint:    "/fake/mount",
			root:          "/container0",
			cgroupRelPath: "/container0",
			want:          "/fake/mount",
		},
		{
			name:          "valid subpath under root",
			mountpoint:    "/fake/mount",
			root:          "/container0",
			cgroupRelPath: "/container0/group1",
			want:          "/fake/mount/group1",
		},
		{
			name:          "invalid cgroup path outside root",
			mountpoint:    "/fake/mount",
			root:          "/container0",
			cgroupRelPath: "/other_container",
			wantErr:       "invalid cgroup path: /other_container is not under root /container0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveCgroupPath(tt.mountpoint, tt.root, tt.cgroupRelPath)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("expected path %q, got %q", tt.want, got)
			}
		})
	}
}
