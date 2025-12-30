package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ScanMigrationFiles scans the migrations directory for migration files
// and returns them sorted by version number
func ScanMigrationFiles() ([]Migration, error) {
	var migrations []Migration

	// Get migrations directory path (relative to project root)
	migrationsDir := "migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		// Try alternative path if running from different directory
		migrationsDir = filepath.Join("..", "..", "..", "migrations")
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			return nil, fmt.Errorf("migrations directory not found")
		}
	}

	// Read all files from the migrations directory
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !strings.HasSuffix(filename, ".sql") {
			continue
		}

		// Parse version number from filename (format: 0001_name.sql)
		parts := strings.SplitN(filename, "_", 2)
		if len(parts) < 2 {
			continue
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid migration filename format: %s (version must be numeric)", filename)
		}

		// Read migration file content
		filePath := filepath.Join(migrationsDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Extract name (remove .sql extension)
		name := strings.TrimSuffix(parts[1], ".sql")

		migration := Migration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		}

		migrations = append(migrations, migration)
	}

	// Sort by version number
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}
