# Database Migration Planning Document

## Overview

This document outlines the database migration system architecture, schema design, and implementation strategy for the 75 Hard Discord Bot. The migration system ensures version-controlled, validated, and extensible database schema management.

---

## 1. Migration System Architecture

### 1.1 Version Tracking

The migration system uses a `schema_migrations` table to track applied migrations:

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    checksum VARCHAR(64) NOT NULL,
    UNIQUE(version, name)
);

CREATE INDEX idx_schema_migrations_version ON schema_migrations(version);
```

**Key Features:**
- **Version Number**: Sequential integer (1, 2, 3, ...) for ordering migrations
- **Name**: Human-readable migration name for identification
- **Applied At**: Timestamp when migration was executed
- **Checksum**: SHA-256 hash of migration SQL for integrity validation

### 1.2 Validation on Startup

The bot performs the following validation steps on startup:

1. **Connection Check**: Verify database connectivity
2. **Schema Migrations Table Check**: Ensure `schema_migrations` table exists (create if missing)
3. **Migration Integrity Check**: 
   - Read all migration files from `migrations/` directory
   - Calculate checksums for each migration file
   - Compare against stored checksums in `schema_migrations` table
   - Fail startup if any checksum mismatch detected (indicates manual database modification)
4. **Pending Migrations Check**:
   - Identify migrations with version numbers higher than the highest applied version
   - Apply pending migrations in sequential order
   - Log each migration application
5. **Rollback Prevention**: Once a migration is applied, it cannot be rolled back (immutable history)

### 1.3 Migration Execution Flow

```
Startup Sequence:
1. Connect to database
2. Ensure schema_migrations table exists
3. Scan migrations/ directory for .sql files
4. Sort migrations by version number
5. For each migration:
   a. Check if already applied (version exists in schema_migrations)
   b. If not applied:
      - Calculate checksum
      - Execute migration SQL within transaction
      - Insert record into schema_migrations
      - Commit transaction
6. Validate all migrations match stored checksums
7. Continue with application startup
```

---

## 2. Expandable Schema Design

### 2.1 Table-Per-Feature Pattern

Each challenge requirement gets its own dedicated table, enabling:
- **Isolated tracking**: Each requirement tracked independently
- **Easy extensibility**: Add new requirements without modifying existing tables
- **Flexible queries**: Query specific requirements or combine as needed
- **Independent validation**: Each table can have its own validation logic

### 2.2 Core Tables Structure

Each feature table follows a consistent pattern:

```sql
-- Template for feature tables
CREATE TABLE {feature_name} (
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    metadata JSONB,  -- Optional: store additional context (photo URL, notes, etc.)
    PRIMARY KEY (user_id, challenge_day),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1)  -- No upper limit due to failure extensions
);
```

**Benefits:**
- Consistent structure across all feature tables
- Easy to add new features by creating new tables
- Metadata field allows future extensibility without schema changes
- Foreign key ensures referential integrity

---

## 3. Primary Key Strategy Using Discord User ID

### 3.1 User Identification

Discord User IDs are 18-digit snowflake identifiers (stored as VARCHAR(20) for safety):

```sql
CREATE TABLE users (
    user_id VARCHAR(20) PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    discriminator VARCHAR(4),
    avatar_url TEXT,
    challenge_start_date DATE NOT NULL,
    challenge_end_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CHECK (challenge_end_date > challenge_start_date)
);

