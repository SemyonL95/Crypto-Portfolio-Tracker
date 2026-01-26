-- Migration: Seed dummy data for portfolios and holdings
-- Created: Seed migration with sample data

-- Insert dummy portfolios
INSERT INTO portfolios (id, address, updated_at) VALUES
    ('550e8400-e29b-41d4-a716-446655440000', '0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb', datetime('now')),
    ('550e8400-e29b-41d4-a716-446655440001', '0x8ba1f109551bD432803012645Hac136c22C19b00', datetime('now')),
    ('550e8400-e29b-41d4-a716-446655440002', '0x1234567890123456789012345678901234567890', datetime('now'))
ON CONFLICT(id) DO NOTHING;

-- Insert dummy holdings for portfolio 1 (Ethereum mainnet)
INSERT INTO holdings (id, portfolio_id, chain_id, token_id, token_symbol, token_address, amount, created_at, updated_at) VALUES
    ('650e8400-e29b-41d4-a716-446655440000', '550e8400-e29b-41d4-a716-446655440000', 1, 'ethereum', 'ETH', '0x0000000000000000000000000000000000000000', '5000000000000000000', datetime('now', '-7 days'), datetime('now')),
    ('650e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440000', 1, 'bitcoin', 'BTC', '0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599', '100000000', datetime('now', '-5 days'), datetime('now')),
    ('650e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440000', 1, 'usd-coin', 'USDC', '0xA0b86991c6218b36c1d5194A4C4C4C4C4C4C4C4C4', '5000000000', datetime('now', '-3 days'), datetime('now'))
ON CONFLICT(id) DO NOTHING;

-- Insert dummy holdings for portfolio 2 (Multiple chains)
INSERT INTO holdings (id, portfolio_id, chain_id, token_id, token_symbol, token_address, amount, created_at, updated_at) VALUES
    ('650e8400-e29b-41d4-a716-446655440010', '550e8400-e29b-41d4-a716-446655440001', 1, 'ethereum', 'ETH', '0x0000000000000000000000000000000000000000', '10000000000000000000', datetime('now', '-10 days'), datetime('now')),
    ('650e8400-e29b-41d4-a716-446655440011', '550e8400-e29b-41d4-a716-446655440001', 56, 'binancecoin', 'BNB', '0x0000000000000000000000000000000000000000', '2000000000000000000', datetime('now', '-8 days'), datetime('now')),
    ('650e8400-e29b-41d4-a716-446655440012', '550e8400-e29b-41d4-a716-446655440001', 137, 'matic-network', 'MATIC', '0x0000000000000000000000000000000000000000', '50000000000000000000', datetime('now', '-6 days'), datetime('now')),
    ('650e8400-e29b-41d4-a716-446655440013', '550e8400-e29b-41d4-a716-446655440001', 1, 'chainlink', 'LINK', '0x514910771AF9Ca656af840dff83E8264EcF986CA', '100000000000000000000', datetime('now', '-4 days'), datetime('now'))
ON CONFLICT(id) DO NOTHING;

-- Insert dummy holdings for portfolio 3 (Smaller portfolio)
INSERT INTO holdings (id, portfolio_id, chain_id, token_id, token_symbol, token_address, amount, created_at, updated_at) VALUES
    ('650e8400-e29b-41d4-a716-446655440020', '550e8400-e29b-41d4-a716-446655440002', 1, 'ethereum', 'ETH', '0x0000000000000000000000000000000000000000', '1000000000000000000', datetime('now', '-2 days'), datetime('now')),
    ('650e8400-e29b-41d4-a716-446655440021', '550e8400-e29b-41d4-a716-446655440002', 1, 'usd-coin', 'USDC', '0xA0b86991c6218b36c1d5194A4C4C4C4C4C4C4C4C4', '1000000000', datetime('now', '-1 day'), datetime('now'))
ON CONFLICT(id) DO NOTHING;

