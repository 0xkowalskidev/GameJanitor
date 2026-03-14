ALTER TABLE games RENAME COLUMN min_memory_mb TO recommended_memory_mb;
ALTER TABLE games DROP COLUMN min_cpu;
