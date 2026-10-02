-- Part 1/5 of enforcing uniqueness on Type.name (see
-- internal/platform/db/postgres/migrations/000026_type_name_unique.up.sql
-- for the full rationale). Split across several single-statement files
-- because golang-migrate's mysql driver runs one statement per file.
--
-- Repoint Artifact.type_id off a duplicate Type row onto the row that will
-- survive (lowest id per name), unless doing so would collide with that
-- table's own UNIQUE (type_id, name) constraint -- in which case the row is
-- left pointing at the duplicate being removed, for the final ADD UNIQUE KEY
-- migration to fail loudly on instead of silently dropping data.
UPDATE `Artifact` a
JOIN (
    SELECT t.id AS dup_id, (SELECT MIN(t2.id) FROM `Type` t2 WHERE t2.name = t.name) AS keep_id
    FROM `Type` t
) d ON a.type_id = d.dup_id AND d.dup_id <> d.keep_id
LEFT JOIN `Artifact` survivor ON survivor.type_id = d.keep_id AND survivor.name = a.name
SET a.type_id = d.keep_id
WHERE survivor.id IS NULL;
