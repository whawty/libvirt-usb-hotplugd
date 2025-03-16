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
	"io"
	"log"
	"os"
)

var (
	wl  = log.New(os.Stdout, "[whawty.libvirt-usb-hotplugd]\t", log.LstdFlags)
	wdl = log.New(io.Discard, "[whawty.libvirt-usb-hotplugd dbg]\t", log.LstdFlags)
)

func init() {
	if _, exists := os.LookupEnv("WHAWTY_LIBVIRT_USB_HOTPLUGD_DEBUG"); exists {
		wdl.SetOutput(os.Stderr)
	}
}

func run(conf *Config) {
	wdl.Printf("got config: %+v", conf)

	// list usb devices
	devices, err := ListUSBDevices()
	if err != nil {
		wl.Fatalf("failed to list usb devices: %v", err)
	}
	for _, d := range devices {
		xml, _ := d.HostDevXML()
		wl.Printf("Bus %03d Device %03d: ID %04x:%04x %s %s\n", d.Handle.Bus, d.Handle.Device, d.Handle.Vendor.ID, d.Handle.Product.ID, d.Handle.Vendor.Name(), d.Handle.Product.Name())
		wl.Printf("Digest('%s'): %s", d.Digest(), xml)
	}

	// list running virtual machines
	machines, err := ListVirtualMachines()
	if err != nil {
		wl.Fatalf("failed to list virtual machines: %v", err)
	}
	for _, m := range machines {
		wl.Printf("VM %s (ID=%d, UUID=%x)\n", m.Domain.Name, m.Domain.ID, m.Domain.UUID)
		for alias, device := range m.Devices {
			wl.Printf(" assigned device '%s': Bus %03d Device %03d: ID %04x:%04x", alias, device.Bus, device.Device, device.VendorID, device.ProductID)
		}
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s <config-file>\n", os.Args[0])
		os.Exit(1)
	}

	conf, err := readConfig(os.Args[1])
	if err != nil {
		fmt.Printf("failed to parse config: %v\n", err)
		os.Exit(1)
	}

	wl.Printf("starting...")
	run(conf)
}
