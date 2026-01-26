-- Migration: Create portfolios and holdings tables
-- Created: Initial migration

-- Create portfolios table
CREATE TABLE IF NOT EXISTS portfolios (
    id TEXT PRIMARY KEY,
    address TEXT UNIQUE NOT NULL,
    updated_at DATETIME NOT NULL
);

-- Create holdings table
CREATE TABLE IF NOT EXISTS holdings (
    id TEXT PRIMARY KEY,
    portfolio_id TEXT NOT NULL,
    chain_id INTEGER NOT NULL,
    token_id TEXT NOT NULL,
    token_symbol TEXT NOT NULL,
    token_address TEXT NOT NULL,
    amount TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (portfolio_id) REFERENCES portfolios(id) ON DELETE CASCADE
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_portfolios_address ON portfolios(address);
CREATE INDEX IF NOT EXISTS idx_holdings_portfolio_id ON holdings(portfolio_id);

