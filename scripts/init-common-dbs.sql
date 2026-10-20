-- Create dedicated databases for workspace microservices if they do not exist
SELECT 'CREATE DATABASE nouri_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'nouri_db')\gexec

SELECT 'CREATE DATABASE pulse_app'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'pulse_app')\gexec
