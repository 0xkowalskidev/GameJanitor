CREATE TABLE games (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    image TEXT NOT NULL,
    default_ports JSON NOT NULL DEFAULT '[]',
    default_env JSON NOT NULL DEFAULT '[]',
    min_memory_mb INTEGER NOT NULL DEFAULT 0,
    min_cpu REAL NOT NULL DEFAULT 0,
    gsq_game_slug TEXT,
    disabled_capabilities JSON NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE gameservers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    game_id TEXT NOT NULL REFERENCES games(id),
    ports JSON NOT NULL DEFAULT '[]',
    env JSON NOT NULL DEFAULT '{}',
    memory_limit_mb INTEGER NOT NULL DEFAULT 0,
    cpu_limit REAL NOT NULL DEFAULT 0,
    auto_start BOOLEAN NOT NULL DEFAULT 0,
    container_id TEXT,
    volume_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'stopped',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE schedules (
    id TEXT PRIMARY KEY,
    gameserver_id TEXT NOT NULL REFERENCES gameservers(id),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    cron_expr TEXT NOT NULL,
    payload JSON NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT 1,
    last_run DATETIME,
    next_run DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE backups (
    id TEXT PRIMARY KEY,
    gameserver_id TEXT NOT NULL REFERENCES gameservers(id),
    name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    size_bytes INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
