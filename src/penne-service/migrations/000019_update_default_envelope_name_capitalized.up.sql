-- Capitalize system envelope and envelope group names to 'Default'
UPDATE envelope SET name = 'Default' WHERE is_system = true OR lower(name) = 'default';
UPDATE envelope_group SET name = 'Default' WHERE is_system = true OR lower(name) = 'default';
