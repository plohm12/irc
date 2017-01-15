package irc

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

var (
	db *sql.DB
)

// Initialize database tables. Defer destroyDB() immediately following this
// function call.
func CreateDB() *sql.DB {
	var err error
	db, err = sql.Open(DB_DRIVER, DB_DATASOURCE)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS " + DB_NAME)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS " + TABLE_USERS + " (" +
		"id INT(10) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY," +
		"username VARCHAR(20) NOT NULL," +
		"nickname VARCHAR(20) NOT NULL," +
		"password VARCHAR(20) NOT NULL," +
		"mode INT(10) NOT NULL DEFAULT 0," +
		"realname VARCHAR(30) NOT NULL)")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS " + TABLE_CHANNELS + " (" +
		"channel_name VARCHAR(50) NOT NULL PRIMARY KEY," +
		"creator VARCHAR(20) NOT NULL," +
		"topic VARCHAR(128) NOT NULL)")
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
	return db
}

// Drops all tables and the database. DestroyDB should be deferred immediately
// after a call to CreateDB.
func DestroyDB() {
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
	_, err = db.Exec("DROP DATABASE " + DB_NAME)
	if err != nil {
		panic(err)
	}
}
