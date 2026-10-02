-- Enforce uniqueness on Type.name.
--
-- Type.name is already the de facto unique key used by the application:
-- TypeRepository.Save looks up a type by name alone (never version), and the
-- 000016 seed data dedups the same way. Without a database constraint,
-- concurrent initializers (for example an old-version pod racing a new one
-- during a rolling upgrade, or any MySQL deployment where no advisory lock
-- serializes initialization) can each miss the lookup and insert their own
-- row, producing duplicate Type rows with different ids for the same name.
--
-- Existing duplicates are collapsed first (lowest id per name survives) so
-- the constraint can be added:
--
--   * Artifact/Context/Execution rows are real user data; their type_id is
--     repointed to the surviving row unless doing so would collide with
--     that table's own UNIQUE (type_id, name) constraint, in which case the
--     row is left pointing at the row being removed. Such a collision means
--     a same-named instance already exists under the surviving type, which
--     indicates a pre-existing data issue unrelated to this migration.
--   * TypeProperty rows for a duplicate are simply dropped rather than
--     repointed: syncTypes always (re)creates the full property set for
--     whatever type id it resolves in a given run, so the surviving row
--     already has its own complete, independently-created property set.
--   * ParentType is schema-only dead weight — no application code reads or
--     writes it — so duplicate rows there are left as-is.
--
-- If any Artifact/Context/Execution collision above prevented a duplicate
-- from being fully repointed, the ADD CONSTRAINT below fails loudly rather
-- than silently dropping or merging that data, surfacing the conflict for
-- manual resolution.
CREATE TEMP TABLE _type_survivors AS
SELECT name, MIN(id) AS keep_id
FROM "Type"
GROUP BY name;

CREATE TEMP TABLE _type_dupes AS
SELECT t.id AS dup_id, s.keep_id
FROM "Type" t
JOIN _type_survivors s ON t.name = s.name AND t.id <> s.keep_id;

UPDATE "Artifact" a
SET type_id = d.keep_id
FROM _type_dupes d
WHERE a.type_id = d.dup_id
  AND NOT EXISTS (
    SELECT 1 FROM "Artifact" survivor
    WHERE survivor.type_id = d.keep_id AND survivor.name = a.name
  );

UPDATE "Context" c
SET type_id = d.keep_id
FROM _type_dupes d
WHERE c.type_id = d.dup_id
  AND NOT EXISTS (
    SELECT 1 FROM "Context" survivor
    WHERE survivor.type_id = d.keep_id AND survivor.name = c.name
  );

UPDATE "Execution" e
SET type_id = d.keep_id
FROM _type_dupes d
WHERE e.type_id = d.dup_id
  AND NOT EXISTS (
    SELECT 1 FROM "Execution" survivor
    WHERE survivor.type_id = d.keep_id AND survivor.name = e.name
  );

DELETE FROM "TypeProperty" tp
USING _type_dupes d
WHERE tp.type_id = d.dup_id;

DELETE FROM "Type" t
USING _type_dupes d
WHERE t.id = d.dup_id
  AND NOT EXISTS (SELECT 1 FROM "Artifact" a WHERE a.type_id = t.id)
  AND NOT EXISTS (SELECT 1 FROM "Context" c WHERE c.type_id = t.id)
  AND NOT EXISTS (SELECT 1 FROM "Execution" e WHERE e.type_id = t.id);

DROP TABLE _type_survivors;
DROP TABLE _type_dupes;

ALTER TABLE "Type" ADD CONSTRAINT uq_type_name UNIQUE (name);
