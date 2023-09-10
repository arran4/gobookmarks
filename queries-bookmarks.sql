-- name: CreateBookmarks :exec
-- This query adds a new entry to the "bookmarks" table and returns the last inserted ID as "returnthis".
INSERT INTO bookmarks (userReference, list)
VALUES (?, ?);
SELECT LAST_INSERT_ID() AS returnthis;

-- name: UpdateBookmarks :exec
-- This query updates the "list" column in the "bookmarks" table for a specific user based on their "users_idusers".
UPDATE bookmarks
SET list = ?
WHERE userReference = ?;

-- name: GetBookmarksForUser :one
-- This query retrieves the "list" from the "bookmarks" table for a specific user based on their "users_idusers".
SELECT Idbookmarks, list
FROM bookmarks
WHERE userReference = ?;

