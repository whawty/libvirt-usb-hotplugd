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
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type UdevEnvMatcher struct {
	Name    string  `yaml:"name"`
	Equals  *string `yaml:"equals"`
	Pattern *string `yaml:"pattern"`
	re      *regexp.Regexp
}

type DeviceMatcher struct {
	Bus       *int    `yaml:"bus"`
	Device    *int    `yaml:"device"`
	VendorID  *uint16 `yaml:"vendor-id"`
	ProductID *uint16 `yaml:"product-id"`
	Udev      struct {
		Env         []UdevEnvMatcher `yaml:"env"`
		Tags        []string         `yaml:"tags"`
		CurrentTags []string         `yaml:"current-tags"`
	} `yaml:"udev"`
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
		return fmt.Errorf("failed to open config snippet: %v", err)
	}
	defer file.Close() //nolint:errcheck

	mname := strings.TrimSuffix(filepath.Base(filename), ".yml")
	mconf := &MachineConfig{}

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	if err = decoder.Decode(mconf); err != nil {
		return fmt.Errorf("failed to parse config snippet '%s': %v", filename, err)
	}
	if _, exists := conf.Machines[mname]; exists {
		wl.Printf("machine '%s' has been found in the global config file as well as in machines.d directory. The latter takes precedence", mname)
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

func (conf *Config) initialize() error {
	for machine, mconf := range conf.Machines {
		if len(mconf.DeviceMatchers) == 0 {
			return fmt.Errorf("machine %s has no device matchers", machine)
		}
		for idx, matcher := range mconf.DeviceMatchers {
			if len(matcher.Udev.Env) > 0 {
				for i, udevEnv := range matcher.Udev.Env {
					if udevEnv.Name == "" {
						return fmt.Errorf("device matcher %d of machine %s: udev-env name must not be empty ", idx, machine)
					}
					if udevEnv.Equals != nil {
						if udevEnv.Pattern != nil {
							return fmt.Errorf("device matcher %d of machine %s: 'equals' and 'pattern' are mutually exclusive ", idx, machine)
						}
						continue
					}
					if udevEnv.Pattern != nil {
						re, err := regexp.Compile(*udevEnv.Pattern)
						if err != nil {
							return fmt.Errorf("device matcher %d of machine %s: failed to compile pattern: %v", idx, machine, err)
						}
						matcher.Udev.Env[i].re = re
						continue
					}
					return fmt.Errorf("device matcher %d of machine %s: udev-env needs at least one of 'equals' or 'pattern'", idx, machine)
				}
			} else {
				if matcher.Bus == nil && matcher.Device == nil && matcher.VendorID == nil && matcher.ProductID == nil && len(matcher.Udev.Tags) == 0 && len(matcher.Udev.CurrentTags) == 0 {
					return fmt.Errorf("device matcher %d of machine %s: empty matcher is not allowed", idx, machine)
				}
			}
		}
	}
	return nil

}

func readConfig(configfile string) (*Config, error) {
	file, err := os.Open(configfile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %s", err)
	}
	defer file.Close() //nolint:errcheck

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	c := &Config{}
	if err = decoder.Decode(c); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %s", err)
	}
	if c.Interval == 0 {
		c.Interval = 5 * time.Second
	}
	if err = c.loadMachinesConfigFromDirectory(configfile); err != nil {
		return nil, err
	}
	if err = c.initialize(); err != nil {
		return nil, err
	}
	return c, nil
}
