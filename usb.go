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
	"slices"

	"github.com/Emposat/usb"
	// "github.com/citilinkru/libudev"
)

func ListUSBDevices() ([]Device, error) {
	devices, err := usb.List()
	if err != nil {
		return nil, err
	}

	result := make([]Device, 0, len(devices))
	for _, device := range devices {
		// TODO: enhance Device with attributes from udev
		result = append(result, NewDeviceFromLibUSB(device))
	}
	return result, nil
}

type DeviceDB struct {
	devices map[string]Device
}

func NewDeviceDB() DeviceDB {
	db := DeviceDB{}
	db.devices = make(map[string]Device)
	return db
}

func (db DeviceDB) Reconcile() error {
	devices, err := ListUSBDevices()
	if err != nil {
		return err
	}

	slugs := make([]string, 0, len(devices))
	for _, device := range devices {
		slug := device.Slug()
		slugs = append(slugs, slug)
		if _, exists := db.devices[slug]; exists {
			continue
		}
		db.devices[slug] = device
	}
	for slug := range db.devices {
		if slices.Contains(slugs, slug) {
			continue
		}
		delete(db.devices, slug)
	}

	return nil
}
