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
	"net/url"
	"os"

	"github.com/Emposat/usb"
	"github.com/digitalocean/go-libvirt"
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
	devices, err := usb.List()
	if err != nil {
		wl.Fatalf("failed to list usb devices: %v", err)
	}
	for _, d := range devices {
		wl.Printf("Bus %03d Device %03d: ID %04x:%04x %s %s\n", d.Bus, d.Device, d.Vendor.ID, d.Product.ID, d.Vendor.Name(), d.Product.Name())
	}

	// list running virtual machines
	uri, _ := url.Parse(string(libvirt.QEMUSystem))
	l, err := libvirt.ConnectToURI(uri)
	if err != nil {
		wl.Fatalf("failed to connect: %v", err)
	}

	v, err := l.ConnectGetLibVersion()
	if err != nil {
		wl.Fatalf("failed to retrieve libvirt version: %v", err)
	}
	wl.Println("Libvirt-Version:", v)

	flags := libvirt.ConnectListDomainsRunning
	domains, _, err := l.ConnectListAllDomains(1, flags)
	if err != nil {
		wl.Fatalf("failed to retrieve domains: %v", err)
	}

	wl.Println("ID\tName\t\tUUID")
	wl.Printf("--------------------------------------------------------\n")
	for _, d := range domains {
		wl.Printf("%d\t%s\t%x\n", d.ID, d.Name, d.UUID)
	}

	if err = l.Disconnect(); err != nil {
		wl.Fatalf("failed to disconnect: %v", err)
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
