package oci

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func tcpInventory(payload []byte) []byte {
	header := make([]byte, 8)
	binary.LittleEndian.PutUint32(header, 0x58313116)
	binary.LittleEndian.PutUint32(header[4:], uint32(len(payload)))
	return append(header, payload...)
}

func TestInventoryTCPNeedsClose(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
		fail bool
	}{
		{"required", tcpInventory([]byte{8, 2, 80, 1}), true, false},
		{"false", tcpInventory([]byte{8, 2, 80, 0}), false, false},
		{"absent", tcpInventory([]byte{8, 2}), false, false},
		{"legacy", tcpInventory([]byte{8, 1, 80, 1}), true, false},
		{"unknown fields", tcpInventory([]byte{8, 2, 26, 2, 8, 1, 80, 1}), true, false},
		{"duplicate last wins", tcpInventory([]byte{8, 2, 80, 1, 80, 0}), false, false},
		{"empty", nil, false, true},
		{"bad magic", make([]byte, 8), false, true},
		{"truncated record", tcpInventory([]byte{8, 2})[:9], false, true},
		{"trailing record", append(tcpInventory([]byte{8, 2}), 0), false, true},
		{"missing version", tcpInventory([]byte{80, 1}), false, true},
		{"unknown version", tcpInventory([]byte{8, 3}), false, true},
		{"wrong type", tcpInventory([]byte{8, 2, 82, 0}), false, true},
		{"truncated varint", tcpInventory([]byte{8, 2, 80, 128}), false, true},
		{"invalid tag", tcpInventory([]byte{8, 2, 0}), false, true},
		{"bad unknown field", tcpInventory([]byte{8, 2, 26, 10}), false, true},
		{"oversize", make([]byte, maxTCPInventorySize+1), false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := inventoryTCPNeedsClose(tt.data)
			if (err != nil) != tt.fail || got != tt.want {
				t.Fatalf("got %v, %v; want %v, error=%v", got, err, tt.want, tt.fail)
			}
		})
	}
}

func TestCheckpointTCPRestoreArgs(t *testing.T) {
	dir := t.TempDir()
	if _, err := checkpointTCPRestoreArgs(dir); err == nil {
		t.Fatal("missing inventory accepted")
	}
	for _, enabled := range []byte{0, 1} {
		if err := os.WriteFile(filepath.Join(dir, "inventory.img"), tcpInventory([]byte{8, 2, 80, enabled}), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := checkpointTCPRestoreArgs(dir)
		var want []string
		if enabled == 1 {
			want = []string{"--runtime-opt", "--tcp-close"}
		}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, %v; want %v", got, err, want)
		}
	}
}

func FuzzInventoryTCPNeedsClose(f *testing.F) {
	f.Add(tcpInventory([]byte{8, 2, 80, 1}))
	f.Add(tcpInventory([]byte{8, 2}))
	f.Fuzz(func(t *testing.T, data []byte) {
		closeTCP, err := inventoryTCPNeedsClose(data)
		if err != nil && closeTCP {
			t.Fatal("invalid inventory enabled tcp-close")
		}
	})
}
