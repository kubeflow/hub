-- The row dedup performed by the up migration is not reversible.
ALTER TABLE "Type" DROP CONSTRAINT IF EXISTS uq_type_name;
