-- Part 3/5 of enforcing uniqueness on Type.name. See
-- 000022_type_name_unique_repoint_artifact.up.sql for the full rationale.
UPDATE `Execution` e
JOIN (
    SELECT t.id AS dup_id, (SELECT MIN(t2.id) FROM `Type` t2 WHERE t2.name = t.name) AS keep_id
    FROM `Type` t
) d ON e.type_id = d.dup_id AND d.dup_id <> d.keep_id
LEFT JOIN `Execution` survivor ON survivor.type_id = d.keep_id AND survivor.name = e.name
SET e.type_id = d.keep_id
WHERE survivor.id IS NULL;
