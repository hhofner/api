package models

import (
	"code.vikunja.io/api/pkg/mail"
	"fmt"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"gopkg.in/testfixtures.v2"
	"os"
	"path/filepath"
	"testing"
)

// IsTesting is set to true when we're running tests.
// We don't have a good solution to test email sending yet, so we disable email sending when testing
var IsTesting bool

// MainTest creates the test engine
func MainTest(m *testing.M, pathToRoot string) {
	var err error
	fixturesDir := filepath.Join(pathToRoot, "models", "fixtures")
	if err = createTestEngine(fixturesDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating test engine: %v\n", err)
		os.Exit(1)
	}

	IsTesting = true

	// Start the pseudo mail queue
	mail.StartMailDaemon()

	// Create test database
	PrepareTestDatabase()

	os.Exit(m.Run())
}

func createTestEngine(fixturesDir string) error {
	var err error
	x, err = xorm.NewEngine("sqlite3", "file::memory:?cache=shared")
	//x, err = xorm.NewEngine("sqlite3", "db.db")
	if err != nil {
		return err
	}
	x.SetMapper(core.GonicMapper{})

	// Sync dat shit
	if err = x.StoreEngine("InnoDB").Sync2(tables...); err != nil {
		return fmt.Errorf("sync database struct error: %v", err)
	}

	// Show SQL-Queries if necessary
	if os.Getenv("UNIT_TESTS_VERBOSE") == "1" {
		x.ShowSQL(true)
	}

	return InitFixtures(&testfixtures.SQLite{}, fixturesDir)
}

// PrepareTestDatabase load test fixtures into test database
func PrepareTestDatabase() error {
	return LoadFixtures()
}