CREATE INDEX idx_users_challenge_dates ON users(challenge_start_date, challenge_end_date);
```

### 3.2 Primary Key Usage

All feature tables use composite primary keys:
- **Primary Key**: `(user_id, challenge_day)`
- **Rationale**: 
  - Prevents duplicate entries for same user/day combination
  - Enables efficient queries by user or day
  - Natural partitioning key for large-scale deployments

### 3.3 Foreign Key Relationships

All feature tables reference `users` table:
- **Cascade Delete**: When a user is removed, all their challenge data is automatically cleaned up
- **Referential Integrity**: Ensures only valid users can have challenge entries

---

## 4. Migration File Structure and Naming Conventions

### 4.1 Directory Structure

```
migrations/
├── 0001_initial_schema.sql
├── 0002_add_exercise_tracking.sql
├── 0003_add_diet_tracking.sql
├── 0004_add_water_tracking.sql
├── 0005_add_self_improvement_tracking.sql
├── 0006_add_accountability_tracking.sql
├── 0007_add_finances_tracking.sql
├── 0008_add_auto_populate_trigger.sql
├── 0009_add_failure_tracking.sql
├── 0010_create_summary_view.sql
└── README.md
```

### 4.2 Naming Convention

Format: `{version}_{descriptive_name}.sql`

**Rules:**
- Version: Zero-padded 4-digit number (0001, 0002, ...)
- Descriptive name: Snake_case, lowercase, descriptive of the migration purpose
- Extension: Always `.sql`
- Examples:
  - `0001_initial_schema.sql`
  - `0015_add_analytics_table.sql`
  - `0023_add_notification_preferences.sql`

### 4.3 Migration File Template

Each migration file should follow this structure:

```sql
-- Migration: {version}_{descriptive_name}
-- Description: {Brief description of what this migration does}
-- Author: {Optional: author name}
-- Date: {Optional: creation date}

BEGIN;

-- Migration SQL statements here
-- Always use transactions for atomicity

-- Example:
CREATE TABLE IF NOT EXISTS users (
    user_id VARCHAR(20) PRIMARY KEY,
    -- ... other columns
);

COMMIT;
```

**Best Practices:**
- Always wrap in `BEGIN;` and `COMMIT;` for transaction safety
- Use `IF NOT EXISTS` clauses where appropriate to allow idempotent execution
- Include comments explaining the purpose
- Test migrations on a copy of production data before applying

---

## 5. Go Package Structure for Migrations

### 5.1 Package Organization

```
internal/
├── database/
│   ├── migrations/
│   │   ├── migrations.go          # Migration execution logic
│   │   ├── validator.go           # Checksum validation
│   │   └── scanner.go             # File system scanning
│   ├── connection.go              # Database connection management
│   └── models.go                  # Database models/types
└── ...
```

### 5.2 Core Migration Package (`internal/database/migrations`)

**migrations.go** - Main migration execution:

```go
package migrations

import (
    "database/sql"
    "embed"
    "fmt"
    "sort"
)

//go:embed *.sql
var migrationFiles embed.FS

type Migration struct {
    Version int
    Name    string
    SQL     string
    Checksum string
}

type Manager struct {
    db *sql.DB
}

func NewManager(db *sql.DB) *Manager {
    return &Manager{db: db}
}

func (m *Manager) EnsureMigrationsTable() error {
    // Create schema_migrations table if it doesn't exist
}

func (m *Manager) ScanMigrations() ([]Migration, error) {
    // Scan embedded migration files, parse version numbers, calculate checksums
}

func (m *Manager) GetAppliedMigrations() (map[int]Migration, error) {
    // Query schema_migrations table for already-applied migrations
}

func (m *Manager) ApplyPending(migrations []Migration) error {
    // Apply migrations in order, record in schema_migrations
}

func (m *Manager) ValidateChecksums() error {
    // Compare file checksums with stored checksums
}

func (m *Manager) Run() error {
    // Main entry point: ensure table, scan, validate, apply pending
}
```

**validator.go** - Checksum validation:

```go
package migrations

import (
    "crypto/sha256"
    "encoding/hex"
    "fmt"
)

func CalculateChecksum(content []byte) string {
    hash := sha256.Sum256(content)
    return hex.EncodeToString(hash[:])
}

