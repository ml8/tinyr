package db

import (
	"log/slog"
	"os"
	"path"
	"sort"
	"strings"
)

type Schema struct {
	Schema   string
	Filename string
}

type MigrationArgs struct {
	Logger    *slog.Logger
	SchemaDir string
	Basename  string
	Extension string
	DryRun    bool
	Migrator  Migrator
}

type Migrator interface {
	InitDB() (err error)
	ApplySchema(schema Schema, dry_run bool) (err error)
	Complete()
}

func RunMigration(args MigrationArgs) (err error) {
	logger = args.Logger
	schemas, err := getSchemas(args.SchemaDir, args.Basename, args.Extension)
	if err != nil {
		return
	}

	// Connect to DB
	err = args.Migrator.InitDB()
	if err != nil {
		return
	}
	defer args.Migrator.Complete()

	err = applyAll(args.Migrator, args.SchemaDir, schemas, args.DryRun)
	if err != nil {
		return
	}

	return
}

func dot(s string) string {
	if !strings.HasPrefix(s, ".") {
		return "." + s
	}
	return s
}

// Get list of schema files in given directory, sorted according to apply order.
func getSchemas(dir string, basename string, extension string) (schemas []string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), basename+dot(extension)) {
			logger.Info("Ignoring " + e.Name())
			continue
		}

		logger.Info("Found " + e.Name())
		schemas = append(schemas, e.Name())
	}

	sort.Slice(schemas, func(i, j int) bool {
		// make sure that base schema comes first. rest are sorted
		// lexicographically.
		if schemas[i] == basename+extension {
			return true
		} else if schemas[j] == basename+extension {
			return false
		}

		return schemas[i] < schemas[j]
	})
	return
}

func applyAll(m Migrator, dir string, files []string, dry_run bool) (err error) {
	var schemas []Schema
	for _, fn := range files {
		var b []byte
		b, err = os.ReadFile(path.Join(dir, fn))
		if err != nil {
			logger.Error("Error parsing", "filename", fn, "error", err)
			return
		}
		schemas = append(schemas, Schema{Schema: string(b), Filename: fn})
	}

	// now apply
	for _, s := range schemas {
		logger.Info("applying", "file", s.Filename)
		err = m.ApplySchema(s, dry_run)
		if err != nil {
			return
		}
	}
	return
}
