# Custom CRI-O restore validation

## Source and scope

This tree imports lehuannhatrang/leehun-cri-o at
`843bd56d977a86faffb248d9872bd14bff48eb4c` (CRI-O 1.37 development).
It adds the admission-bound annotation restore adapter and CDI-aware import.
Restore also preserves the current CRI request's UserSpecifiedImage for the
existing signature-policy check, while retaining the checkpoint rootfs identity.
It does not disable policy checks or fall back to archive metadata as a policy
name. A missing request identity still fails wherever signature policy requires it.
Upstream copyright/license notices are retained. Upstream workflows are archived
in `.github/upstream-workflows`, not enabled for this experimental repository.

Both patches are already integrated. Do NOT apply annotation-adapter.patch or
cdi-restore.patch again. Operator images, CRIU and CUDA plugins are not rebuilt here.

Only the CURRENT CRI allocation authorizes CDI mounts. Checkpoint-era CDI
annotations are discarded, current CDI devices are preserved, and the existing
OCI CDI injection regenerates mounts, hooks, environment and devices. Ordinary
bind-mount checks remain in place. Missing devices and conflicting mounts fail
closed. Keep the administrator-owned CDI registry stable during restore.

## Build on MGMT (Linux)

Use a NEW directory; preserve the existing runtime-src/leehun-cri-o worktree.
Pin the reviewed custom-crio commit before building. Existing Linux build
dependencies and Go >= the version in go.mod are required (currently 1.26.3).
See upstream install.md for native build dependencies.

```bash
(
  set -euo pipefail
  cd /root/hybridspot-validation/runtime-src
  git clone https://github.com/GProjectdev/custom-crio.git custom-crio
  cd custom-crio
  git rev-parse HEAD
  go test -mod=vendor -v \
    server/container_restore_annotation.go server/container_restore_annotation_test.go \
    server/container_restore_cdi.go server/container_restore_cdi_test.go \
    server/container_restore_image.go server/container_restore_image_test.go
  make -j2 BUILDTAGS="containers_image_openpgp containers_image_ostree_stub seccomp selinux" binaries
  ./bin/crio --version
  sha256sum bin/crio
)
```

For an existing checkout, inspect git status first. Fetch and checkout the
reviewed commit without discarding local modifications. Stop on any build error.
Full Linux compilation is mandatory; isolated helper tests do not prove restore.

## Install ONLY on the isolated test Worker

Keep restore-test-ondemand-01 cordoned. Preserve its RestorePlan, PVC, checkpoint
archive and existing restore-smoke Pod. Do not touch trainer worker-00/worker-01.
Transfer bin/crio via your existing SSH route to /tmp/crio-cdi on the test Worker.
Compare sha256sum /tmp/crio-cdi with the MGMT output before installation.

On the test Worker, verify systemctl show crio -p ExecStart uses
/usr/local/bin/crio. Then run during the test node maintenance window:

```bash
(
  set -euo pipefail
  test "$(hostname)" = restore-test-ondemand-01
  /tmp/crio-cdi --version
  sha256sum /tmp/crio-cdi
  sudo systemctl show crio -p ExecStart
  sudo systemctl show crio -p ExecStart --value | grep -F 'path=/usr/local/bin/crio '
  STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
  sudo cp -a /usr/local/bin/crio "/usr/local/bin/crio.before-cdi-$STAMP"
  sudo install -m 0755 /tmp/crio-cdi /usr/local/bin/crio.cdi-new
  sudo systemctl stop kubelet
  sudo systemctl stop crio
  sudo mv /usr/local/bin/crio.cdi-new /usr/local/bin/crio
  sudo systemctl start crio
  sudo systemctl is-active --quiet crio
  sudo systemctl start kubelet
)
```

If startup fails, keep kubelet stopped, stop CRI-O, restore the recorded backup
binary, then start CRI-O followed by kubelet. Never delete container storage.
Preserve runtime.json; its old package hash becomes stale after this manual
override. Record the new hash and rebuild/version the runtime package before
automatic provisioning.

## Check the retry

### Sandbox containerenv after daemon restart

LoadSandbox now restores the current infra container's .containerenv path before
new workload containers are created. Previously the in-memory path stayed empty
after daemon restart, so /run/.containerenv was absent from the generated OCI
mount list. CRIU archives using that external mount could then fail with
`No mapping for ... mountpoint`. Existing regular marker files are preserved;
creation errors and non-regular marker paths are rejected.

Run the focused helper tests before building:

```bash
go test -mod=vendor -v internal/lib/sandbox/containerenv.go internal/lib/sandbox/containerenv_test.go
# Full package tests require a supported OS and native build dependencies.
go test -mod=vendor -tags 'containers_image_openpgp containers_image_ostree_stub' ./internal/lib/sandbox
```

Rebuild bin/crio and use the isolated-Worker binary replacement procedure above.
Preserve the Pod and archive so the test exercises reloading an existing sandbox.
Verify /run/.containerenv in the new OCI spec points at the current sandbox, not
the source archive's sandbox. Confirm a fresh attempt no longer fails on that
external mount; this alone does not prove full GPU restore success.

For archives requiring TCP-close with the static crun binary, follow the
[TCP-close restore procedure](tcp-close-restore.md).

Kubelet may retry the existing failed container automatically. Inspect fresh
events for the current Pod UID; do not delete the Pod just to replace CRI-O.

```bash
# MGMT
kubectl --kubeconfig="$AWS_KUBECONFIG" -n fluidcr-demo describe pod restore-smoke
kubectl --kubeconfig="$AWS_KUBECONFIG" -n fluidcr-demo get restoreplan restore-smoke-local-01 -o json
# Test Worker
sudo journalctl -u crio --since '10 minutes ago' --no-pager
```

Require positive native checkpoint import/CRIU restore evidence for this container,
the expected archive annotation, and then application checkpoint identity and
advancing globalStep after resume. Running/Ready alone or loading latest.pt in a
new process is NOT native restore proof. Stop on new mount, CRIU, CUDA or driver
errors; do not bypass mount validation.

Validation performed on the publishing workstation: focused annotation/CDI tests
and go vet. The symlink test was skipped because Windows denied symlink creation.
Upstream make lint/format/TOC checks could not run because make is unavailable.
Full Linux build and GPU restore have NOT been validated by those tests.
Distributed NCCL restore and SpotReplacement remain separate end-to-end tests.