func ValidateMigration(migration Migration, storedChecksum string) error {
    calculated := CalculateChecksum([]byte(migration.SQL))
    if calculated != storedChecksum {
        return fmt.Errorf("checksum mismatch for migration %d: expected %s, got %s",
            migration.Version, storedChecksum, calculated)
    }
    return nil
}
```

**scanner.go** - File system scanning:

```go
package migrations

import (
    "embed"
    "fmt"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
)

func ScanMigrationFiles(fs embed.FS) ([]Migration, error) {
    // Read all .sql files from embedded filesystem
    // Parse version numbers from filenames
    // Sort by version number
    // Return sorted list of migrations
}
```

### 5.3 Integration with Main Application

**internal/database/connection.go**:

```go
package database

import (
    "database/sql"
    "fmt"
    _ "github.com/lib/pq" // PostgreSQL driver
    
    "github.com/75-hard-discord-bot/internal/database/migrations"
)

func Connect(dsn string) (*sql.DB, error) {
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }
    
    // Run migrations on startup
    mgr := migrations.NewManager(db)
    if err := mgr.Run(); err != nil {
        return nil, fmt.Errorf("migration failed: %w", err)
    }
    
    return db, nil
}
```

---

## 6. Initial Schema with Tables for Each 75 Half Chub for Dads Requirement

### 6.1 The 75 Half Chub for Dads Challenge Requirements

The challenge tracks six core feats:

1. **Daily Movement**: 
   - 1 workout per day (30+ minutes, indoor or outdoor)
   - Walking counts only if wearing a weight vest (intentional pace)
   - + 10 minutes daily of core or mobility (abs, planks, stretching, yoga; pre or post)

2. **Diet**: 
   - Follow one defined diet
   - No cheat meals
   - No alcohol

3. **Water**: 
   - Drink 1 gallon of plain water daily

4. **Self-Improvement**: 
   - 30 minutes per day of intentional self-improvement
   - Reading, learning, journaling, studying, skill-building
   - Zero screen time if done before bed

5. **Accountability**: 
   - Weekly progress photo
   - Daily check-in with champ / the boys

6. **Finances**: 
   - Spending limited to necessities only
   - No impulse purchases, luxury items, or discretionary spending

**Failure Rule**: Miss a day → add 7 days to the end (no restart, no stacking penalties)

**Council Exception Rule**: Missed days may be forgiven only by Council approval (champ / the boys). Approval must be explicit and within 24 hours. Forgiveness waives the +7 day penalty but does not erase the miss.

### 6.2 Initial Schema Migration (`0001_initial_schema.sql`)

```sql
-- Migration: 0001_initial_schema
-- Description: Creates initial schema with users table and schema_migrations tracking

BEGIN;

-- Schema migrations tracking table
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    checksum VARCHAR(64) NOT NULL,
    PRIMARY KEY (version, name)
);

CREATE INDEX IF NOT EXISTS idx_schema_migrations_version 
    ON schema_migrations(version);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    user_id VARCHAR(20) PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    discriminator VARCHAR(4),
    avatar_url TEXT,
    challenge_start_date DATE NOT NULL,
    original_challenge_end_date DATE NOT NULL,  -- Original 75-day end date
    current_challenge_end_date DATE NOT NULL,   -- Current end date (may be extended due to failures)
    days_added INTEGER DEFAULT 0,               -- Total days added due to failures
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CHECK (current_challenge_end_date >= original_challenge_end_date),
    CHECK (original_challenge_end_date > challenge_start_date)
);

CREATE INDEX IF NOT EXISTS idx_users_challenge_dates 
    ON users(challenge_start_date, current_challenge_end_date);

