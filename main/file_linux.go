// Originally part of github.com/gen2brain/dlgs but we only need file save
// and unfortunately that is one feature that library does not support
// TODO: determine how we should handle the two major OSes (OSX, Win10)
// +build linux,!windows,!darwin,!js

package main

import (
	"errors"
	"os/exec"
	"strings"
	"syscall"
)

// cmdPath looks for supported programs in PATH
func cmdPath() (string, error) {
	cmd, err := exec.LookPath("qarma")
	if err != nil {
		e := err
		cmd, err = exec.LookPath("zenity")
		if err != nil {
			return "", errors.New("dlgs: " + err.Error() + "; " + e.Error())
		}
	}

	return cmd, err
}

// File displays a file dialog, returning the selected file/directory and a bool for success.
func File(title, filter string, directory bool) (string, bool, error) {
	cmd, err := cmdPath()
	if err != nil {
		return "", false, err
	}

	dir := ""
	if directory {
		dir = "--directory"
	}

	fileFilter := ""
	if filter != "" {
		fileFilter = "--file-filter=" + filter
	}

	o, err := exec.Command(cmd, "--file-selection", "--save", "--confirm-overwrite", "--title", title, fileFilter, dir).Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			return "", ws.ExitStatus() == 0, nil
		}
	}

	ret := true
	out := strings.TrimSpace(string(o))
	if out == "" {
		ret = false
	}

	return out, ret, err
}
