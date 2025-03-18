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
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type DeviceMatcher struct {
	Bus       *int    `yaml:"bus"`
	Device    *int    `yaml:"device"`
	VendorID  *uint16 `yaml:"vendor-id"`
	ProductID *uint16 `yaml:"product-id"`
	// TODO: add matchers for udev attributes
}

type MachineConfig struct {
	DeviceMatchers []DeviceMatcher `yaml:"devices"`
}

type Config struct {
	Interval time.Duration            `yaml:"interval"`
	Machines map[string]MachineConfig `yaml:"machines"`
}

func (conf *Config) loadMachineConfigFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("Error opening config file: %s", err)
	}
	defer file.Close()

	mname := strings.TrimSuffix(filepath.Base(filename), ".yml")
	mconf := &MachineConfig{}

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err = decoder.Decode(mconf); err != nil {
		return fmt.Errorf("Error parsing config snippet '%s': %s", filename, err)
	}
	if _, exists := conf.Machines[mname]; exists {
		wdl.Printf("machine '%s' has been found in the global config file as well as in machines.d directory. The latter takes precedence")
	}
	conf.Machines[mname] = *mconf
	return nil
}

func (conf *Config) loadMachinesConfigFromDirectory(configfile string) error {
	path, err := filepath.Abs(configfile)
	if err != nil {
		return err
	}
	machinesDir := filepath.Join(filepath.Dir(path), "machines.d")
	info, err := os.Stat(machinesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	wdl.Printf("looking for additinal config files in '%s'", machinesDir)
	files, err := os.ReadDir(machinesDir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filename := file.Name()
		if filepath.Ext(filename) != ".yml" {
			continue
		}
		wdl.Printf("loading machine config from %s", filename)
		if err = conf.loadMachineConfigFromFile(filepath.Join(machinesDir, filename)); err != nil {
			return err
		}
	}
	return nil
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
	if c.Interval == 0 {
		c.Interval = 5 * time.Second
	}
	if err = c.loadMachinesConfigFromDirectory(configfile); err != nil {
		return nil, err
	}
	// TODO: sanity check matchers??
	return c, nil
}
