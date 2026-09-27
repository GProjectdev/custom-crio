# TCP-close restore on the isolated Worker

The bundled crun is statically linked. LD_LIBRARY_PATH cannot replace its
linked libcriu. Opening default.conf alone does not prove the final RPC option
is enabled. This fix reads inventory.img and passes --runtime-opt --tcp-close
to conmon only if that archive requires closed TCP connections. Ordinary
container creation is unchanged; unreadable or malformed inventory fails closed.
Closing TCP does not restore distributed application connections automatically.

## Build on MGMT

Fetch and checkout the exact published fix commit without discarding local edits.
Then run from /root/hybridspot-validation/runtime-src/custom-crio:

```bash
(
  set -euo pipefail
  go test -mod=vendor -v internal/oci/checkpoint_tcp.go internal/oci/checkpoint_tcp_test.go
  make -j2 BUILDTAGS="containers_image_openpgp containers_image_ostree_stub seccomp selinux" binaries
  ./bin/crio --version
  sha256sum bin/crio
)
```

Transfer bin/crio through the existing SSH route to /tmp/crio-tcp on
restore-test-ondemand-01. Verify its SHA256 against MGMT before proceeding.
Keep the test node cordoned. Do not delete the Pod, RestorePlan, PVC or archive.

## Install on the test Worker

This restarts the runtime only on the isolated Worker. The two temporary
LD_LIBRARY_PATH overrides are backed up outside the configuration directories.
The original 10-crio.conf and installed library files are left unchanged.

```bash
(
  set -euo pipefail
  test "$(hostname)" = restore-test-ondemand-01
  sudo systemctl show crio -p ExecStart --value | grep -F 'path=/usr/local/bin/crio '
  chmod +x /tmp/crio-tcp
  /tmp/crio-tcp --version
  sha256sum /tmp/crio-tcp
  BACKUP="/var/backups/crio-tcp-$(date -u +%Y%m%dT%H%M%SZ)"
  sudo mkdir -p "$BACKUP"
  sudo cp -a /usr/local/bin/crio "$BACKUP/crio"
  echo "BACKUP=$BACKUP"
  sudo install -m 0755 /tmp/crio-tcp /usr/local/bin/crio.tcp-new
  sudo systemctl stop kubelet
  sudo systemctl stop crio
  for f in /etc/systemd/system/crio.service.d/90-restore-libcriu.conf \
           /etc/crio/crio.conf.d/99-z-restore-libcriu.conf; do
    if sudo test -f "$f"; then
      sudo mv "$f" "$BACKUP/"
    fi
  done
  sudo systemctl daemon-reload
  sudo mv /usr/local/bin/crio.tcp-new /usr/local/bin/crio
  sudo systemctl start crio
  sudo systemctl is-active --quiet crio
  sudo systemctl start kubelet
  sudo journalctl -u crio --since '2 minutes ago' --no-pager
)
```

On startup failure, stop kubelet and CRI-O, restore the backup binary and any
moved overrides to their original paths, daemon-reload, then start CRI-O and
kubelet. Do not remove container storage. Record the new binary hash: the
packaged runtime.json still describes the original runtime package.

## Verify

Look for `Checkpoint inventory requires --tcp-close for container` and correlate
the container ID with restore-smoke. This proves argument selection, not restore
success. Confirm the previous `Need to set the --tcp-close options` error is gone
in a fresh attempt; inspect the full restore log for any subsequent error.
Require positive native restore evidence, the expected archive binding, and
application progress after resume. Running alone is insufficient.

Focused helper tests can run on Windows; full Linux compilation, static runtime
integration and GPU restore require the isolated Worker test. Do not deploy to
the two training Workers until that verification passes.
