//go:build linux

package main

import "github.com/sstallion/go-hid"

// linuxDevice usa a biblioteca go-hid (hidapi via cgo) para falar com o cooler.
type linuxDevice struct {
	dev *hid.Device
}

func openDevice(vid, pid uint16) (Device, error) {
	d, err := hid.Open(vid, pid, "")
	if err != nil {
		return nil, err
	}
	return &linuxDevice{dev: d}, nil
}

func (d *linuxDevice) WriteTemp(temp byte) error {
	// Report HID: byte 0 = report ID (0x00), byte 1 = temperatura.
	_, err := d.dev.Write([]byte{0x00, temp})
	return err
}

func (d *linuxDevice) Close() error {
	return d.dev.Close()
}
