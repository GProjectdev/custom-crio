package server

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	spec "github.com/opencontainers/runtime-spec/specs-go"
)

const serviceAccountMount = "/var/run/secrets/kubernetes.io/serviceaccount"
const canonicalServiceAccountMount = "/run/secrets/kubernetes.io/serviceaccount"

// CRIU records the resolved mountpoint, while CRI can retain /var/run.
// Resolve only this known alias against the target rootfs, never the host root.
func normalizeRestoreServiceAccountMount(rootfs string, mounts []spec.Mount) error {
	index := -1
	for i := range mounts {
		if mounts[i].Destination == serviceAccountMount {
			if index != -1 {
				return fmt.Errorf("duplicate serviceaccount mount")
			}
			index = i
		}
	}
	if index == -1 {
		return nil
	}
	if !filepath.IsAbs(rootfs) {
		return fmt.Errorf("target rootfs must be absolute")
	}
	resolved, err := securejoin.SecureJoin(rootfs, serviceAccountMount)
	if err != nil {
		return fmt.Errorf("resolve serviceaccount destination: %w", err)
	}
	if resolved != filepath.Join(rootfs, filepath.FromSlash(canonicalServiceAccountMount)) {
		return nil
	}
	for i, mount := range mounts {
		if i == index {
			continue
		}
		dest := path.Clean(mount.Destination)
		// Parent mounts can hide the rootfs symlink; do not guess their contents.
		if dest == canonicalServiceAccountMount || dest == "/" ||
			strings.HasPrefix(serviceAccountMount, dest+"/") ||
			strings.HasPrefix(canonicalServiceAccountMount, dest+"/") {
			return fmt.Errorf("mount %q overlaps serviceaccount alias", mount.Destination)
		}
	}
	// Keep the current Pod's source, read-only flags and ID mappings untouched.
	mounts[index].Destination = canonicalServiceAccountMount
	return nil
}
