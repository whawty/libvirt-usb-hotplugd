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
	"time"
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

func reconcile(conf *Config, devices map[string]Device, machines map[string]Machine) {
	// attach new devices
	for mname, mconf := range conf.Machines {
		machine, exists := machines[mname]
		if !exists {
			wl.Printf("skipping device matchers for unknown machine: %s", mname)
			continue
		}
		for _, matcher := range mconf.DeviceMatchers {
			for slug, device := range devices {
				if device.Matches(matcher) {
					if _, exists := machine.Devices[slug]; exists {
						wdl.Printf("device '%s' is already attached to machine '%s'", device.String(), mname)
						continue
					}
					err := AttachDeviceToVirtualMachine(machine, device)
					if err != nil {
						wl.Printf("failed to attach device '%s' to machine '%s': %v", device.String(), mname, err)
					} else {
						wl.Printf("sucessfully attached device '%s' to machine '%s'", device.String(), mname)
					}
				}
			}
		}
	}

	// detach stale devices
	for mname, machine := range machines {
		for _, device := range machine.Devices {
			if _, exists := devices[device.Slug()]; !exists {
				err := DetachDeviceFromVirtualMachine(machine, device)
				if err != nil {
					wl.Printf("failed to detach device '%s' from machine '%s': %v", device.String(), mname, err)
				} else {
					wl.Printf("successfully detached device '%s' from machine '%s'", device.String(), mname)
				}
			}
		}
	}
}

func run(conf *Config) {
	wdl.Printf("got config: %+v", conf)

	for {
		// list usb devices
		devices, err := ListUSBDevices()
		if err != nil {
			wl.Printf("failed to list usb devices: %v", err)
		}
		for _, device := range devices {
			wdl.Printf("found Device: %s", device.String())
		}

		// list running virtual machines
		machines, err := ListVirtualMachines()
		if err != nil {
			wl.Printf("failed to list virtual machines: %v", err)
		}
		for _, machine := range machines {
			wdl.Printf("found VM: %s\n", machine.String())
		}

		// attach/detach devices
		reconcile(conf, devices, machines)
		time.Sleep(conf.Interval)
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
