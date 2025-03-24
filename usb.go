//
// Copyright (c) 2025 whawty contributors (see AUTHORS file)
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
//
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
//
// * Neither the name of whawty.libvirt-usb-hotplugd nor the names of its
//   contributors may be used to endorse or promote products derived from
//   this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/Emposat/usb"
	"golang.org/x/sys/unix"
)

const (
	devBusUSBBasePath     = "/dev/bus/usb"
	udevDataBasePath      = "/run/udev/data"
	sysfsDevBlockBasePath = "/sys/dev/block"
	sysfsDevCharBasePath  = "/sys/dev/char"
)

func USBDeviceToSysfsDevicesAndUdevDataPath(dev Device) (string, string, error) {
	devBusUSBPath := filepath.Join(devBusUSBBasePath, fmt.Sprintf("%03d/%03d", dev.Bus, dev.Device))
	info, err := os.Stat(devBusUSBPath)
	if err != nil {
		return "", "", fmt.Errorf("device %s failed to stat(%s): %w", dev.String(), devBusUSBPath, err)
	}
	if info.Mode()&os.ModeDevice == 0 {
		return "", "", fmt.Errorf("%s is not a device file", devBusUSBPath)
	}
	sysfsDevBasePath := sysfsDevCharBasePath
	udevDataNamePrefix := "c"
	if info.Mode()&os.ModeCharDevice == 0 {
		// not sure if this is necessary - even USB-Sticks are character devices at this level
		// but since this check is cheap and easy let's keep it in here.
		sysfsDevBasePath = sysfsDevBlockBasePath
		udevDataNamePrefix = "b"
	}

	stat_t, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		panic("os.Stat() returned unexpected result")
	}
	major := unix.Major(stat_t.Rdev)
	minor := unix.Minor(stat_t.Rdev)

	sysfsDevPath := filepath.Join(sysfsDevBasePath, fmt.Sprintf("%d:%d", major, minor))
	sysfsDevicesPath, err := os.Readlink(sysfsDevPath)
	if err != nil {
		return "", "", fmt.Errorf("could not resolve symlink %s: %w", sysfsDevPath, err)
	}
	if !filepath.IsAbs(sysfsDevicesPath) {
		sysfsDevicesPath = filepath.Join(sysfsDevBasePath, sysfsDevicesPath)
	}

	udevDataPath := filepath.Join(udevDataBasePath, fmt.Sprintf("%s%d:%d", udevDataNamePrefix, major, minor))

	return sysfsDevicesPath, udevDataPath, nil
}

func splitKeyValue(line string) (string, string, error) {
	fields := strings.SplitN(line, "=", 2)
	if len(fields) != 2 {
		return "", "", errors.New("string does not contain '='")
	}
	return fields[0], fields[1], nil
}

func readUeventFile(device *Device, basePath string) error {
	file, err := os.Open(filepath.Join(basePath, "uevent"))
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, err := splitKeyValue(scanner.Text())
		if err != nil {
			// silently ignore invalid lines
			continue
		}
		switch key {
		case "DEVNAME":
			value = filepath.Join("/dev", value)
		}

		device.Udev.Env[key] = value
	}
	return scanner.Err()
}

func readUdevData(device *Device, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.SplitN(scanner.Text(), ":", 2)
		if len(fields) != 2 {
			// silently ignore invalid lines
			continue
		}
		switch fields[0] {
		case "I":
			device.Udev.Env["USEC_INITIALIZED"] = fields[1]
		case "G":
			device.Udev.Tags = append(device.Udev.Tags, fields[1])
		case "Q":
			device.Udev.CurrentTags = append(device.Udev.CurrentTags, fields[1])
		case "E":
			key, value, err := splitKeyValue(fields[1])
			if err == nil {
				device.Udev.Env[key] = value
			}
			// silently ignore invalid lines
		}
	}
	return scanner.Err()
}

func ListUSBDevices() (map[string]Device, error) {
	devices, err := usb.List()
	if err != nil {
		return nil, err
	}

	result := make(map[string]Device)
	for _, device := range devices {
		d := NewDeviceFromLibUSB(device)

		sysfsDevicesPath, udevDataPath, err := USBDeviceToSysfsDevicesAndUdevDataPath(d)
		if err != nil {
			wl.Printf("failed to resolve sysfs and udev paths for %s: %v", d.Slug(), err)
		} else {
			d.Udev.Env["DEVPATH"] = strings.TrimPrefix(sysfsDevicesPath, "/sys")
			d.Udev.Env["SUBSYSTEM"] = "usb"
			if err := readUeventFile(&d, sysfsDevicesPath); err != nil {
				wl.Printf("failed to read udev attributes from uevent file for %s: %v", d.Slug(), err)
			}
			if err := readUdevData(&d, udevDataPath); err != nil {
				wl.Printf("failed to read udev attributes from udev/data file for %s: %v", d.Slug(), err)
			}
		}

		result[d.Slug()] = d
	}
	return result, nil
}
