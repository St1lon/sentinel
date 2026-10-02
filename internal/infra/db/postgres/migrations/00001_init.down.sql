DROP INDEX IF EXISTS incidents_monitor_started_idx;
DROP INDEX IF EXISTS incidents_single_open_per_monitor;
DROP TABLE IF EXISTS incidents;

DROP INDEX IF EXISTS checks_monitor_time_idx;
DROP TABLE IF EXISTS checks;

DROP INDEX IF EXISTS monitors_user_created_idx;
DROP INDEX IF EXISTS monitors_due_idx;
DROP TABLE IF EXISTS monitors;

DROP TYPE IF EXISTS monitor_status;
DROP TYPE IF EXISTS monitor_kind;

DROP TABLE IF EXISTS users;