COMMIT;
```

### 6.3 Auto-Population on Check-In

**Key Design Decision**: When a user checks the box (reacts with ✅ emoji) in Discord, a single insert into `accountability_checkins` automatically triggers inserts into all other feat tables with default values. This means:

- ✅ **User Experience**: One simple action (checking the box) marks all feats complete
- ✅ **Default Values**: All tables populate with sensible defaults:
  - Exercise: 30 min workout, 10 min core/mobility
  - Diet: No cheat meals, no alcohol (enforced by CHECK constraints)
  - Water: 128 oz (1 gallon) plain water
  - Self-Improvement: 30 minutes
  - Finances: Compliant status
- ✅ **No User Input Required**: Users don't need to fill out forms or provide details
- ✅ **Optional Updates**: Users can later update specific feats with actual values if desired
- ✅ **Database Trigger**: Automatic population handled by PostgreSQL trigger function

**Implementation**: See migration `0008_add_auto_populate_trigger.sql` for the trigger function that handles this.

### 6.4 Feature Tables Migrations

**0002_add_exercise_tracking.sql**:

```sql
-- Migration: 0002_add_exercise_tracking
-- Description: Creates table for tracking daily movement (workout + core/mobility)

BEGIN;

CREATE TABLE IF NOT EXISTS exercise_completions (
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    -- Main workout (30+ minutes required)
    workout_duration_minutes INTEGER DEFAULT 30,  -- Default minimum requirement
    workout_type VARCHAR(100) DEFAULT 'general',  -- Default type
    workout_location VARCHAR(50) DEFAULT 'indoor',  -- Default location
    weight_vest_used BOOLEAN DEFAULT false,  -- Required if walking
    -- Core/mobility (10 minutes required)
    core_mobility_duration_minutes INTEGER DEFAULT 10,  -- Default minimum requirement
    core_mobility_type VARCHAR(100) DEFAULT 'general',  -- Default type
    notes TEXT,
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_day),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1),
    CHECK (workout_duration_minutes IS NULL OR workout_duration_minutes >= 30),
    CHECK (core_mobility_duration_minutes IS NULL OR core_mobility_duration_minutes >= 10)
);

CREATE INDEX IF NOT EXISTS idx_exercise_completions_user_day 
    ON exercise_completions(user_id, challenge_day);

COMMIT;
```

**0003_add_diet_tracking.sql**:

```sql
-- Migration: 0003_add_diet_tracking
-- Description: Creates table for tracking daily diet compliance (one defined diet, no cheat meals, no alcohol)

BEGIN;

CREATE TABLE IF NOT EXISTS diet_completions (
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    diet_type VARCHAR(100),  -- User's chosen diet (e.g., 'keto', 'paleo', 'calorie_deficit', etc.)
    cheat_meal BOOLEAN DEFAULT false,  -- Must be false for compliance
    alcohol_consumed BOOLEAN DEFAULT false,  -- Must be false for compliance
    notes TEXT,
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_day),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1),
    CHECK (cheat_meal = false),  -- Enforce no cheat meals
    CHECK (alcohol_consumed = false)  -- Enforce no alcohol
);

CREATE INDEX IF NOT EXISTS idx_diet_completions_user_day 
    ON diet_completions(user_id, challenge_day);

COMMIT;
```

**0004_add_water_tracking.sql**:

```sql
-- Migration: 0004_add_water_tracking
-- Description: Creates table for tracking daily water intake (1 gallon plain water required)

BEGIN;

CREATE TABLE IF NOT EXISTS water_completions (
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    amount_ounces DECIMAL(5, 2) DEFAULT 128.00,  -- Default: 1 gallon (128 ounces)
    is_plain_water BOOLEAN DEFAULT true,  -- Must be plain water (no additives, flavoring, etc.)
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_day),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1),
    CHECK (amount_ounces IS NULL OR amount_ounces >= 0),
    CHECK (is_plain_water = true)  -- Enforce plain water only
);

CREATE INDEX IF NOT EXISTS idx_water_completions_user_day 
    ON water_completions(user_id, challenge_day);

COMMIT;
```

**0005_add_self_improvement_tracking.sql**:

```sql
-- Migration: 0005_add_self_improvement_tracking
-- Description: Creates table for tracking daily self-improvement (30 minutes required)

