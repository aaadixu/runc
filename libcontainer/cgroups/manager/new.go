package manager

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	"github.com/opencontainers/runc/libcontainer/cgroups/systemd"
	"github.com/opencontainers/runc/libcontainer/configs"
)

// New returns the instance of a cgroup manager, which is chosen
// based on the local environment (whether cgroup v1 or v2 is used)
// and the config (whether config.Systemd is set or not).
func New(config *configs.Cgroup) (cgroups.Manager, error) {
	return NewWithPaths(config, nil)
}

// NewWithPaths is similar to New, and can be used in case cgroup paths
// are already well known, which can save some resources.
//
// For cgroup v1, the keys are controller/subsystem name, and the values
// are absolute filesystem paths to the appropriate cgroups.
//
// For cgroup v2, the only key allowed is "" (empty string), and the value
// is the unified cgroup path.
func NewWithPaths(config *configs.Cgroup, paths map[string]string) (cgroups.Manager, error) {
	if config == nil {
		return nil, errors.New("cgroups/manager.New: config must not be nil")
	}
	if config.Systemd && !systemd.IsRunningSystemd() {
		return nil, errors.New("systemd not running on this host, cannot use systemd cgroups manager")
	}

	// Cgroup v2 aka unified hierarchy.
	if cgroups.IsCgroup2UnifiedMode() {
		path, err := getUnifiedPath(paths)
		if err != nil {
			return nil, fmt.Errorf("manager.NewWithPaths: inconsistent paths: %w", err)
		}
		if config.Systemd {
			// 如果 config.Systemd 为 true，创建并返回一个新的 systemd 统一管理器。
			// 使用 systemd 作为 cgroup 管理器。
			return systemd.NewUnifiedManager(config, path)
		}
		// 否则，创建并返回一个新的 fs2 管理器。
		// 使用直接操作文件系统的方式来管理 cgroups。
		// 不依赖 systemd，可以在没有 systemd 的环境下使用。
		return fs2.NewManager(config, path)
	}

	// Cgroup v1.
	if config.Systemd {
		return systemd.NewLegacyManager(config, paths)
	}
	// 建并返回一个新的 fs 管理器。
	return fs.NewManager(config, paths)
}

// getUnifiedPath is an implementation detail of libcontainer.
// Historically, libcontainer.Create saves cgroup paths as per-subsystem path
// map (as returned by cm.GetPaths(""), but with v2 we only have one single
// unified path (with "" as a key).
//
// This function converts from that map to string (using "" as a key),
// and also checks that the map itself is sane.
func getUnifiedPath(paths map[string]string) (string, error) {
	if len(paths) > 1 {
		return "", fmt.Errorf("expected a single path, got %+v", paths)
	}
	path := paths[""]
	// can be empty
	if path != "" {
		if filepath.Clean(path) != path || !filepath.IsAbs(path) {
			return "", fmt.Errorf("invalid path: %q", path)
		}
	}

	return path, nil
}
