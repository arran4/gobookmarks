CREATE TABLE IF NOT EXISTS bookmarks (
    user TEXT PRIMARY KEY,
    list BLOB
);
CREATE TABLE IF NOT EXISTS passwords (
    user TEXT PRIMARY KEY,
    hash BLOB
);
CREATE TABLE IF NOT EXISTS history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user TEXT,
    sha TEXT,
    message TEXT,
    text BLOB,
    date TIMESTAMP
);
CREATE TABLE IF NOT EXISTS branches (
    user TEXT,
    name TEXT,
    sha TEXT,
    PRIMARY KEY(user, name)
);
CREATE TABLE IF NOT EXISTS tags (
    user TEXT,
    name TEXT,
    sha TEXT,
    PRIMARY KEY(user, name)
);
CREATE TABLE IF NOT EXISTS meta (
    version INTEGER
);
