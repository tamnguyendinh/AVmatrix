package main

import (
	"runtime"
	"testing"
)

func TestHiddenProcAttr(t *testing.T) {
	attr := hiddenProcAttr()
	if attr == nil {
		t.Fatalf("hiddenProcAttr returned nil")
	}
	if runtime.GOOS == "windows" {
		if !attr.HideWindow {
			t.Fatalf("HideWindow = false, want true")
		}
		if attr.CreationFlags&0x08000000 == 0 {
			t.Fatalf("CreationFlags = %#x, want CREATE_NO_WINDOW", attr.CreationFlags)
		}
	}
}
