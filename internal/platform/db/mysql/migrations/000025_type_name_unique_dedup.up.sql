-- Part 4/5 of enforcing uniqueness on Type.name. See
-- 000022_type_name_unique_repoint_artifact.up.sql for the full rationale.
--
-- Remove duplicate Type rows (every row for a name except the lowest id,
-- which the preceding repoint migrations treated as the survivor), along
-- with their TypeProperty rows. TypeProperty rows are dropped rather than
-- repointed: syncTypes always (re)creates the full property set for
-- whatever type id it resolves in a given run, so the surviving row already
-- has its own complete, independently-created property set.
--
-- A duplicate that still has an Artifact/Context/Execution row pointing at
-- it (because the corresponding repoint migration skipped it to avoid a
-- UNIQUE (type_id, name) collision) is left in place; the next migration's
-- ADD UNIQUE KEY then fails loudly instead of silently losing that
-- reference.
DELETE tp, dup
FROM `Type` dup
JOIN `Type` keep ON keep.name = dup.name AND keep.id < dup.id
LEFT JOIN `TypeProperty` tp ON tp.type_id = dup.id
LEFT JOIN `Artifact` a ON a.type_id = dup.id
LEFT JOIN `Context` ctx ON ctx.type_id = dup.id
LEFT JOIN `Execution` e ON e.type_id = dup.id
WHERE a.id IS NULL AND ctx.id IS NULL AND e.id IS NULL;
