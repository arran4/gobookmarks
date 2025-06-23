-- Migrates goa4web-style users and bookmarks tables to the gobookmarks SQL schema.
-- Creates the required tables and copies legacy data.
CREATE TABLE IF NOT EXISTS bookmarks (
    user VARCHAR(255) PRIMARY KEY,
    list BLOB
);
CREATE TABLE IF NOT EXISTS passwords (
    user VARCHAR(255) PRIMARY KEY,
    hash BLOB
);
CREATE TABLE IF NOT EXISTS history (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    user VARCHAR(255),
    sha CHAR(40),
    message TEXT,
    text BLOB,
    date TIMESTAMP
);
CREATE TABLE IF NOT EXISTS branches (
    user VARCHAR(255),
    name VARCHAR(255),
    sha CHAR(40),
    PRIMARY KEY(user, name)
);
CREATE TABLE IF NOT EXISTS tags (
    user VARCHAR(255),
    name VARCHAR(255),
    sha CHAR(40),
    PRIMARY KEY(user, name)
);
CREATE TABLE IF NOT EXISTS meta (
    version INTEGER
);
INSERT INTO meta(version) VALUES(1);
REPLACE INTO passwords(user, hash)
SELECT username, passwd FROM users;
REPLACE INTO bookmarks(user, list)
SELECT u.username, b.list FROM bookmarks b JOIN users u ON b.users_idusers=u.idusers;
REPLACE INTO history(user, sha, message, text, date)
SELECT u.username, SHA1(CONCAT(NOW(), b.list)), 'import', b.list, NOW()
FROM bookmarks b JOIN users u ON b.users_idusers=u.idusers;
REPLACE INTO branches(user, name, sha)
SELECT u.username, 'main', SHA1(CONCAT(NOW(), b.list))
FROM bookmarks b JOIN users u ON b.users_idusers=u.idusers;
