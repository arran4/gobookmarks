CREATE TABLE `bookmarks` (
  `idbookmarks` int(10) NOT NULL AUTO_INCREMENT,
  `userReference` VARCHAR(1024) NOT NULL,
  `list` mediumblob DEFAULT NULL,
  PRIMARY KEY (`idbookmarks`),
  KEY `userReference_FKIndex1` (`userReference`)
);
