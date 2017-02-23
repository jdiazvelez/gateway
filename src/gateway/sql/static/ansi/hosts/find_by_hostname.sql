SELECT
  hosts.api_id as api_id,
  hosts.id as id,
  hosts.name as name,
  hosts.hostname as hostname,
  hosts.cert as cert,
  hosts.private_key as private_key,
  hosts.force_ssl as force_ssl
FROM hosts
WHERE hosts.hostname = ?
