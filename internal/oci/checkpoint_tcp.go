package oci

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/encoding/protowire"
)

const maxTCPInventorySize = 1 << 20

func checkpointTCPRestoreArgs(directory string) ([]string, error) {
	f, err := os.Open(filepath.Join(directory, "inventory.img"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxTCPInventorySize {
		return nil, fmt.Errorf("invalid inventory file type or size")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxTCPInventorySize+1))
	if err != nil {
		return nil, err
	}
	closeTCP, err := inventoryTCPNeedsClose(data)
	if err != nil {
		return nil, err
	}
	if closeTCP {
		return []string{"--runtime-opt", "--tcp-close"}, nil
	}
	return nil, nil
}

// CRIU inventory.img has no common image magic: inventory magic, uint32
// record length, then inventory_entry (images/inventory.proto). Read only
// img_version (field 1) and tcp_close (field 10); skip unknown fields.
func inventoryTCPNeedsClose(data []byte) (bool, error) {
	if len(data) < 8 || len(data) > maxTCPInventorySize {
		return false, fmt.Errorf("invalid inventory length")
	}
	if binary.LittleEndian.Uint32(data[:4]) != 0x58313116 {
		return false, fmt.Errorf("invalid inventory magic")
	}
	if uint64(binary.LittleEndian.Uint32(data[4:8])) != uint64(len(data)-8) {
		return false, fmt.Errorf("invalid inventory record length")
	}
	data = data[8:]
	var version uint64
	var closeTCP bool
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return false, fmt.Errorf("invalid inventory tag: %w", protowire.ParseError(n))
		}
		data = data[n:]
		if num == 1 || num == 10 {
			if typ != protowire.VarintType {
				return false, fmt.Errorf("invalid inventory field %d type", num)
			}
			value, size := protowire.ConsumeVarint(data)
			if size < 0 {
				return false, fmt.Errorf("invalid inventory field %d: %w", num, protowire.ParseError(size))
			}
			if num == 1 {
				version = value
			} else {
				closeTCP = value != 0
			}
			data = data[size:]
			continue
		}
		size := protowire.ConsumeFieldValue(num, typ, data)
		if size < 0 {
			return false, fmt.Errorf("invalid inventory field %d: %w", num, protowire.ParseError(size))
		}
		data = data[size:]
	}
	if version != 1 && version != 2 {
		return false, fmt.Errorf("unsupported inventory image version %d", version)
	}
	return closeTCP, nil
}
