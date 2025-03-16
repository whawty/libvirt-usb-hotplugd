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
	"os"

	"gopkg.in/yaml.v3"
)

type DeviceMatcher struct {
	Bus         *int    `yaml:"bus"`
	Device      *int    `yaml:"device"`
	VendorID    *uint16 `yaml:"vendor-id"`
	ProductID   *uint16 `yaml:"product-id"`
	VendorName  *string `yaml:"vendor-name"`
	ProductName *string `yaml:"product-name"`
}

type MachineConfig struct {
	DeviceMatchers []DeviceMatcher `yaml:"devices"`
}

type Config struct {
	Machines map[string]MachineConfig `yaml:"machines"`
}

func readConfig(configfile string) (*Config, error) {
	file, err := os.Open(configfile)
	if err != nil {
		return nil, fmt.Errorf("Error opening config file: %s", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	c := &Config{}
	if err = decoder.Decode(c); err != nil {
		return nil, fmt.Errorf("Error parsing config file: %s", err)
	}
	// TODO: sanity check matchers??
	return c, nil
}
