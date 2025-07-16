package seeder

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var errProviderNotSet = fmt.Errorf("provider not set")

type SQLProvider struct {
	Dialect string `yaml:"dialect"`
	Host    string `yaml:"host"`
	Port    string `yaml:"port"`
	User    string `yaml:"user"`
	Pass    string `yaml:"pass"`
	DB      string `yaml:"db"`
	Secure  bool   `yaml:"secure"`
}

type Config struct {
	SQL     *SQLProvider `yaml:"sql"`
	Folders []string     `yaml:"folders"`
}

func (cfg *Config) Seed() error {
	switch {
	case cfg.SQL != nil:
		return cfg.seedSQL()
	default:
		return errProviderNotSet
	}
}

func (cfg *Config) seedSQL() (err error) {
	info := cfg.SQL
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s",
		info.Host, info.User, info.Pass, info.DB, info.Port,
	)

	if info.Secure {
		dsn += " sslmode=require"
	} else {
		dsn += " sslmode=disable"
	}

	var db *sql.DB

	switch strings.ToLower(cfg.SQL.Dialect) {
	case "postgres", "pg":
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			return err
		}

		for {
			err = db.Ping()
			if err != nil {
				time.Sleep(time.Second)
				fmt.Println("error on ping: ", err)

				continue
			}

			break
		}
	}

	for _, folder := range cfg.Folders {
		_ = filepath.WalkDir(
			folder,
			func(path string, d fs.DirEntry, err error) error {
				if d.IsDir() {
					return nil
				}

				sqlData, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				_, err = db.Exec(string(sqlData))
				if err != nil {
					fmt.Println("got error on seeding:", path, err)

					return nil
				}

				return nil
			},
		)
	}

	return nil
}
