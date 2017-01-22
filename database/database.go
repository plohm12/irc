//TODO move login info to config file
//TODO remove user_channel table and use lookups instead
//TODO encrypt passwords

package database

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

// Session ID, obtained by creating a new user record
type Id int64

/* Database Definitions */
const DB_DRIVER string = "mysql"
const DB_USER string = "root"
const DB_PASS string = "root"
const DB_NAME string = "irc"
const TABLE_USERS string = DB_NAME + ".users"
const TABLE_CHANNELS string = DB_NAME + ".channels"
const TABLE_USER_CHANNEL string = DB_NAME + ".user_channel"
const DB_DATASOURCE string = DB_USER + ":" + DB_PASS + "@/"

var (
	db *sql.DB
)

// Initialize database tables.
func Create() {
	var err error
	db, err = sql.Open(DB_DRIVER, DB_DATASOURCE)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS " + DB_NAME)
	if err != nil {
		panic(err)
	}

	createTables()
	prepareUserStatements()
	prepareChannelStatements()
}

func createTables() {
	var err error
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS " + TABLE_USERS + " (" +
		"id INT(10) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY," +
		"username VARCHAR(20) DEFAULT ''," +
		"nickname VARCHAR(20) DEFAULT ''," +
		"password VARCHAR(20) DEFAULT ''," +
		"mode INT(10) NOT NULL DEFAULT 0," +
		"realname VARCHAR(30) DEFAULT '')")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS " + TABLE_CHANNELS + " (" +
		"channel_name VARCHAR(50) NOT NULL PRIMARY KEY," +
		"creator VARCHAR(20) DEFAULT ''," +
		"topic VARCHAR(128) DEFAULT '')")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS " + TABLE_USER_CHANNEL + " (" +
		"user_id INT(10) UNSIGNED NOT NULL," +
		"channel_name VARCHAR(50) NOT NULL," +
		"PRIMARY KEY(user_id, channel_name))")
	if err != nil {
		panic(err)
	}
}

// Drops all tables and the database.
func Destroy() {
	var err error

	closeUserStatements()
	closeChannelStatements()
	destroyTables()

	_, err = db.Exec("DROP DATABASE " + DB_NAME)
	if err != nil {
		panic(err)
	}
	db.Close()
}

func destroyTables() {
	var err error
	_, err = db.Exec("DROP TABLE " + TABLE_USER_CHANNEL)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("DROP TABLE " + TABLE_CHANNELS)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("DROP TABLE " + TABLE_USERS)
	if err != nil {
		panic(err)
	}
}
