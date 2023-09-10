CREATE TABLE bookmarks (
  idbookmarks INTEGER PRIMARY KEY AUTOINCREMENT,
  userReference TEXT NOT NULL,
  list BLOB
);
