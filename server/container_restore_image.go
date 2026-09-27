// SPDX-License-Identifier: Apache-2.0
package server

import types "k8s.io/cri-api/pkg/apis/runtime/v1"

// Keep the checkpoint's rootfs identity and the current request's policy identity
// separate. Never use an archive path or checkpoint metadata as the policy name.
func restoreRootFSImage(rootFSImage string, requested *types.ImageSpec) *types.ImageSpec {
	return &types.ImageSpec{
		Image:              rootFSImage,
		UserSpecifiedImage: requested.GetUserSpecifiedImage(),
	}
}
