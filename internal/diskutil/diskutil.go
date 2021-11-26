// Package diskutil provides the functionality necessary for interacting with macOS's diskutil CLI.
package diskutil

//go:generate mockgen -source=diskutil.go -destination=mocks/mock_diskutil.go

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/ec2-macos-utils/internal/diskutil/types"
	"github.com/aws/ec2-macos-utils/internal/system"

	"github.com/Masterminds/semver"
)

const (
	// minimumGrowFreeSpace defines the minimum amount of free space (in bytes) required to attempt running
	// diskutil's resize command.
	minimumGrowFreeSpace = 1000000
)

// FreeSpaceError defines an error to distinguish when there's not enough space to grow the specified container.
type FreeSpaceError struct {
	freeSpaceBytes uint64
}

func (e FreeSpaceError) Error() string {
	return fmt.Sprintf("%d bytes available", e.freeSpaceBytes)
}

// DiskUtil outlines the functionality necessary for wrapping macOS's diskutil tool.
type DiskUtil interface {
	// APFS outlines the functionality necessary for wrapping diskutil's "apfs" verb.
	APFS
	// Info fetches raw disk information for the specified device identifier.
	Info(id string) (*types.DiskInfo, error)
	// List fetches all disk and partition information for the system.
	// This output will be filtered based on the args provided.
	List(args []string) (*types.SystemPartitions, error)
	// RepairDisk attempts to repair the disk for the specified device identifier.
	// This process requires root access.
	RepairDisk(id string) (string, error)
}

// APFS outlines the functionality necessary for wrapping diskutil's "apfs" verb.
type APFS interface {
	// ResizeContainer attempts to grow the APFS container with the given device identifier
	// to the specified size. If the given size is 0, ResizeContainer will attempt to grow
	// the disk to its maximum size.
	ResizeContainer(id string, size string) (string, error)
}

// ForProduct creates a new diskutil controller for the given product.
func ForProduct(p *system.Product) (DiskUtil, error) {
	switch p.Release {
	case system.Mojave:
		return newMojave(p.Version)
	case system.Catalina:
		return newCatalina(p.Version)
	case system.BigSur:
		return newBigSur(p.Version)
	case system.Monterey:
		return newMonterey(p.Version)
	default:
		return nil, errors.New("unknown release")
	}
}

// newMojave configures the DiskUtil for the specified Mojave version.
func newMojave(version semver.Version) (*DiskUtilityMojave, error) {
	du := &DiskUtilityMojave{
		embeddedDiskutil: &DiskUtilityCmd{},
		dec:              &PlistDecoder{},
	}

	return du, nil
}

// newCatalina configures the DiskUtil for the specified Catalina version.
func newCatalina(version semver.Version) (*DiskUtilityCatalina, error) {
	du := &DiskUtilityCatalina{
		embeddedDiskutil: &DiskUtilityCmd{},
		dec:              &PlistDecoder{},
	}

	return du, nil
}

// newBigSur configures the DiskUtil for the specified Big Sur version.
func newBigSur(version semver.Version) (*DiskUtilityBigSur, error) {
	du := &DiskUtilityBigSur{
		embeddedDiskutil: &DiskUtilityCmd{},
		dec:              &PlistDecoder{},
	}

	return du, nil
}

// newMonterey configures the DiskUtil for the specified Monterey version.
func newMonterey(version semver.Version) (*DiskUtilityBigSur, error) {
	du := &DiskUtilityBigSur{
		embeddedDiskutil: &DiskUtilityCmd{},
		dec:              &PlistDecoder{},
	}

	return du, nil
}

// embeddedDiskutil is a private interface used to embed UtilImpl into implementation-specific structs.
type embeddedDiskutil interface {
	UtilImpl
}

// DiskUtilityMojave wraps all the functionality necessary for interacting with macOS's diskutil on Mojave. The
// major difference is that the raw plist data emitted by macOS's diskutil CLI doesn't include the physical store data.
// This requires a separate fetch to find the specific physical store information for the disk(s).
type DiskUtilityMojave struct {
	// embeddedDiskutil provides the diskutil implementation to prevent manual wiring between UtilImpl and DiskUtil.
	embeddedDiskutil

	// dec is the Decoder used to decode the raw output from UtilImpl into usable structs.
	dec Decoder
}

// List utilizes the UtilImpl.List method to fetch the raw list output from diskutil and returns the decoded
// output in a SystemPartitions struct. List also attempts to update each APFS Volume's physical store via a separate
// fetch method since the version of diskutil on Mojave doesn't provide that information in its List verb.
//
// It is possible for List to fail when updating the physical stores, but it will still return the original data
// that was decoded into the SystemPartitions struct.
func (d *DiskUtilityMojave) List(args []string) (*types.SystemPartitions, error) {
	partitions, err := list(d.embeddedDiskutil, d.dec, args)
	if err != nil {
		return nil, err
	}

	err = updatePhysicalStores(partitions)
	if err != nil {
		return partitions, err
	}

	return partitions, nil
}

