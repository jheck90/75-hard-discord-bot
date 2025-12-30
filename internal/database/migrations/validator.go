package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// CalculateChecksum calculates SHA-256 checksum of migration content
func CalculateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// ValidateMigration compares a migration's checksum with a stored checksum
func ValidateMigration(migration Migration, storedChecksum string) error {
	calculated := CalculateChecksum(migration.SQL)
	if calculated != storedChecksum {
		return fmt.Errorf("checksum mismatch for migration %d (%s): expected %s, got %s",
			migration.Version, migration.Name, storedChecksum, calculated)
	}
	return nil
}
