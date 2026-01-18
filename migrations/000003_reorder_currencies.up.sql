DROP INDEX IF EXISTS idx_currencies_not_deleted;
CREATE INDEX idx_currencies_code_not_deleted ON currencies (code) WHERE is_deleted = false;
