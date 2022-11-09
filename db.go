package main

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var DB *pgxpool.Pool

const SQL_SETUP_DONATIONS = `CREATE TABLE IF NOT EXISTS donations (
id     TEXT PRIMARY KEY,
order_id        TEXT UNIQUE,
capture_id      TEXT UNIQUE,
donor           TEXT,
amount          NUMERIC(6, 2),
amount_received NUMERIC(6, 2),
message         VARCHAR(128),
fund            TEXT,

CONSTRAINT is_donor
	FOREIGN KEY(donor)
		REFERENCES donors(id),

CONSTRAINT is_fund
	FOREIGN KEY(fund)
		REFERENCES funds(id)
)`

const SQL_SETUP_DONORS = `CREATE TABLE IF NOT EXISTS donors (
id          TEXT PRIMARY KEY,
discord_id  TEXT,
paypal      TEXT,
cycle       NUMERIC(2, 0),
CONSTRAINT uq_donor 
	UNIQUE(discord_id, paypal)
)`

const SQL_SETUP_FUNDS = `CREATE TABLE IF NOT EXISTS funds (
id TEXT PRIMARY KEY,
def BOOLEAN DEFAULT false,
goal NUMERIC(9, 2) DEFAULT 0,
complete BOOLEAN DEFAULT false,

alias TEXT DEFAULT '',
short_title TEXT DEFAULT '',
description TEXT DEFAULT ''
)`

func InitDB() {
	conf, err := pgxpool.ParseConfig(DB_URI)
	PanicIfErr(err)

	db, err := pgxpool.ConnectConfig(context.Background(), conf)
	PanicIfErr(err)

	err = db.Ping(context.Background())
	PanicIfErr(err)

	DB = db

	_, err = DBExec(SQL_SETUP_DONORS)
	PanicIfErr(err)

	_, err = DBExec(SQL_SETUP_FUNDS)
	PanicIfErr(err)

	_, err = DBExec(SQL_SETUP_DONATIONS)
	PanicIfErr(err)

}

func DBExec(sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return DB.Exec(context.Background(), sql, args...)
}

func DBQuery(sql string, args ...interface{}) (pgx.Rows, error) {
	return DB.Query(context.Background(), sql, args...)
}

func DBQueryRow(sql string, args ...interface{}) pgx.Row {
	return DB.QueryRow(context.Background(), sql, args...)
}
