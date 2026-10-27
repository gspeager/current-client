//go:build nohost

package app

import (
	"errors"
	"time"
)

func hostInboxRows(string) ([]InboxRow, bool, *time.Time, string) { return nil, false, nil, "" }

func markTodoDone(string, string) error { return errors.New("host integration is off") }
