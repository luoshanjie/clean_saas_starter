-- PostgreSQL bootstrap script (run with psql, connected to "postgres" DB).
-- This script is idempotent in psql via \gexec.

SELECT 'CREATE DATABASE service_dev'
WHERE NOT EXISTS (
  SELECT 1 FROM pg_database WHERE datname = 'service_dev'
)\gexec

-- SELECT 'CREATE DATABASE service_release'
-- WHERE NOT EXISTS (
--   SELECT 1 FROM pg_database WHERE datname = 'service_release'
-- )\gexec
