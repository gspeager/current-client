//go:build nohost

package app

import "time"

func hostInboxRows(string) ([]InboxRow, bool, *time.Time) { return nil, false, nil }