BEGIN;

CREATE TABLE IF NOT EXISTS self_improvement_completions (
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    duration_minutes INTEGER NOT NULL DEFAULT 30,  -- Default minimum requirement
    activity_type VARCHAR(100) DEFAULT 'general',  -- Default type
    description TEXT,
    completed_before_bed BOOLEAN DEFAULT false,  -- If true, must have zero screen time
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_day),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1),
    CHECK (duration_minutes >= 30)
);

CREATE INDEX IF NOT EXISTS idx_self_improvement_completions_user_day 
    ON self_improvement_completions(user_id, challenge_day);

COMMIT;
```

**0006_add_accountability_tracking.sql**:

```sql
-- Migration: 0006_add_accountability_tracking
-- Description: Creates tables for tracking accountability (daily check-in + weekly progress photo)

BEGIN;

-- Daily check-in tracking (primary accountability mechanism)
-- When a user checks in, this triggers automatic population of all feat tables
CREATE TABLE IF NOT EXISTS accountability_checkins (
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    check_in_method VARCHAR(50) DEFAULT 'emoji_reaction',  -- e.g., 'emoji_reaction', 'slash_command', 'message'
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_day),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1)
);

CREATE INDEX IF NOT EXISTS idx_accountability_checkins_user_day 
    ON accountability_checkins(user_id, challenge_day);

CREATE INDEX IF NOT EXISTS idx_accountability_checkins_date 
    ON accountability_checkins(completed_at);

-- Weekly progress photo tracking (once per week)
CREATE TABLE IF NOT EXISTS progress_photos (
    user_id VARCHAR(20) NOT NULL,
    challenge_week INTEGER NOT NULL,  -- Week number (1+ for standard 75 days, can extend)
    challenge_day INTEGER NOT NULL,    -- Day of the week when photo was taken
    photo_taken_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    photo_url TEXT,
    photo_storage_key VARCHAR(255),
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_week),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_week >= 1),
    CHECK (challenge_day >= 1)
);

CREATE INDEX IF NOT EXISTS idx_progress_photos_user_week 
    ON progress_photos(user_id, challenge_week);

COMMIT;
```

**0007_add_finances_tracking.sql**:

```sql
-- Migration: 0007_add_finances_tracking
-- Description: Creates table for tracking daily spending compliance (necessities only)

BEGIN;

CREATE TABLE IF NOT EXISTS finances_completions (
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    compliance_status VARCHAR(20) NOT NULL DEFAULT 'compliant',  -- 'compliant', 'non_compliant'
    -- Track any non-necessity spending (for accountability)
    non_necessity_spending DECIMAL(10, 2) DEFAULT 0,
    spending_category VARCHAR(100),  -- e.g., 'necessity', 'impulse', 'luxury', 'discretionary'
    notes TEXT,
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_day),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1),
    CHECK (compliance_status IN ('compliant', 'non_compliant')),
    CHECK (non_necessity_spending >= 0)
);

CREATE INDEX IF NOT EXISTS idx_finances_completions_user_day 
    ON finances_completions(user_id, challenge_day);

COMMIT;
```

**0008_add_auto_populate_trigger.sql**:

```sql
-- Migration: 0008_add_auto_populate_trigger
-- Description: Creates trigger function to automatically populate all feat tables when user checks in
-- This allows a single Discord checkbox/emoji reaction to mark all feats as complete with defaults

BEGIN;

