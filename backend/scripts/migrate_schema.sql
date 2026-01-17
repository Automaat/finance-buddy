-- Migration to split Account into Asset and Account types
-- Run this before executing migrate_split_accounts.py

-- Create assets table
CREATE TABLE IF NOT EXISTS assets (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

-- Add asset_id column to snapshot_values (nullable)
ALTER TABLE snapshot_values
ADD COLUMN IF NOT EXISTS asset_id INTEGER;

-- Make account_id nullable
ALTER TABLE snapshot_values
ALTER COLUMN account_id DROP NOT NULL;

-- Add foreign key constraint for asset_id
ALTER TABLE snapshot_values
ADD CONSTRAINT fk_snapshot_values_asset_id FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE;

-- Drop old unique constraint
ALTER TABLE snapshot_values
DROP CONSTRAINT IF EXISTS uix_snapshot_account;

-- Add CHECK constraint to ensure exactly one of asset_id or account_id is set
ALTER TABLE snapshot_values
ADD CONSTRAINT ck_asset_or_account CHECK (
    (asset_id IS NOT NULL AND account_id IS NULL) OR
    (asset_id IS NULL AND account_id IS NOT NULL)
);

-- Add new unique constraints
ALTER TABLE snapshot_values
ADD CONSTRAINT uix_snapshot_asset UNIQUE (snapshot_id, asset_id);

ALTER TABLE snapshot_values
ADD CONSTRAINT uix_snapshot_account UNIQUE (snapshot_id, account_id);
