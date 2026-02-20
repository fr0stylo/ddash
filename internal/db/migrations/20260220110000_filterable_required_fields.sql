-- +goose Up
ALTER TABLE organization_required_fields
ADD COLUMN is_filterable INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE organization_required_fields
DROP COLUMN is_filterable;
