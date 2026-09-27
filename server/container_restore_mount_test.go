package server

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	spec "github.com/opencontainers/runtime-spec/specs-go"
)

func TestRestoreServiceAccountNoAlias(t *testing.T) {
	root := t.TempDir()
	mounts := []spec.Mount{{Destination: serviceAccountMount, Source: "/current-pod/token", Options: []string{"rbind", "ro"}}}
	want := append([]spec.Mount(nil), mounts...)
	if err := normalizeRestoreServiceAccountMount(root, mounts); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(mounts, want) {
		t.Fatal("changed mount without a rootfs alias")
	}
	if err := normalizeRestoreServiceAccountMount("", mounts); err == nil {
		t.Fatal("empty rootfs accepted")
	}
	if err := normalizeRestoreServiceAccountMount(root, append(mounts, mounts[0])); err == nil {
		t.Fatal("duplicate destination accepted")
	}
}

func TestRestoreServiceAccountAlias(t *testing.T) {
	for _, target := range []string{"../run", "/run"} {
		t.Run(target, func(t *testing.T) {
			root := t.TempDir()
			for _, dir := range []string{"var", "run"} {
				if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(target, filepath.Join(root, "var", "run")); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			original := spec.Mount{Destination: serviceAccountMount, Type: "bind", Source: "/current-pod/token", Options: []string{"rbind", "rprivate", "ro"}}
			mounts := []spec.Mount{original}
			if err := normalizeRestoreServiceAccountMount(root, mounts); err != nil {
				t.Fatal(err)
			}
			want := original
			want.Destination = canonicalServiceAccountMount
			if !reflect.DeepEqual(mounts[0], want) {
				t.Fatalf("got %#v; want %#v", mounts[0], want)
			}
			for _, dest := range []string{"/run", "/var", "/var/run", canonicalServiceAccountMount} {
				mounts = []spec.Mount{original, {Destination: dest}}
				if err := normalizeRestoreServiceAccountMount(root, mounts); err == nil {
					t.Fatalf("overlap %s accepted", dest)
				}
				if mounts[0].Destination != serviceAccountMount {
					t.Fatal("modified mount on error")
				}
			}
		})
	}
}
