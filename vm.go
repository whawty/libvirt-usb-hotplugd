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
	"net/url"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/digitalocean/go-libvirt"
)

type Machine struct {
	Domain  libvirt.Domain
	Devices map[string]Device
}

func (m Machine) String() string {
	return fmt.Sprintf("%s (ID=%d, UUID=%x): %d attached devices", m.Domain.Name, m.Domain.ID, m.Domain.UUID, len(m.Devices))
}

func MachineFromLibvirtDomain(l *libvirt.Libvirt, domain libvirt.Domain) (*Machine, error) {
	domxml, err := l.DomainGetXMLDesc(domain, 0)
	if err != nil {
		return nil, err
	}

	domdata, err := xmlquery.Parse(strings.NewReader(domxml))
	if err != nil {
		return nil, err
	}
	hostdevs := xmlquery.Find(domdata, "/domain/devices/hostdev[@type='usb']")

	m := &Machine{Domain: domain}
	m.Devices = make(map[string]Device)
	for _, hostdev := range hostdevs {
		dev, err := NewDeviceFromLibVirtHostdev(hostdev)
		if err != nil {
			return nil, err
		}
		m.Devices[dev.Slug()] = dev
	}
	return m, nil
}

func NewVirshConnection() (*libvirt.Libvirt, error) {
	uri, err := url.Parse(string(libvirt.QEMUSystem))
	if err != nil {
		return nil, err
	}

	l, err := libvirt.ConnectToURI(uri)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func ListActiveVirtualMachines(conf *Config) (map[string]Machine, error) {
	l, err := NewVirshConnection()
	if err != nil {
		return nil, err
	}
	defer l.Disconnect() //nolint:errcheck

	machines := make(map[string]Machine)
	for mname := range conf.Machines {
		domain, err := l.DomainLookupByName(mname)
		if err != nil {
			if libvirt.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		state, err := l.DomainIsActive(domain)
		if err != nil {
			return nil, err
		}
		if state == 0 {
			continue
		}

		machine, err := MachineFromLibvirtDomain(l, domain)
		if err != nil {
			return nil, err
		}
		machines[domain.Name] = *machine
	}

	return machines, nil
}

func AttachDeviceToVirtualMachine(machine Machine, device Device) error {
	l, err := NewVirshConnection()
	if err != nil {
		return err
	}
	defer l.Disconnect() //nolint:errcheck

	xml, err := device.HostDevXML()
	if err != nil {
		return err
	}
	return l.DomainAttachDevice(machine.Domain, xml)
}

func DetachDeviceFromVirtualMachine(machine Machine, device Device) error {
	l, err := NewVirshConnection()
	if err != nil {
		return err
	}
	defer l.Disconnect() //nolint:errcheck

	xml, err := device.HostDevXML()
	if err != nil {
		return err
	}
	return l.DomainDetachDevice(machine.Domain, xml)
}
