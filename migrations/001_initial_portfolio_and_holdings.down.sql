-- Migration: Drop portfolios and holdings tables
-- Rollback: Remove initial schema

-- Drop indexes
DROP INDEX IF EXISTS idx_holdings_portfolio_id;
DROP INDEX IF EXISTS idx_portfolios_address;

-- Drop tables (order matters due to foreign key)
DROP TABLE IF EXISTS holdings;
DROP TABLE IF EXISTS portfolios;

