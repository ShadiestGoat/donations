package db

// array of [2]string{SQL statement, context}
var setup = [][2]string{
	{sql_SETUP_DONORS, "creating the donor table"},
	{sql_SETUP_FUNDS, "creating the fund table"},
	{sql_SETUP_DONATIONS, "creating the donations table"},
}

const sql_SETUP_DONATIONS = `CREATE TABLE IF NOT EXISTS donations (
	id     TEXT PRIMARY KEY,
	order_id        TEXT UNIQUE,
	capture_id      TEXT UNIQUE,
	donor           TEXT,
	amount          NUMERIC(6, 2) NOT NULL,
	amount_received NUMERIC(6, 2) NOT NULL,
	message         VARCHAR(128),
	fund            TEXT,
	
	CONSTRAINT is_donor
		FOREIGN KEY(donor)
			REFERENCES donors(id),
	
	CONSTRAINT is_fund
		FOREIGN KEY(fund)
			REFERENCES funds(id)
)`

const sql_SETUP_DONORS = `CREATE TABLE IF NOT EXISTS donors (
	id          TEXT PRIMARY KEY,
	discord_id  TEXT,
	paypal      TEXT,
	cycle       NUMERIC(2, 0),
	CONSTRAINT uq_donor 
		UNIQUE(discord_id, paypal)
)`

const sql_SETUP_FUNDS = `CREATE TABLE IF NOT EXISTS funds (
	id TEXT PRIMARY KEY,
	def BOOLEAN DEFAULT false,
	goal NUMERIC(9, 2) DEFAULT 0,
	complete BOOLEAN DEFAULT false,
	
	alias TEXT UNIQUE NOT NULL,
	short_title TEXT DEFAULT '',
	description TEXT DEFAULT ''
)`
