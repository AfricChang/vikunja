// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

//go:build mage && windows

package main

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

func pnpmCommand(ctx context.Context, args ...string) *exec.Cmd {
	// Corepack installs pnpm.cmd, which CreateProcess cannot execute directly.
	cmd := exec.CommandContext(ctx, "cmd.exe", append([]string{"/d", "/c", "pnpm"}, args...)...) //nolint:gosec // Only developer-supplied pnpm CLI arguments are forwarded.
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP, HideWindow: true}
}

func killProcessGroup(ctx context.Context, cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	// Killing only the pnpm shell would leave its Node preview server running.
	kill := exec.CommandContext(ctx, "taskkill.exe", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)) //nolint:gosec // The PID is the numeric id of the child this task started.
	kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	killErr := kill.Run()
	if killErr != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	var exitErr *exec.ExitError
	if waitErr == nil || (killErr == nil && errors.As(waitErr, &exitErr)) {
		return nil
	}
	return errors.Join(killErr, waitErr)
}
