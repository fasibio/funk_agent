package main

import (
	"testing"

	"github.com/bradleyjkemp/cupaloy/v2"
)

func Test_MessageTypeLog(t *testing.T) {
	t.Run("snapshot MessageTypeLog ", func(tt *testing.T) {
		cupaloy.SnapshotT(t, MessageTypeLog)
	})
}

func Test_MessageTypeStats(t *testing.T) {
	t.Run("snapshot MessageTypeStats ", func(tt *testing.T) {
		cupaloy.SnapshotT(t, MessageTypeStats)
	})
}
