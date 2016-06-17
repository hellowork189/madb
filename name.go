// Copyright 2016 The Vanadium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/olekukonko/tablewriter"

	"v.io/x/lib/cmdline"
)

var cmdMadbName = &cmdline.Command{
	Children:         []*cmdline.Command{cmdMadbNameSet, cmdMadbNameUnset, cmdMadbNameList, cmdMadbNameClearAll},
	Name:             "name",
	DontInheritFlags: true,
	Short:            "Manage device nicknames",
	Long: `
Manages device nicknames, which are meant to be more human-friendly compared to
the device serials provided by adb tool.
`,
}

var cmdMadbNameSet = &cmdline.Command{
	Runner: subCommandRunnerWithFilepath{runMadbNameSet, getDefaultConfigFilePath},
	Name:   "set",
	Short:  "Set a nickname to be used in place of the device serial.",
	Long: `
Sets a human-friendly nickname that can be used when specifying the device in
any madb commands.

The device serial can be obtained using the 'adb devices -l' command.
For example, consider the following example output:

    HT4BVWV00023           device usb:3-3.4.2 product:volantisg model:Nexus_9 device:flounder_lte

The first value, 'HT4BVWV00023', is the device serial.
To assign a nickname for this device, run the following command:

    madb name set HT4BVWV00023 MyTablet

and it will assign the 'MyTablet' nickname to the device serial 'HT4BVWV00023'.
The alternative device specifiers (e.g., 'usb:3-3.4.2', 'product:volantisg')
can also have nicknames.

When a nickname is set for a device serial, the nickname can be used to specify
the device within madb commands.

There can only be one nickname for a device serial.
When the 'madb name set' command is invoked with a device serial with an already
assigned nickname, the old one will be replaced with the newly provided one.
`,
	ArgsName: "<device_serial> <nickname>",
	ArgsLong: `
<device_serial> is a device serial (e.g., 'HT4BVWV00023') or an alternative device qualifier (e.g., 'usb:3-3.4.2') obtained from 'adb devices -l' command
<nickname> is an alpha-numeric string with no special characters or spaces.
`,
}

func runMadbNameSet(env *cmdline.Env, args []string, filename string) error {
	// Check if the arguments are valid.
	if len(args) != 2 {
		return env.UsageErrorf("There must be exactly two arguments.")
	}

	serial, nickname := args[0], args[1]
	if !isValidSerial(serial) {
		return env.UsageErrorf("Not a valid device serial: %v", serial)
	}

	if !isValidName(nickname) {
		return env.UsageErrorf("Not a valid nickname: %v", nickname)
	}

	cfg, err := readConfig(filename)
	if err != nil {
		return err
	}

	// If the nickname is already in use, don't allow it at all.
	if isNameInUse(nickname, cfg) {
		return fmt.Errorf("The provided nickname %q is already in use.", nickname)
	}

	// If the serial number already has an assigned nickname, delete it first.
	// Need to do this check, because the nickname-serial map should be a one-to-one mapping.
	if name, present := reverseMap(cfg.Names)[serial]; present {
		delete(cfg.Names, name)
	}

	// Add the nickname serial mapping.
	cfg.Names[nickname] = serial

	return writeConfig(cfg, filename)
}

var cmdMadbNameUnset = &cmdline.Command{
	Runner: subCommandRunnerWithFilepath{runMadbNameUnset, getDefaultConfigFilePath},
	Name:   "unset",
	Short:  "Unset a nickname set by the 'madb name set' command.",
	Long: `
Unsets a nickname assigned by the 'madb name set' command. Either the device
serial or the assigned nickname can be specified to remove the mapping.
`,
	ArgsName: "<device_serial | nickname>",
	ArgsLong: `
There should be only one argument, which is either the device serial or the nickname.
`,
}

func runMadbNameUnset(env *cmdline.Env, args []string, filename string) error {
	// Check if the arguments are valid.
	if len(args) != 1 {
		return env.UsageErrorf("There must be exactly one argument.")
	}

	name := args[0]
	if !isValidSerial(name) && !isValidName(name) {
		return env.UsageErrorf("Not a valid device serial or name: %v", name)
	}

	cfg, err := readConfig(filename)
	if err != nil {
		return err
	}

	found := false
	for nickname, serial := range cfg.Names {
		if nickname == name || serial == name {
			delete(cfg.Names, nickname)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("The provided argument is neither a known nickname nor a device serial.")
	}

	return writeConfig(cfg, filename)
}

var cmdMadbNameList = &cmdline.Command{
	Runner: subCommandRunnerWithFilepath{runMadbNameList, getDefaultConfigFilePath},
	Name:   "list",
	Short:  "List all the existing nicknames.",
	Long: `
Lists all the currently stored nicknames of device serials.
`,
}

func runMadbNameList(env *cmdline.Env, args []string, filename string) error {
	cfg, err := readConfig(filename)
	if err != nil {
		return err
	}

	tw := tablewriter.NewWriter(os.Stdout)
	tw.SetHeader([]string{"Serial", "Nickname"})
	tw.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	tw.SetAutoFormatHeaders(false)
	tw.SetAlignment(tablewriter.ALIGN_LEFT)

	data := make([][]string, 0, len(cfg.Names))
	for nickname, serial := range cfg.Names {
		data = append(data, []string{serial, nickname})
	}

	sort.Sort(byFirstElement(data))

	for _, row := range data {
		tw.Append(row)
	}
	tw.Render()

	return nil
}

var cmdMadbNameClearAll = &cmdline.Command{
	Runner: subCommandRunnerWithFilepath{runMadbNameClearAll, getDefaultConfigFilePath},
	Name:   "clear-all",
	Short:  "Clear all the existing nicknames.",
	Long: `
Clears all the currently stored nicknames of device serials.
`,
}

func runMadbNameClearAll(env *cmdline.Env, args []string, filename string) error {
	cfg, err := readConfig(filename)
	if err != nil {
		return err
	}

	cfg.Names = make(map[string]string)
	return writeConfig(cfg, filename)
}

// reverseMap returns a new map which contains reversed key, value pairs in the original map.
// The source map is assumed to be a one-to-one mapping between keys and values.
func reverseMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}

	reversed := make(map[string]string, len(source))
	for k, v := range source {
		reversed[v] = k
	}

	return reversed
}