-- Function to auto-populate all feat tables when accountability check-in is recorded
CREATE OR REPLACE FUNCTION auto_populate_feats_on_checkin()
RETURNS TRIGGER AS $$
BEGIN
    -- Insert exercise completion with defaults
    INSERT INTO exercise_completions (user_id, challenge_day, completed_at)
    VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
    ON CONFLICT (user_id, challenge_day) DO NOTHING;

    -- Insert diet completion with defaults
    INSERT INTO diet_completions (user_id, challenge_day, completed_at)
    VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
    ON CONFLICT (user_id, challenge_day) DO NOTHING;

    -- Insert water completion with defaults
    INSERT INTO water_completions (user_id, challenge_day, completed_at)
    VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
    ON CONFLICT (user_id, challenge_day) DO NOTHING;

    -- Insert self-improvement completion with defaults
    INSERT INTO self_improvement_completions (user_id, challenge_day, completed_at)
    VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
    ON CONFLICT (user_id, challenge_day) DO NOTHING;

    -- Insert finances completion with defaults
    INSERT INTO finances_completions (user_id, challenge_day, completed_at)
    VALUES (NEW.user_id, NEW.challenge_day, NEW.completed_at)
    ON CONFLICT (user_id, challenge_day) DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger that fires after insert on accountability_checkins
CREATE TRIGGER trigger_auto_populate_feats
    AFTER INSERT ON accountability_checkins
    FOR EACH ROW
    EXECUTE FUNCTION auto_populate_feats_on_checkin();

COMMIT;
```

**0009_add_failure_tracking.sql**:

```sql
-- Migration: 0009_add_failure_tracking
-- Description: Creates tables for tracking missed days, failures, and council exceptions

BEGIN;

-- Track days where user failed to complete all requirements
CREATE TABLE IF NOT EXISTS challenge_failures (
    failure_id SERIAL PRIMARY KEY,
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    failed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    failed_feats TEXT[],  -- Array of feats that were failed: ['exercise', 'diet', 'water', etc.]
    days_added INTEGER DEFAULT 7,  -- Days added to challenge (default 7, can be waived by council)
    council_forgiven BOOLEAN DEFAULT false,
    council_forgiven_at TIMESTAMP WITH TIME ZONE,
    council_forgiven_by VARCHAR(20),  -- User ID of council member who approved
    notes TEXT,
    metadata JSONB,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1),
    CHECK (days_added >= 0),
    UNIQUE(user_id, challenge_day)
);

CREATE INDEX IF NOT EXISTS idx_challenge_failures_user_day 
    ON challenge_failures(user_id, challenge_day);

CREATE INDEX IF NOT EXISTS idx_challenge_failures_forgiven 
    ON challenge_failures(council_forgiven);

-- Track council exception approvals (for audit trail)
CREATE TABLE IF NOT EXISTS council_exceptions (
    exception_id SERIAL PRIMARY KEY,
    failure_id INTEGER NOT NULL REFERENCES challenge_failures(failure_id) ON DELETE CASCADE,
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    requested_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    approved_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    approved_by VARCHAR(20) NOT NULL,  -- User ID of council member
    reason TEXT NOT NULL,  -- Reason for exception (illness, injury, family emergency, etc.)
    approved_within_24h BOOLEAN NOT NULL,  -- Must be approved within 24 hours
    metadata JSONB,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (approved_at <= requested_at + INTERVAL '24 hours')
);

CREATE INDEX IF NOT EXISTS idx_council_exceptions_user_day 
    ON council_exceptions(user_id, challenge_day);

CREATE INDEX IF NOT EXISTS idx_council_exceptions_approved_by 
    ON council_exceptions(approved_by);

COMMIT;
```

### 6.5 Summary View (Optional Helper)

**0010_create_summary_view.sql**:

```sql
-- Migration: 0010_create_summary_view
-- Description: Creates a materialized view for quick progress summaries across all feats

BEGIN;

