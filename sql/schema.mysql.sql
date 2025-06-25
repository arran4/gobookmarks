CREATE TABLE IF NOT EXISTS bookmarks (
    user TEXT,
    list BLOB,
    PRIMARY KEY(user(191))
);

CREATE TABLE IF NOT EXISTS passwords (
    user TEXT,
    hash BLOB,
    PRIMARY KEY(user(191))
);

CREATE TABLE IF NOT EXISTS history (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
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
    PRIMARY KEY(user(191), name(191))
);

CREATE TABLE IF NOT EXISTS tags (
    user TEXT,
    name TEXT,
    sha TEXT,
    PRIMARY KEY(user(191), name(191))
);

CREATE TABLE IF NOT EXISTS meta (
    version INTEGER
);
