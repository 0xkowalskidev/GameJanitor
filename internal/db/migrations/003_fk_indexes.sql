CREATE INDEX IF NOT EXISTS idx_gameservers_game_id ON gameservers(game_id);
CREATE INDEX IF NOT EXISTS idx_schedules_gameserver_id ON schedules(gameserver_id);
CREATE INDEX IF NOT EXISTS idx_backups_gameserver_id ON backups(gameserver_id);
