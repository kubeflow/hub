-- Part 5/5 of enforcing uniqueness on Type.name. See
-- 000022_type_name_unique_repoint_artifact.up.sql for the full rationale.
ALTER TABLE `Type` ADD CONSTRAINT `uq_type_name` UNIQUE (`name`);
