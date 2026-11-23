// Package fieldexport is the single contract between the web field exports
// and the native save bridge (Task 19 / F25): the approved filenames, the size
// bound and the safe write of a user-chosen destination.
package fieldexport

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	// Itinerary is the filename suggested for an imported itinerary export.
	Itinerary = "eridion-itinerary.csv"
	// PrivateActivity is the filename suggested for the private activity
	// (BGS tally) export.
	PrivateActivity = "eridion-private-activity.csv"
	// MaxBytes bounds one export.
	MaxBytes = 2 << 20
)

// ErrCancelled reports that the user dismissed the save dialog.
var ErrCancelled = errors.New("Export cancelled")

// Allowed reports whether name is one of the approved export filenames.
func Allowed(name string) bool { return name == Itinerary || name == PrivateActivity }

// Write creates path with data. The path comes from the native save dialog:
// it must be absolute, end in .csv and not exist yet (existing files are
// preserved); the file is private to the user and synced before success.
func Write(path, data string) error {
	if path == "" {
		return ErrCancelled
	}
	if len(data) > MaxBytes {
		return errors.New("Export exceeds 2 MiB")
	}
	if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Ext(path), ".csv") {
		return errors.New("Choose a CSV destination")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return errors.New("Choose a new CSV filename; existing files are preserved")
		}
		return errors.New("CSV could not be created at the selected destination")
	}
	_, err = f.WriteString(data)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
		return errors.New("CSV could not be written; the incomplete file was removed")
	}
	return nil
}
