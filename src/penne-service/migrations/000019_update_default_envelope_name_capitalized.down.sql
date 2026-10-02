-- Revert system envelope and envelope group names to 'default'
UPDATE envelope SET name = 'default' WHERE is_system = true OR lower(name) = 'default';
UPDATE envelope_group SET name = 'default' WHERE is_system = true OR lower(name) = 'default';