CREATE MATERIALIZED VIEW IF NOT EXISTS user_daily_progress AS
SELECT 
    u.user_id,
    u.username,
    u.challenge_start_date,
    u.current_challenge_end_date,
    u.days_added,
    day.day_number,
    CASE WHEN e.user_id IS NOT NULL THEN true ELSE false END AS exercise_completed,
    CASE WHEN d.user_id IS NOT NULL THEN true ELSE false END AS diet_completed,
    CASE WHEN w.user_id IS NOT NULL THEN true ELSE false END AS water_completed,
    CASE WHEN si.user_id IS NOT NULL THEN true ELSE false END AS self_improvement_completed,
    CASE WHEN a.user_id IS NOT NULL THEN true ELSE false END AS accountability_checkin_completed,
    CASE WHEN f.user_id IS NOT NULL AND f.compliance_status = 'compliant' THEN true ELSE false END AS finances_completed,
    -- Check if day was failed
    CASE WHEN cf.failure_id IS NOT NULL THEN true ELSE false END AS day_failed,
    CASE WHEN cf.council_forgiven THEN true ELSE false END AS failure_forgiven,
    -- Overall completion: all feats completed for the day
    CASE WHEN 
        e.user_id IS NOT NULL AND
        d.user_id IS NOT NULL AND
        w.user_id IS NOT NULL AND
        si.user_id IS NOT NULL AND
        a.user_id IS NOT NULL AND
        f.user_id IS NOT NULL AND
        f.compliance_status = 'compliant' AND
        cf.failure_id IS NULL
    THEN true ELSE false END AS all_feats_completed
FROM users u
CROSS JOIN generate_series(1, (SELECT MAX(current_challenge_end_date - challenge_start_date) FROM users)) AS day(day_number)
LEFT JOIN exercise_completions e ON e.user_id = u.user_id AND e.challenge_day = day.day_number
LEFT JOIN diet_completions d ON d.user_id = u.user_id AND d.challenge_day = day.day_number
LEFT JOIN water_completions w ON w.user_id = u.user_id AND w.challenge_day = day.day_number
LEFT JOIN self_improvement_completions si ON si.user_id = u.user_id AND si.challenge_day = day.day_number
LEFT JOIN accountability_checkins a ON a.user_id = u.user_id AND a.challenge_day = day.day_number
LEFT JOIN finances_completions f ON f.user_id = u.user_id AND f.challenge_day = day.day_number
LEFT JOIN challenge_failures cf ON cf.user_id = u.user_id AND cf.challenge_day = day.day_number
WHERE day.day_number <= (u.current_challenge_end_date - u.challenge_start_date);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_daily_progress_user_day 
    ON user_daily_progress(user_id, day_number);

-- Refresh function for the materialized view
CREATE OR REPLACE FUNCTION refresh_user_daily_progress()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY user_daily_progress;
END;
$$ LANGUAGE plpgsql;

COMMIT;
```

---

## 7. Future Extensibility Considerations

### 7.1 Adding New Challenge Requirements

To add a new requirement:

1. **Create Migration File**: `00XX_add_new_feature_tracking.sql`
2. **Follow Pattern**: Use the same table structure as existing feature tables
3. **Update Views**: If using summary views, update them to include the new feature
4. **No Breaking Changes**: New tables don't affect existing functionality

**Example - Adding Meditation Tracking**:

```sql
-- Migration: 0010_add_meditation_tracking.sql
CREATE TABLE IF NOT EXISTS meditation_completions (
    user_id VARCHAR(20) NOT NULL,
    challenge_day INTEGER NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    duration_minutes INTEGER,
    metadata JSONB,
    PRIMARY KEY (user_id, challenge_day),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (challenge_day >= 1)  -- No upper limit due to failure extensions
);
```

### 7.2 Schema Evolution Strategies

**Adding Columns to Existing Tables**:

```sql
-- Migration: 00XX_add_column_to_existing_table.sql
ALTER TABLE diet_completions 
ADD COLUMN IF NOT EXISTS meal_plan_type VARCHAR(50);
```

**Adding Indexes for Performance**:

```sql
-- Migration: 00XX_add_performance_indexes.sql
CREATE INDEX IF NOT EXISTS idx_diet_completions_date_range 
    ON diet_completions(user_id, challenge_day) 
    WHERE challenge_day BETWEEN 1 AND 75;
