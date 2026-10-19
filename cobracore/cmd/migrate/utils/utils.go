package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	migrate "github.com/golang-migrate/migrate/v4"
	/*
		// these must be included in your final app.
		_ "github.com/golang-migrate/migrate/v4/database/cassandra"
		_ "github.com/golang-migrate/migrate/v4/database/mongodb"
		_ "github.com/golang-migrate/migrate/v4/database/mysql"
		_ "github.com/golang-migrate/migrate/v4/database/postgres"
		_ "github.com/golang-migrate/migrate/v4/database/snowflake"
		_ "github.com/golang-migrate/migrate/v4/database/sqlserver"
		_ "github.com/golang-migrate/migrate/v4/source/aws_s3"
		_ "github.com/golang-migrate/migrate/v4/source/bitbucket"
		_ "github.com/golang-migrate/migrate/v4/source/file"
		_ "github.com/golang-migrate/migrate/v4/source/github"
		_ "github.com/golang-migrate/migrate/v4/source/gitlab"
		_ "github.com/golang-migrate/migrate/v4/source/go_bindata"
		_ "github.com/golang-migrate/migrate/v4/source/google_cloud_storage"
		_ "github.com/golang-migrate/migrate/v4/source/httpfs"
	*/
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const relativePath = "./dbmigrate"

func GetDbMigratePath() string {
	var configPath string
	_, err := os.Stat(relativePath)
	if !os.IsNotExist(err) {
		absPath, absErr := filepath.Abs(relativePath)
		if absErr != nil {
			log.Error().Err(absErr).Str("path", relativePath).Msg("Failed to resolve absolute path")
			return ""
		}
		configPath = absPath
		log.Info().Str("path", configPath).Msg("Configuration Root Folder")
	}
	return configPath
}

var Source string
var StepN uint64
var Verbose bool

// isMongoURI reports whether connectionString is a MongoDB URI (standard or SRV).
func isMongoURI(connectionString string) bool {
	return strings.HasPrefix(connectionString, "mongodb://") || strings.HasPrefix(connectionString, "mongodb+srv://")
}

// fixupMongoConnectionString sets the database path of a MongoDB URI to databaseName:
//
//	mongodb://[user:pass@]host1[:port1][,host2[:port2]...][/[defaultdb]][?options]
//
// It used to find hosts with regexes (`word:digits`), which put the database inside the
// credentials when the password was numeric (user:1234@host), appended it after every host
// of a multi-host URI without options, turned an existing /olddb into /db/olddb, and was
// never applied to mongodb+srv:// URIs. It now splits the URI structurally: the authority
// runs to the first '/' after the userinfo, and any existing database is replaced.
func fixupMongoConnectionString(connectionString, databaseName string) string {
	scheme := ""
	for _, s := range []string{"mongodb+srv://", "mongodb://"} {
		if strings.HasPrefix(connectionString, s) {
			scheme = s
			break
		}
	}
	if scheme == "" {
		return connectionString
	}
	rest := strings.TrimPrefix(connectionString, scheme)

	query := ""
	if i := strings.Index(rest, "?"); i >= 0 {
		rest, query = rest[:i], rest[i:]
	}
	// userinfo ends at the last '@' (a password's '@' must be escaped, but be lenient).
	hostsStart := strings.LastIndex(rest, "@") + 1
	authority := rest
	if i := strings.Index(rest[hostsStart:], "/"); i >= 0 {
		authority = rest[:hostsStart+i]
	}
	return scheme + authority + "/" + databaseName + query
}

// upPersistentPreRunE validateds required flags
func UpPersistentPreRunE(cmd *cobra.Command, args []string) error {
	log.Info().Msg("Validating input")
	var err error
	connectionString := viper.GetString("database")
	if len(connectionString) == 0 {
		err = fmt.Errorf("--database missing and env:DATABASE not present")
		log.Error().Err(err).Send()
		return err
	}
	log.Info().Msg("Found --database")
	databaseName := viper.GetString("database-name")
	if len(databaseName) == 0 {
		log.Info().Msg("no --database-name present")
	} else {
		if isMongoURI(connectionString) {
			log.Info().Msg("Found --database == mongodb")
			connectionString = fixupMongoConnectionString(connectionString, databaseName)
			viper.Set("database", connectionString)
		}
	}
	verbose := viper.GetBool("verbose")
	if verbose {
		log.Info().Msg("Found --verbose")
		Verbose = verbose
	}
	connectionString = viper.GetString("database")

	Source = viper.GetString("source")
	if len(Source) == 0 {
		// check to see if we have an embedded dbmigrate folder
		dbMigratePath := GetDbMigratePath()
		_, err := os.ReadDir(dbMigratePath)
		if err != nil {
			log.Error().Err(err).Msg("no source and no embedded dbmigrate path")
			return err
		}
		log.Info().Str("dbMigratePath", dbMigratePath).Msg("Found local dbmigration scripts")

	}
	step := viper.GetString("step")

	if len(step) > 0 {
		log.Info().Str("step", step).Msg("Found --step")

		StepN, err = strconv.ParseUint(step, 10, 64)
		if err != nil {
			log.Error().Err(err).Str("step", step).
				Msg("Invalid step argument")
			return err
		}
	}

	return nil

}

