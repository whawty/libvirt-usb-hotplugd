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
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"strings"
	"text/template"

	"github.com/Emposat/usb"
)

const (
	hostdevXMLTemplateText = `
    <hostdev mode='subsystem' type='usb' managed='yes'>
      <source startupPolicy='optional'>
        <vendor id='{{ printf "0x%04x" .Handle.Vendor.ID }}' />
        <product id='{{ printf "0x%04x" .Handle.Product.ID }}' />
        <address bus='{{ printf "%d" .Handle.Bus }}' device='{{ printf "%d" .Handle.Device }}' />
      </source>
      <alias name='whawty-{{ .Digest }}'/>
    </hostdev>
`
)

var (
	hostdevXMLTemplate = template.Must(template.New("attach-device-xml").Parse(hostdevXMLTemplateText))
)

type Device struct {
	Handle *usb.Device
}

func (d *Device) Digest() string {
	hash := fnv.New128a()
	slug := fmt.Sprintf("%03d/%03d: %04x:%04x %s %s", d.Handle.Bus, d.Handle.Device, d.Handle.Vendor.ID, d.Handle.Product.ID, d.Handle.Vendor.Name(), d.Handle.Product.Name())
	_, _ = hash.Write([]byte(slug)) // hash.Write() never returns an error
	return base64.RawURLEncoding.EncodeToString(hash.Sum(nil))

}

func (d *Device) HostDevXML() (string, error) {
	var buf strings.Builder
	if err := hostdevXMLTemplate.Execute(&buf, d); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (d *Device) Matches(matcher DeviceMatcher) bool {
	if matcher.Bus != nil && *matcher.Bus != d.Handle.Bus {
		return false
	}
	if matcher.Device != nil && *matcher.Device != d.Handle.Device {
		return false
	}
	if matcher.VendorID != nil && *matcher.VendorID != d.Handle.Vendor.ID {
		return false
	}
	if matcher.ProductID != nil && *matcher.ProductID != d.Handle.Product.ID {
		return false
	}
	if matcher.VendorName != nil && *matcher.VendorName != d.Handle.Vendor.Name() {
		return false
	}
	if matcher.ProductName != nil && *matcher.ProductName != d.Handle.Product.Name() {
		return false
	}
	return true
}

func ListUSBDevices() ([]*Device, error) {
	devices, err := usb.List()
	if err != nil {
		return nil, err
	}

	result := make([]*Device, 0, len(devices))
	for _, device := range devices {
		result = append(result, &Device{Handle: device})
	}
	return result, nil
}