```

**Partitioning for Scale** (Future):

```sql
-- Migration: 00XX_partition_by_challenge_day.sql
-- Partition large tables by challenge_day ranges for better performance
```

### 7.3 Metadata Field Strategy

The `metadata JSONB` field in each table allows storing:
- Additional context without schema changes
- Feature flags for A/B testing
- Integration data (e.g., Strava workout IDs)
- Custom user preferences
- Audit trail information

**Example Usage**:

```sql
-- Store additional context
UPDATE diet_completions 
SET metadata = '{"meal_type": "keto", "calories": 2000, "notes": "Felt great today"}'::jsonb
WHERE user_id = '123456789' AND challenge_day = 5;
```

### 7.4 Performance Considerations

**Current Design Supports**:
- Efficient queries by user_id (indexed)
- Efficient queries by challenge_day (indexed)
- Fast lookups for daily completion status
- Scalable to thousands of users

**Future Optimizations**:
- **Partitioning**: Partition tables by challenge_day ranges for very large datasets
- **Archiving**: Archive completed challenges older than 1 year
- **Read Replicas**: Use read replicas for analytics queries
- **Caching**: Cache frequently accessed summary data

### 7.5 Multi-Challenge Support (Future)

To support multiple challenge types (75 Hard, 75 Soft, custom challenges):

```sql
-- Migration: 00XX_add_challenge_types.sql
CREATE TABLE IF NOT EXISTS challenge_types (
    challenge_type_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    duration_days INTEGER NOT NULL,
    requirements JSONB NOT NULL
);

ALTER TABLE users 
ADD COLUMN challenge_type_id INTEGER REFERENCES challenge_types(challenge_type_id);

-- Update feature tables to be challenge-agnostic
ALTER TABLE diet_completions 
ADD COLUMN challenge_type_id INTEGER REFERENCES challenge_types(challenge_type_id);
```

### 7.6 Analytics and Reporting Tables (Future)

```sql
-- Migration: 00XX_add_analytics_tables.sql
CREATE TABLE IF NOT EXISTS daily_statistics (
    stat_date DATE PRIMARY KEY,
    total_active_users INTEGER,
    total_completions INTEGER,
    average_completion_rate DECIMAL(5, 2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_statistics (
    user_id VARCHAR(20) PRIMARY KEY,
    total_days_completed INTEGER DEFAULT 0,
    current_streak INTEGER DEFAULT 0,
    longest_streak INTEGER DEFAULT 0,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);
```

---

## 8. Migration Execution Best Practices

### 8.1 Development Workflow

1. **Create Migration File**: Write SQL in `migrations/00XX_feature_name.sql`
2. **Test Locally**: Run migrations against local database
3. **Validate**: Ensure checksums match after execution
4. **Commit**: Include migration file in version control
5. **Deploy**: Migrations run automatically on application startup

### 8.2 Production Deployment

1. **Backup First**: Always backup production database before migrations
2. **Test in Staging**: Run migrations in staging environment first
3. **Monitor**: Watch for migration execution logs during deployment
4. **Rollback Plan**: Have a plan for manual intervention if migrations fail
5. **Verification**: Verify data integrity after migration completes

### 8.3 Error Handling

- **Transaction Safety**: All migrations wrapped in transactions
- **Idempotent Operations**: Use `IF NOT EXISTS` clauses where possible
- **Validation**: Checksum validation prevents corrupted migrations
- **Logging**: Log all migration operations for audit trail

---

## 9. Conclusion

This migration system provides:

✅ **Version Control**: Track all schema changes  
✅ **Validation**: Ensure database integrity on startup  
✅ **Extensibility**: Easy to add new features  
✅ **Safety**: Transaction-based, checksum-validated migrations  
✅ **Scalability**: Designed for growth  
✅ **Maintainability**: Clear structure and conventions  

The system is ready for initial implementation and can evolve with the application's needs.
