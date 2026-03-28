-- Add 6 new fields to business_terms table
ALTER TABLE business_terms ADD COLUMN IF NOT EXISTS standard_code VARCHAR(50) UNIQUE;
ALTER TABLE business_terms ADD COLUMN IF NOT EXISTS domain VARCHAR(50);
ALTER TABLE business_terms ADD COLUMN IF NOT EXISTS data_type_standard VARCHAR(50);
ALTER TABLE business_terms ADD COLUMN IF NOT EXISTS validation_rule TEXT;
ALTER TABLE business_terms ADD COLUMN IF NOT EXISTS owner VARCHAR(100);
ALTER TABLE business_terms ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active';
