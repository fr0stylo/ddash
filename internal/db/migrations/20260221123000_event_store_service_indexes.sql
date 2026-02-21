-- +goose Up
CREATE INDEX IF NOT EXISTS idx_event_store_org_subjecttype_time
ON event_store(organization_id, subject_type, event_ts_ms DESC, seq DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_event_store_org_subjecttype_time;
