package diskutil

import (
	"fmt"

	"github.com/aws/ec2-macos-utils/pkg/util"
)

// DiskUtility outlines the functionality necessary for wrapping macOS's diskutil.
type DiskUtility interface {
	List(args []string) (out string, err error)
	Info(id string) (out string, err error)
	RepairDisk(id string) (out string, err error)
	APFS
}

// APFS outlines the functionality necessary for wrapping diskutil's APFS verb.
type APFS interface {
	ResizeContainer(id, size string) (out string, err error)
}

// DiskUtilityCmd is an empty struct that provides the implementation for the DiskUtility interface.
type DiskUtilityCmd struct{}

// List uses the macOS diskutil list command to list disks and partitions in a plist format by passing the -plist arg.
// List also appends any given args to fully support the diskutil list verb.
func (d *DiskUtilityCmd) List(args []string) (out string, err error) {
	// Create the diskutil command for retrieving all disk and partition information
	//   * -plist converts diskutil's output from human-readable to the plist format
	cmdListDisks := []string{"diskutil", "list", "-plist"}

	// Append arguments to the diskutil list verb
	if len(args) > 0 {
		cmdListDisks = append(cmdListDisks, args...)
	}

	// Execute the diskutil list command and store the output
	cmdOut, err := util.ExecuteCommand(cmdListDisks, "", []string{})
	if err != nil {
		return cmdOut.Stdout, fmt.Errorf("diskutil: failed to run diskutil command to list all disks, stderr: [%s]: %v", cmdOut.Stderr, err)
	}

	return cmdOut.Stdout, nil
}

// Info uses the macOS diskutil info command to get detailed information about a disk, partition or container in a plist
// format by passing the -plist arg.
func (d *DiskUtilityCmd) Info(id string) (out string, err error) {
	// Create the diskutil command for retrieving disk information given a device identifier
	//   * -plist converts diskutil's output from human-readable to the plist format
	//   * id - the device identifier for the disk to be fetched
	cmdDiskInfo := []string{"diskutil", "info", "-plist", id}

	// Execute the diskutil info command and store the output
	cmdOut, err := util.ExecuteCommand(cmdDiskInfo, "", []string{})
	if err != nil {
		return cmdOut.Stdout, fmt.Errorf("failed to run diskutil command to fetch disk information, stderr: [%s]: %v", cmdOut.Stderr, err)
	}

	return cmdOut.Stdout, nil
}

// RepairDisk uses the macOS diskutil diskRepair command to repair the specified volume and get updated information
// (e.g. amount of free space).
func (d *DiskUtilityCmd) RepairDisk(id string) (out string, err error) {
	// TODO: this will need to be versioned for mojave and catalina/big sur since mojave uses bash
	// cmdRepairDisk represents the command used for executing macOS's diskutil to repair a disk
	// this is done by having zsh directly execute the diskutil command and provide "yes" to skip manual typing
	//   * repairDisk - indicates that a disk is going to be repaired (used to fetch amount of free space)
	//   * id - the device identifier for the disk to be repaired
	cmdRepairDisk := []string{"/bin/zsh", "-c", "yes | diskutil repairDisk " + id}

	// Execute the diskutil repairDisk command and store the output
	cmdOut, err := util.ExecuteCommand(cmdRepairDisk, "", []string{})
	if err != nil {
		return cmdOut.Stdout, fmt.Errorf("failed to run diskutil command to repair the disk, stderr: [%s]: %v", cmdOut.Stderr, err)
	}

	return cmdOut.Stdout, nil
}

// ResizeContainer uses the macOS diskutil apfs resizeContainer command to change the size of the specific container ID.
func (d *DiskUtilityCmd) ResizeContainer(id, size string) (out string, err error) {
	// cmdResizeContainer represents the command used for executing macOS's diskutil to resize a container
	//   * apfs - specifies that a virtual APFS volume is going to be modified
	//   * resizeContainer - indicates that a container is going to be resized
	//   * id - the device identifier for the container
	//   * size - the size which can be in a human readable format (e.g. "0", "110g", and "1.5t")
	cmdResizeContainer := []string{"diskutil", "apfs", "resizeContainer", id, size}

	// Execute the diskutil apfs resizeContainer command and store the output
	cmdOut, err := util.ExecuteCommand(cmdResizeContainer, "", []string{})
	if err != nil {
		return cmdOut.Stdout, fmt.Errorf("failed to run diskutil command to resize the container, stderr [%s]: %v", cmdOut.Stderr, err)
	}

	return cmdOut.Stdout, nil
}
