# libvirt-usb-hotplugd

Daemon that automatically attaches and detaches USB devices from
libvirt-based virtual machines.

## How does it work?

Every check interval libvirt-usb-hotplugd creates a list of all USB
devices connected to the host. It then compares the device attributes
to a list of configured matchers for a given virtual machine. If the
device attributes are a match this device is then attached to the
virtual machine using libvirt. It also removes devices from virtual
machines that no longer match the configured attributes.
Devices can be matched by a multitude of attributes. The simpliest
ones are USB bus and device number. Since at least the device number
cannot be considered stable across reboots those attributes are not
very useful. Devices might also be matched by the USB vendor and
product ids. This is more useful but could also be done using the
standard libvirt domain XML format. Where libvirt-usb-hotplugd shines
is that it makes it possible to match to every environment variable
and tag exposed by udev. This is very useful for devices that have
a unique serial number and the device driver exposes this number to
udev.
Udev environment variables and tags are read from the uevent file in
sysfs as well as the udev data directy. The resulting environment
variable names should be the same as can be queried using `udevadm`.

To find out the names and values of those variables first find the
bus and device number of the device using `lsusb`:

```
equinox@ws ~ % lsusb
Bus 001 Device 001: ID 1d6b:0002 Linux Foundation 2.0 root hub
Bus 002 Device 001: ID 1d6b:0003 Linux Foundation 3.0 root hub
Bus 003 Device 001: ID 1d6b:0002 Linux Foundation 2.0 root hub
Bus 003 Device 002: ID 05e3:0608 Genesys Logic, Inc. Hub
Bus 003 Device 003: ID 046d:c08e Logitech, Inc. G MX518 Gaming Mouse (MU0053)
Bus 003 Device 005: ID 046d:0825 Logitech, Inc. Webcam C270
Bus 003 Device 007: ID 3434:0211 Keychron Keychron K1 Pro
Bus 004 Device 001: ID 1d6b:0003 Linux Foundation 3.0 root hub
```

Let's say we want to pass the Logitech webcam to a virtual machine.
This device is currently connected to bus `003` and uses the device
number `005`. To find out what udev enviroment variables exist run this
command:

```
equinox@ws ~ % udevadm info --query=all --name /dev/bus/usb/003/005
P: /devices/pci0000:00/0000:00:01.2/0000:02:00.0/0000:03:08.0/0000:06:00.3/usb3/3-6/3-6.3
M: 3-6.3
R: 3
U: usb
T: usb_device
D: c 189:260
N: bus/usb/003/005
L: 0
V: usb
E: DEVPATH=/devices/pci0000:00/0000:00:01.2/0000:02:00.0/0000:03:08.0/0000:06:00.3/usb3/3-6/3-6.3
E: DEVNAME=/dev/bus/usb/003/005
E: DEVTYPE=usb_device
E: DRIVER=usb
E: PRODUCT=46d/825/12
E: TYPE=239/2/1
E: BUSNUM=003
E: DEVNUM=005
E: MAJOR=189
E: MINOR=260
E: SUBSYSTEM=usb
E: USEC_INITIALIZED=4269642
E: ID_BUS=usb
E: ID_MODEL=0825
E: ID_MODEL_ENC=0825
E: ID_MODEL_ID=0825
E: ID_SERIAL=046d_0825_<redacted-serial>
E: ID_SERIAL_SHORT=<redacted-serial>
E: ID_VENDOR=046d
E: ID_VENDOR_ENC=046d
E: ID_VENDOR_ID=046d
E: ID_REVISION=0012
E: ID_USB_MODEL=0825
E: ID_USB_MODEL_ENC=0825
E: ID_USB_MODEL_ID=0825
E: ID_USB_SERIAL=046d_0825_<redacted-serial>
E: ID_USB_SERIAL_SHORT=<redacted-serial>
E: ID_USB_VENDOR=046d
E: ID_USB_VENDOR_ENC=046d
E: ID_USB_VENDOR_ID=046d
E: ID_USB_REVISION=0012
E: ID_USB_INTERFACES=:0e0100:0e0200:010100:010200:
E: ID_VENDOR_FROM_DATABASE=Logitech, Inc.
E: ID_MODEL_FROM_DATABASE=Webcam C270
E: ID_PATH_WITH_USB_REVISION=pci-0000:06:00.3-usbv2-0:6.3
E: ID_PATH=pci-0000:06:00.3-usb-0:6.3
E: ID_PATH_TAG=pci-0000_06_00_3-usb-0_6_3
E: TAGS=:snap_cups_ippeveprinter:snap_cups_cupsd:
E: CURRENT_TAGS=:snap_cups_ippeveprinter:snap_cups_cupsd:
```

Every line starting with `E:` contains an environment variable that might
be used to match the device. Exceptions to this rule are `TAGS` and
`CURRENT_TAGS`. Matching against tags is also possible but done in a sligtly
different way.

Another way to find the variable names and tags available is to run the daemon
in debug mode:

```
WHAWTY_LIBVIRT_USB_HOTPLUGD_DEBUG=1  ./whawty-libvirt-usb-hotplugd config.yml
```

Please mind that the daemon won't start with an empty configuration file. You
can work around this issue by putting the following line into `config.yml`.

```yaml
{}
```


## How do i configure it?

The daemon takes the path to a single configuration file as it's first
and only argument.

Given the above example the following config file can be used to
pass the USB webcam to a virtual machine called `webcam-test` in libvirt:

```yaml
interval: 5s
machines:
  webcam-test:
    devices:
    - vendor-id: 0x046d
      product-id: 0x0825
      udev:
        env:
        - name: ID_USB_SERIAL_SHORT
          equals: '<redacted-serial>'
```

Besides `equals` you can also use a regular expression to match against
the value of the given environment variable. See the [example configuration](sample-config.yml)
to see how this is done.

The configuration file can be broken up into several files for easier management. For
this the daemon looks for a directory named `machines.d` in the same directory as the main
configuration file. Any file ending with `.yml` corresponds to a virtual machine. The name
of the machine is extracted from the file name and the contents must be the device matchers.
For example the configuration above could be split up into two files:

The main file `/path/to/global.yml`

```yaml
interval: 5s
machines: {}
```
Please mind taht since 5 seconds is the default interval you might as well use `{}` as
the only contents of the configuration.
You can add then add the machine specific snippet `/path/to/machines.d/webcam-test.yml`:

```yaml
devices:
- vendor-id: 0x046d
  product-id: 0x0825
  udev:
    env:
    - name: ID_USB_SERIAL_SHORT
      equals: '<redacted-serial>'
```

Machines defined in the `/path/to/global.yml` would be merged with the ones found in the
`/path/to/machines.d/`. In case a machine is found in the main config file as well as in
the `machines.d` directory the latter will take precedence and overwrite the matchers
found in the main configuration (they won't get merged together).
