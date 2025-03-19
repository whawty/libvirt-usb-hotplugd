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
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/Emposat/usb"
	"github.com/antchfx/xmlquery"
	// "github.com/citilinkru/libudev"
)

const (
	hostdevXMLTemplateText = `
    <hostdev mode='subsystem' type='usb' managed='yes'>
      <source startupPolicy='optional'>
        <vendor id='{{ printf "0x%04x" .VendorID }}' />
        <product id='{{ printf "0x%04x" .ProductID }}' />
        <address bus='{{ printf "%d" .Bus }}' device='{{ printf "%d" .Device }}' />
      </source>
    </hostdev>
`
)

var (
	hostdevXMLTemplate = template.Must(template.New("attach-device-xml").Parse(hostdevXMLTemplateText))
)

type Device struct {
	VendorID  uint16
	ProductID uint16
	Bus       int
	Device    int

	libusb *usb.Device
	// TODO: add udev attributes
}

func NewDeviceFromLibUSB(libusb *usb.Device) (d Device) {
	d.libusb = libusb
	d.VendorID = libusb.Vendor.ID
	d.ProductID = libusb.Product.ID
	d.Bus = libusb.Bus
	d.Device = libusb.Device
	return
}

func uint16From0xString(str string) (uint16, error) {
	val, err := strconv.ParseUint(strings.TrimPrefix(str, "0x"), 16, 16)
	return uint16(val), err
}

func intFromString(str string) (int, error) {
	val, err := strconv.ParseInt(str, 10, 32)
	return int(val), err
}

func NewDeviceFromLibVirtHostdev(hostdev *xmlquery.Node) (d Device, err error) {
	src := hostdev.SelectElement("source")
	if src == nil {
		err = fmt.Errorf("hostdev has no 'source' element")
		return
	}
	vendor := src.SelectElement("vendor")
	if vendor == nil {
		err = fmt.Errorf("hostdev source has no 'vendor' element")
		return
	}
	product := src.SelectElement("product")
	if vendor == nil {
		err = fmt.Errorf("hostdev source has no 'product' element")
		return
	}
	addr := src.SelectElement("address")
	if addr == nil {
		err = fmt.Errorf("hostdev source has no 'address' element")
		return
	}

	if d.VendorID, err = uint16From0xString(vendor.SelectAttr("id")); err != nil {
		return
	}
	if d.ProductID, err = uint16From0xString(product.SelectAttr("id")); err != nil {
		return
	}
	if d.Bus, err = intFromString(addr.SelectAttr("bus")); err != nil {
		return
	}
	if d.Device, err = intFromString(addr.SelectAttr("device")); err != nil {
		return
	}
	return
}

func (d *Device) String() string {
	names := ""
	if d.libusb != nil {
		names = " " + d.libusb.Vendor.Name() + " " + d.libusb.Product.Name()
	}
	return fmt.Sprintf("Bus %03d Device %03d: %04x:%04x%s", d.Bus, d.Device, d.VendorID, d.ProductID, names)
}

func (d *Device) Slug() string {
	return fmt.Sprintf("%03d/%03d %04x:%04x", d.Bus, d.Device, d.VendorID, d.ProductID)
}

func (d *Device) HostDevXML() (string, error) {
	var buf strings.Builder
	if err := hostdevXMLTemplate.Execute(&buf, d); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (d *Device) Matches(matcher DeviceMatcher) bool {
	if matcher.Bus != nil && *matcher.Bus != d.Bus {
		return false
	}
	if matcher.Device != nil && *matcher.Device != d.Device {
		return false
	}
	if matcher.VendorID != nil && *matcher.VendorID != d.VendorID {
		return false
	}
	if matcher.ProductID != nil && *matcher.ProductID != d.ProductID {
		return false
	}
	return true
}
