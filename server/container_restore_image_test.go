// SPDX-License-Identifier: Apache-2.0
package server

import (
	"testing"

	types "k8s.io/cri-api/pkg/apis/runtime/v1"
)

func TestRestoreRootFSImage(t *testing.T) {
	const rootFS = "sha256:checkpoint-rootfs"
	for _, tc := range []struct {
		name    string
		request *types.ImageSpec
		want    string
	}{
		{"annotation archive", &types.ImageSpec{Image: "/var/lib/kubelet/checkpoints/test.tar", UserSpecifiedImage: "docker.io/pytorch/pytorch:2.4.1"}, "docker.io/pytorch/pytorch:2.4.1"},
		{"digest reference", &types.ImageSpec{Image: "/archive.tar", UserSpecifiedImage: "registry.example/train@sha256:123"}, "registry.example/train@sha256:123"},
		{"missing identity stays missing", &types.ImageSpec{Image: "/archive.tar"}, ""},
		{"nil request", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := restoreRootFSImage(rootFS, tc.request)
			if got.GetImage() != rootFS || got.GetUserSpecifiedImage() != tc.want {
				t.Fatalf("unexpected restored image: %v", got)
			}
			if tc.request != nil && (got == tc.request || tc.request.Image == rootFS) {
				t.Fatal("restore modified the original CRI request")
			}
		})
	}
}
