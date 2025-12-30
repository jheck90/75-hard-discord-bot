package migrations

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// Migration represents a database migration
type Migration struct {
	Version   int
	Name      string
	SQL       string
	Checksum  string
	AppliedAt sql.NullTime
}

// Manager handles database migrations
type Manager struct {
	db *sql.DB
}

// NewManager creates a new migration manager
func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// EnsureMigrationsTable creates the schema_migrations table if it doesn't exist
func (m *Manager) EnsureMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER NOT NULL,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			checksum VARCHAR(64) NOT NULL,
			PRIMARY KEY (version, name)
		);
		
		CREATE INDEX IF NOT EXISTS idx_schema_migrations_version 
			ON schema_migrations(version);
	`

	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	return nil
}

// GetAppliedMigrations returns a map of applied migrations by version
func (m *Manager) GetAppliedMigrations() (map[int]Migration, error) {
	applied := make(map[int]Migration)

	query := `SELECT version, name, applied_at, checksum FROM schema_migrations ORDER BY version`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var mig Migration
		err := rows.Scan(&mig.Version, &mig.Name, &mig.AppliedAt, &mig.Checksum)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration row: %w", err)
		}
		applied[mig.Version] = mig
	}

	return applied, rows.Err()
}

// stripTransactionStatements removes BEGIN/COMMIT statements from SQL since we handle transactions in Go
// This function removes lines that contain only BEGIN; or COMMIT; (case-insensitive, with optional whitespace)
func stripTransactionStatements(sql string) string {
	lines := strings.Split(sql, "\n")
	var filteredLines []string
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		upperTrimmed := strings.ToUpper(trimmed)
		// Skip lines that are just BEGIN; or COMMIT;
		if upperTrimmed == "BEGIN;" || upperTrimmed == "COMMIT;" || 
		   upperTrimmed == "BEGIN" || upperTrimmed == "COMMIT" {
			continue
		}
		filteredLines = append(filteredLines, line)
	}
	
	result := strings.Join(filteredLines, "\n")
	return strings.TrimSpace(result)
}

// splitSQLStatements splits SQL into individual statements, handling $$ delimiters for functions
func splitSQLStatements(sql string) []string {
	// Use a more sophisticated approach: find dollar-quoted blocks and protect them
	var statements []string
	var current strings.Builder
	inDollarQuote := false
	dollarTag := ""
	
	i := 0
	for i < len(sql) {
		char := sql[i]
		
		// Check for dollar quote start
		if !inDollarQuote && char == '$' {
			// Find the closing $ to determine the tag (could be $$ or $tag$)
			tagStart := i
			foundTag := false
			for j := i + 1; j < len(sql); j++ {
				if sql[j] == '$' {
					dollarTag = sql[tagStart : j+1]
					inDollarQuote = true
					current.WriteString(dollarTag)
					i = j + 1
					foundTag = true
					break
				}
			}
			if !foundTag {
				// Not a dollar quote, just a regular $
				current.WriteByte(char)
				i++
			}
			continue
		}
		
		// Check for dollar quote end
		if inDollarQuote {
			// Check if we've found the closing tag at current position
			if i+len(dollarTag) <= len(sql) {
				potentialTag := sql[i : i+len(dollarTag)]
				if potentialTag == dollarTag {
					current.WriteString(dollarTag)
					inDollarQuote = false
					dollarTag = ""
					i += len(dollarTag)
					continue
				}
			}
			// Still inside dollar quote, add character
			current.WriteByte(char)
			i++
			continue
		}
		
		// Outside dollar quote - normal processing
		current.WriteByte(char)
		
		// Check for statement terminator (semicolon outside of dollar quotes)
		if char == ';' {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" && stmt != ";" {
				statements = append(statements, stmt)
			}
			current.Reset()
		}
		i++
	}
	
	// Add any remaining statement
	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}
	
	return statements
}

// ApplyMigration applies a single migration within a transaction
func (m *Manager) ApplyMigration(migration Migration) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Strip BEGIN/COMMIT statements since we're already in a transaction
	sql := stripTransactionStatements(migration.SQL)

	// Split SQL into individual statements, properly handling dollar-quoted function bodies
	statements := splitSQLStatements(sql)
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		_, err = tx.Exec(stmt)
		if err != nil {
			return fmt.Errorf("failed to execute migration %d (%s) statement %d: %w", 
				migration.Version, migration.Name, i+1, err)
		}
	}

	// Record migration in schema_migrations table
	// Use stripped SQL for checksum to match what we actually execute
	checksum := CalculateChecksum(sql)
	_, err = tx.Exec(
		`INSERT INTO schema_migrations (version, name, checksum) VALUES ($1, $2, $3)`,
		migration.Version,
		migration.Name,
		checksum,
	)
	if err != nil {
		return fmt.Errorf("failed to record migration %d (%s): %w", migration.Version, migration.Name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration %d (%s): %w", migration.Version, migration.Name, err)
	}

	log.Printf("✅ Applied migration %04d_%s", migration.Version, migration.Name)
	return nil
}

// ValidateChecksums validates that all applied migrations match their stored checksums
func (m *Manager) ValidateChecksums(migrations []Migration) error {
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	for _, migration := range migrations {
		appliedMig, exists := applied[migration.Version]
		if !exists {
			continue // Not applied yet, skip validation
		}

		// Calculate checksum with stripped SQL (current approach)
		strippedSQL := stripTransactionStatements(migration.SQL)
		calculatedChecksum := CalculateChecksum(strippedSQL)
		
		// Also try with original SQL (for migrations applied before this change)
		originalChecksum := CalculateChecksum(migration.SQL)
		
		// Accept either checksum to handle migrations applied before/after this change
		if calculatedChecksum != appliedMig.Checksum && originalChecksum != appliedMig.Checksum {
			return fmt.Errorf("checksum mismatch for migration %d (%s): stored=%s, calculated (stripped)=%s, calculated (original)=%s",
				migration.Version, migration.Name, appliedMig.Checksum, calculatedChecksum, originalChecksum)
		}
	}

	return nil
}

// Run executes the migration process:
// 1. Ensures schema_migrations table exists
// 2. Scans for migration files
// 3. Validates checksums of already-applied migrations
// 4. Applies pending migrations
func (m *Manager) Run() error {
	log.Println("🔄 Starting database migrations...")

	// Step 1: Ensure migrations table exists
	if err := m.EnsureMigrationsTable(); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	// Step 2: Scan migration files
	migrations, err := ScanMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to scan migration files: %w", err)
	}

	if len(migrations) == 0 {
		log.Println("⚠️  No migration files found")
		return nil
	}

	log.Printf("📁 Found %d migration file(s)", len(migrations))

	// Step 3: Get applied migrations
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Step 4: Validate checksums of already-applied migrations
	if err := m.ValidateChecksums(migrations); err != nil {
		return fmt.Errorf("checksum validation failed: %w", err)
	}

	// Step 5: Apply pending migrations
	pendingCount := 0
	for _, migration := range migrations {
		if _, exists := applied[migration.Version]; exists {
			log.Printf("⏭️  Skipping migration %04d_%s (already applied)", migration.Version, migration.Name)
			continue
		}

		log.Printf("🔄 Applying migration %04d_%s...", migration.Version, migration.Name)
		if err := m.ApplyMigration(migration); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}
		pendingCount++
	}

	if pendingCount == 0 {
		log.Println("✅ All migrations are up to date")
	} else {
		log.Printf("✅ Applied %d pending migration(s)", pendingCount)
	}

	return nil
}
