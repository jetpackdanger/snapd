// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2017 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package osutil_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "gopkg.in/check.v1"

	"github.com/snapcore/snapd/osutil"
)

type flockSuite struct{}

var _ = Suite(&flockSuite{})

// Test that opening and closing a lock works as expected, and that the mode is right.
func (s *flockSuite) TestNewFileModeLock(c *C) {
	lock, err := osutil.NewFileModeLock(filepath.Join(c.MkDir(), "name"), 0644)
	c.Assert(err, IsNil)
	defer lock.Close()

	fi, err := os.Stat(lock.Path())
	c.Assert(err, IsNil)
	c.Assert(fi.Mode().Perm(), Equals, os.FileMode(0644))
}

// Test that opening and closing a lock works as expected.
func (s *flockSuite) TestNewFileLock(c *C) {
	lock, err := osutil.NewFileLock(filepath.Join(c.MkDir(), "name"))
	c.Assert(err, IsNil)
	defer lock.Close()

	fi, err := os.Stat(lock.Path())
	c.Assert(err, IsNil)
	c.Assert(fi.Mode().Perm(), Equals, os.FileMode(0600))
}

func flockSupportsConflictExitCodeSwitch(c *C) bool {
	output, err := exec.Command("flock", "--help").CombinedOutput()
	c.Assert(err, IsNil)
	return bytes.Contains(output, []byte("--conflict-exit-code"))
}

// Test that Lock and Unlock work as expected.
func (s *flockSuite) TestLockUnlockWorks(c *C) {
	if os.Getenv("TRAVIS_BUILD_NUMBER") != "" {
		c.Skip("Cannot use this under travis")
		return
	}
	if !flockSupportsConflictExitCodeSwitch(c) {
		c.Skip("flock too old for this test")
	}

	lock, err := osutil.NewFileLock(filepath.Join(c.MkDir(), "name"))
	c.Assert(err, IsNil)
	defer lock.Close()

	// Run a flock command in another process, it should succeed because it can
	// lock the lock as we didn't do it yet.
	cmd := exec.Command("flock", "--exclusive", "--nonblock", lock.Path(), "true")
	c.Assert(cmd.Run(), IsNil)

	// Lock the lock.
	c.Assert(lock.Lock(), IsNil)

	// Run a flock command in another process, it should fail with the distinct
	// error code because we hold the lock already and we asked it not to block.
	cmd = exec.Command("flock", "--exclusive", "--nonblock",
		"--conflict-exit-code", "2", lock.Path(), "true")
	c.Assert(cmd.Run(), ErrorMatches, "exit status 2")

	// Unlock the lock.
	c.Assert(lock.Unlock(), IsNil)

	// Run a flock command in another process, it should succeed because it can
	// grab the lock again now.
	cmd = exec.Command("flock", "--exclusive", "--nonblock", lock.Path(), "true")
	c.Assert(cmd.Run(), IsNil)
}

// Test that locking a locked lock does nothing.
func (s *flockSuite) TestLockLocked(c *C) {
	lock, err := osutil.NewFileLock(filepath.Join(c.MkDir(), "name"))
	c.Assert(err, IsNil)
	defer lock.Close()

	// NOTE: technically this replaces the lock type but we only use LOCK_EX.
	c.Assert(lock.Lock(), IsNil)
	c.Assert(lock.Lock(), IsNil)
}

// Test that unlocking an unlocked lock does nothing.
func (s *flockSuite) TestUnlockUnlocked(c *C) {
	lock, err := osutil.NewFileLock(filepath.Join(c.MkDir(), "name"))
	c.Assert(err, IsNil)
	defer lock.Close()

	c.Assert(lock.Unlock(), IsNil)
}

// Test that locking or unlocking a closed lock fails.
func (s *flockSuite) TestUsingClosedLock(c *C) {
	lock, err := osutil.NewFileLock(filepath.Join(c.MkDir(), "name"))
	c.Assert(err, IsNil)
	lock.Close()

	c.Assert(lock.Lock(), ErrorMatches, "bad file descriptor")
	c.Assert(lock.Unlock(), ErrorMatches, "bad file descriptor")
}

// Test that non-blocking locking reports error on pre-acquired lock.
func (s *flockSuite) TestLockUnlockNonblockingWorks(c *C) {
	if os.Getenv("TRAVIS_BUILD_NUMBER") != "" {
		c.Skip("Cannot use this under travis")
		return
	}

	// Use the "flock" command to grab a lock for 9999 seconds in another process.
	lockPath := filepath.Join(c.MkDir(), "lock")
	sleeperKillerPath := filepath.Join(c.MkDir(), "pid")
	// we can't use --no-fork because we still support 14.04
	cmd := exec.Command("flock", "--exclusive", lockPath, "-c", fmt.Sprintf(`echo "kill $$" > %s && exec sleep 30`, sleeperKillerPath))
	c.Assert(cmd.Start(), IsNil)
	defer func() { exec.Command("/bin/sh", sleeperKillerPath).Run() }()

	// Give flock some chance to create the lock file.
	for i := 0; i < 10; i++ {
		if osutil.FileExists(lockPath) {
			break
		}
		time.Sleep(time.Millisecond * 300)
	}

	// Try to acquire the same lock file and see that it is busy.
	lock, err := osutil.NewFileLock(lockPath)
	c.Assert(err, IsNil)
	c.Assert(lock, NotNil)
	defer lock.Close()

	c.Assert(lock.TryLock(), Equals, osutil.ErrAlreadyLocked)
}
