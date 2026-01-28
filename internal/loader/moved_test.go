package loader

import (
	"path/filepath"
	"testing"
)

func TestParseMovedBlocks(t *testing.T) {
	dir := filepath.Join(getTestdataDir(), "with_moved")
	snap, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(snap.MovedBlocks) != 2 {
		t.Fatalf("expected 2 moved blocks, got %d", len(snap.MovedBlocks))
	}

	// Check first moved block (resource)
	moved1 := snap.MovedBlocks[0]
	if moved1.From != "aws_s3_bucket.old_name" {
		t.Errorf("From = %q, want %q", moved1.From, "aws_s3_bucket.old_name")
	}
	if moved1.To != "aws_s3_bucket.new_name" {
		t.Errorf("To = %q, want %q", moved1.To, "aws_s3_bucket.new_name")
	}

	// Check second moved block (module)
	moved2 := snap.MovedBlocks[1]
	if moved2.From != "module.old_module" {
		t.Errorf("From = %q, want %q", moved2.From, "module.old_module")
	}
	if moved2.To != "module.new_module" {
		t.Errorf("To = %q, want %q", moved2.To, "module.new_module")
	}
}

func TestParseMovedBlocksEmpty(t *testing.T) {
	dir := filepath.Join(getTestdataDir(), "basic")
	snap, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(snap.MovedBlocks) != 0 {
		t.Errorf("expected 0 moved blocks, got %d", len(snap.MovedBlocks))
	}
}
