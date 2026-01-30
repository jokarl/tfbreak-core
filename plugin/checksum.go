package plugin

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

// Checksummer verifies file checksums against a checksums.txt file.
type Checksummer struct {
	checksums map[string]string // filename -> sha256 hex
}

// ParseChecksums parses a checksums.txt file in the standard format:
// <sha256>  <filename>
// or
// <sha256> <filename>
func ParseChecksums(r io.Reader) (*Checksummer, error) {
	checksums := make(map[string]string)
	scanner := bufio.NewScanner(r)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on whitespace (could be single or double space)
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid checksum format at line %d: expected '<hash> <filename>'", lineNum)
		}

		hash := strings.ToLower(parts[0])
		filename := parts[1]

		// Validate hash format (64 hex chars for SHA256)
		if len(hash) != 64 {
			return nil, fmt.Errorf("invalid SHA256 hash at line %d: expected 64 characters, got %d", lineNum, len(hash))
		}
		if _, err := hex.DecodeString(hash); err != nil {
			return nil, fmt.Errorf("invalid hex in hash at line %d: %w", lineNum, err)
		}

		checksums[filename] = hash
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading checksums: %w", err)
	}

	if len(checksums) == 0 {
		return nil, fmt.Errorf("no checksums found in file")
	}

	return &Checksummer{checksums: checksums}, nil
}

// Verify checks if the data matches the expected checksum for the given filename.
func (c *Checksummer) Verify(filename string, r io.Reader) error {
	expected, ok := c.checksums[filename]
	if !ok {
		return fmt.Errorf("no checksum found for %s", filename)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, r); err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	actual := hex.EncodeToString(hash.Sum(nil))

	if actual != expected {
		return &ChecksumMismatchError{
			Filename: filename,
			Expected: expected,
			Actual:   actual,
		}
	}

	return nil
}

// GetChecksum returns the expected checksum for a filename.
func (c *Checksummer) GetChecksum(filename string) (string, bool) {
	hash, ok := c.checksums[filename]
	return hash, ok
}

// ChecksumMismatchError indicates a checksum verification failure.
type ChecksumMismatchError struct {
	Filename string
	Expected string
	Actual   string
}

func (e *ChecksumMismatchError) Error() string {
	return fmt.Sprintf("checksum mismatch for %s: expected %s, got %s", e.Filename, e.Expected, e.Actual)
}

// ComputeSHA256 computes the SHA256 hash of data from a reader.
func ComputeSHA256(r io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, r); err != nil {
		return "", fmt.Errorf("failed to compute checksum: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
