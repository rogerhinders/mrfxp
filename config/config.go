package config

import (
	"database/sql"
	_ "fmt"
	"github.com/mattn/go-sqlite3"
)

const DB_DRIVER = "sqlite_3"
const DB_NAME = "mrfxp.sql"

func RegDBDriver() {
	sql.Register(DB_DRIVER, &sqlite3.SQLiteDriver{})
}

type SiteConfig struct {
	Id       int
	Name     string
	Hostname string
	Port     int
	Tls      bool
	Username string
	Password string
}

type SectionConfig struct {
	Id   int
	Name string
}

type Bleh struct {
	hello string
	ello  string
}

type Config struct {
	db *sql.DB
}

func (c *Config) Init() error {
	database, err := sql.Open(DB_DRIVER, DB_NAME)

	if err != nil {
		return err
	}

	err = database.Ping()

	if err != nil {
		return err
	}

	c.db = database

	err = c.createTables()

	if err != nil {
		return err
	}

	return nil
}

func (c *Config) createTables() error {
	//sites
	_, err := c.db.Exec(
		`
    CREATE TABLE IF NOT EXISTS sites (
      id integer PRIMARY KEY AUTOINCREMENT,
      name varchar(255) NOT NULL,
      hostname varchar(255) NOT NULL,
      port integer DEFAULT '21',
      tls integer default '0',
      username varchar(255) NOT NULL,
      password varchar(255) NOT NULL
    );
    `,
	)

	if err != nil {
		return err
	}

	//sections
	_, err = c.db.Exec(
		`
    CREATE TABLE IF NOT EXISTS sections (
      id integer PRIMARY KEY AUTOINCREMENT,
      name varchar(255) NOT NULL
    );
    `,
	)

	if err != nil {
		return err
	}

	//site<-section relations
	_, err = c.db.Exec(
		`
    CREATE TABLE IF NOT EXISTS site_sections (
      id integer PRIMARY KEY AUTOINCREMENT,
      site_id integer default '0',
      section_id integer default '0'
    );
    `,
	)

	if err != nil {
		return err
	}
	return nil
}

func (c *Config) GetSites() ([]SiteConfig, error) {
	var sites []SiteConfig

	var id sql.NullInt64
	var name sql.NullString
	var hostname sql.NullString
	var port sql.NullInt64
	var tls sql.NullBool
	var username sql.NullString
	var password sql.NullString

	result, err := c.db.Query("SELECT * FROM sites")

	if err != nil {
		return nil, err
	}

	for result.Next() {
		err = result.Scan(&id, &name, &hostname, &port, &tls, &username, &password)

		if err != nil {
			return nil, err
		}

		sites = append(
			sites,
			SiteConfig{
				Id:       int(id.Int64),
				Name:     name.String,
				Hostname: hostname.String,
				Port:     int(port.Int64),
				Tls:      tls.Bool,
				Username: username.String,
				Password: password.String,
			},
		)
	}

	return sites, nil
}

func (c *Config) AddSite(name string, host string, port int, tls bool, username string, password string) error {
	var isTls int

	if tls {
		isTls = 1
	} else {
		isTls = 0
	}

	_, err := c.db.Exec(
		`
    INSERT INTO sites (name,hostname,port,tls,username,password) VALUES (?,?,?,?,?,?)
    `,
		name,
		host,
		port,
		isTls,
		username,
		password,
	)

	if err != nil {
		return err
	}

	return nil
}

func (c *Config) AddSection(name string) error {
	_, err := c.db.Exec(
		`
    INSERT INTO sections (name) VALUES (?)
    `,
		name,
	)

	if err != nil {
		return err
	}

	return nil
}

func (c *Config) GetSections() ([]SectionConfig, error) {
	var sections []SectionConfig

	var id sql.NullInt64
	var name sql.NullString

	result, err := c.db.Query("SELECT * FROM sections")

	if err != nil {
		return nil, err
	}

	for result.Next() {
		err = result.Scan(&id, &name)

		if err != nil {
			return nil, err
		}

		sections = append(
			sections,
			SectionConfig{
				Id:   int(id.Int64),
				Name: name.String,
			},
		)
	}

	return sections, nil
}