// Info utilizes the UtilImpl.Info method to fetch the raw disk output from diskutil and returns the decoded
// output in a DiskInfo struct. Info also attempts to update each APFS Volume's physical store via a separate
// fetch method since the version of diskutil on Mojave doesn't provide that information in its Info verb.
//
// It is possible for Info to fail when updating the physical stores, but it will still return the original data
// that was decoded into the DiskInfo struct.
func (d *DiskUtilityMojave) Info(id string) (*types.DiskInfo, error) {
	disk, err := info(d.embeddedDiskutil, d.dec, id)
	if err != nil {
		return nil, err
	}

	err = updatePhysicalStore(disk)
	if err != nil {
		return disk, err
	}

	return disk, nil
}

// DiskUtilityCatalina wraps all the functionality necessary for interacting with macOS's diskutil in GoLang.
type DiskUtilityCatalina struct {
	// embeddedDiskutil provides the diskutil implementation to prevent manual wiring between UtilImpl and DiskUtil.
	embeddedDiskutil

	// dec is the Decoder used to decode the raw output from UtilImpl into usable structs.
	dec Decoder
}

// List utilizes the UtilImpl.List method to fetch the raw list output from diskutil and returns the decoded
// output in a SystemPartitions struct.
func (d *DiskUtilityCatalina) List(args []string) (*types.SystemPartitions, error) {
	return list(d.embeddedDiskutil, d.dec, args)
}

// Info utilizes the UtilImpl.Info method to fetch the raw disk output from diskutil and returns the decoded
// output in a DiskInfo struct.
func (d *DiskUtilityCatalina) Info(id string) (*types.DiskInfo, error) {
	return info(d.embeddedDiskutil, d.dec, id)
}

// DiskUtilityBigSur wraps all the functionality necessary for interacting with macOS's diskutil in GoLang.
type DiskUtilityBigSur struct {
	// embeddedDiskutil provides the diskutil implementation to prevent manual wiring between UtilImpl and DiskUtil.
	embeddedDiskutil

	// dec is the Decoder used to decode the raw output from UtilImpl into usable structs.
	dec Decoder
}

// List utilizes the UtilImpl.List method to fetch the raw list output from diskutil and returns the decoded
// output in a SystemPartitions struct.
func (d *DiskUtilityBigSur) List(args []string) (*types.SystemPartitions, error) {
	return list(d.embeddedDiskutil, d.dec, args)
}

// Info utilizes the UtilImpl.Info method to fetch the raw disk output from diskutil and returns the decoded
// output in a DiskInfo struct.
func (d *DiskUtilityBigSur) Info(id string) (*types.DiskInfo, error) {
	return info(d.embeddedDiskutil, d.dec, id)
}

// DiskUtilityMonterey wraps all the functionality necessary for interacting with macOS's diskutil in GoLang.
type DiskUtilityMonterey struct {
	// embeddedDiskutil provides the diskutil implementation to prevent manual wiring between UtilImpl and DiskUtil.
	embeddedDiskutil

	// dec is the Decoder used to decode the raw output from UtilImpl into usable structs.
	dec Decoder
}

// List utilizes the UtilImpl.List method to fetch the raw list output from diskutil and returns the decoded
// output in a SystemPartitions struct.
func (d *DiskUtilityMonterey) List(args []string) (*types.SystemPartitions, error) {
	return list(d.embeddedDiskutil, d.dec, args)
}

// Info utilizes the UtilImpl.Info method to fetch the raw disk output from diskutil and returns the decoded
// output in a DiskInfo struct.
func (d *DiskUtilityMonterey) Info(id string) (*types.DiskInfo, error) {
	return info(d.embeddedDiskutil, d.dec, id)
}

// info is a wrapper that fetches the raw diskutil info data and decodes it into a usable types.DiskInfo struct.
func info(util UtilImpl, decoder Decoder, id string) (*types.DiskInfo, error) {
	// Fetch the raw disk information from the util
	rawDisk, err := util.Info(id)
	if err != nil {
		return nil, err
	}

	// Create a reader for the raw data
	reader := strings.NewReader(rawDisk)

	// Decode the raw data into a more usable DiskInfo struct
	disk, err := decoder.DecodeDiskInfo(reader)
	if err != nil {
		return nil, err
	}

	return disk, nil
}

// list is a wrapper that fetches the raw diskutil list data and decodes it into a usable types.SystemPartitions struct.
func list(util UtilImpl, decoder Decoder, args []string) (*types.SystemPartitions, error) {
	// Fetch the raw list information from the util
	rawPartitions, err := util.List(args)
	if err != nil {
		return nil, err
	}

	// Create a reader for the raw data
	reader := strings.NewReader(rawPartitions)

	// Decode the raw data into a more usable SystemPartitions struct
	partitions, err := decoder.DecodeSystemPartitions(reader)
	if err != nil {
		return nil, err
	}

	return partitions, nil
}