type MigrateDirection int

const (
	MigrateUp MigrateDirection = iota
	MigrateDown
	MigrateForce
)

func Migrate(md MigrateDirection) error {
	log.Info().Interface("MigrationDirection", md).Msg("Migrating.....")
	var err error
	//	verbose := viper.GetBool("verbose")
	connectionString := viper.GetString("database")
	//	fmt.Println("inside up, verbose value: ", connectionString)

	var m *migrate.Migrate
	if len(Source) > 0 {
		log.Info().Msg("Migrating from remote source")
		m, err = migrate.New(Source, connectionString)
	} else {
		log.Info().Str("Source", "file://dbmigrate").Msg("Migrating from local source")
		m, err = migrate.New(
			"file://dbmigrate",
			connectionString)
		if err == nil && Verbose {
			// read all the files in the dbmigrate folder and print them out
			dbMigratePath := GetDbMigratePath()
			files, err := os.ReadDir(dbMigratePath)
			if err != nil {
				log.Error().Err(err).Msg("Failed to read dbmigrate folder")
				return err
			}
			fmt.Println("===============================================")
			fmt.Println("Contents of dbmigrate folder:")
			fmt.Println(dbMigratePath)
			fmt.Println("===============================================")

			for _, f := range files {

				// these are all text files, read them and print them out pretty like
				filePath := fmt.Sprintf("%s/%s", dbMigratePath, f.Name())
				// read the entire file into memory
				data, err := os.ReadFile(filePath)
				if err != nil {
					fmt.Println("File reading error", err)
					return err
				}
				fmt.Println(">----------------------------------------------")
				fmt.Println(f.Name())
				fmt.Println(">----------------------------------------------")

				fmt.Println(string(data))
			}
			fmt.Println("===============================================")
		}
	}
	if err != nil {
		log.Error().Err(err).Msg("Failed to migrate.New")
		return err
	}
	// Close releases the source and the database connection; m was never closed.
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			log.Warn().AnErr("sourceErr", srcErr).AnErr("databaseErr", dbErr).Msg("closing migrate")
		}
	}()

	err = run(m, md, StepN, viper.GetInt("version"))
	if err != nil && err != migrate.ErrNoChange {
		log.Error().Err(err).Msg("Failed to migrate")
	}

	if err == nil || err == migrate.ErrNoChange {
		err = nil
		/*
			serves no purpose
			sleepMinutes := viper.GetInt("sleep")
			log.Info().Str("sleep", fmt.Sprintf("%v", sleepMinutes)).Msg("Success")
			var sleepDuration time.Duration
			if sleepMinutes != 0 {
				if sleepMinutes < 0 {
					sleepMinutes = 60 * 24 * 365
				}
				sleepDuration = time.Minute * time.Duration(sleepMinutes)
				time.Sleep(sleepDuration)
			}
		*/
	}
	return err
}

// migrator is the part of *migrate.Migrate that run uses (an interface so it can be tested
// without a database).
type migrator interface {
	Up() error
	Down() error
	Steps(n int) error
	Force(version int) error
}

// run performs md. Force always forces the given version: --step used to take precedence
// over the direction, so `force --version N --step S` ran S *up* migrations instead.
func run(m migrator, md MigrateDirection, steps uint64, version int) error {
	switch md {
	case MigrateForce:
		return m.Force(version)
	case MigrateUp:
		if steps > 0 {
			return m.Steps(int(steps))
		}
		return m.Up()
	case MigrateDown:
		if steps > 0 {
			return m.Steps(-int(steps))
		}
		return m.Down()
	}
	return fmt.Errorf("no valid migrate direction")
}
