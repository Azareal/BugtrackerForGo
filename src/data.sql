CREATE DATABASE bugz;

CREATE TABLE `users`(
	`uid` int not null AUTO_INCREMENT,
	`name` varchar(100) not null,
	`password` varchar(100) not null,
	`salt` varchar(80) DEFAULT '',
	`group` int not null,
	`is_admin` tinyint(1) not null,
	`createdAt` datetime not null,
	`lastActiveAt` datetime not null,
	`session` varchar(200) DEFAULT '',
	primary key(`uid`)
);

CREATE TABLE `users_groups`(
	`gid` int not null AUTO_INCREMENT,
	`name` varchar(100) not null,
	primary key(`gid`)
);

CREATE TABLE `issues`(
	`iid` int not null AUTO_INCREMENT,
	`title` varchar(100) not null,
	`content` text not null,
	`createdAt` datetime not null,
	`lastReplyAt` datetime not null,
	`createdBy` int not null,
	`status` varchar(100) DEFAULT 'open' not null,
	`is_closed` tinyint DEFAULT 0 not null,
	`tags` varchar(200) DEFAULT '' not null,
	primary key(`iid`)
);

CREATE TABLE `issues_replies`(
	`irid` int not null AUTO_INCREMENT,
	`iid` int not null,
	`content` text not null,
	`createdAt` datetime not null,
	`createdBy` int not null,
	primary key(`irid`)
);

INSERT INTO users(`name`,`group`,`is_admin`,`createdAt`,`lastActiveAt`) 
VALUES ('Admin',1,1,NOW(),NOW());
INSERT INTO users_groups(`name`) VALUES ('Administrator');
INSERT INTO issues(`title`,`content`,`createdAt`,`lastReplyAt`,`createdBy`) 
VALUES ('Issue 1','Sample Issue 1',NOW(),NOW(),1);
INSERT INTO issues(`title`,`content`,`createdAt`,`lastReplyAt`,`createdBy`) 
VALUES ('Issue 2','Sample Issue 2',NOW(),NOW(),1);
INSERT INTO issues(`title`,`content`,`createdAt`,`lastReplyAt`,`createdBy`) 
VALUES ('Issue 3','Sample Issue 3',NOW(),NOW(),1);
INSERT INTO issues(`title`,`content`,`createdAt`,`lastReplyAt`,`createdBy`) 
VALUES ('Issue 4','Sample Issue 4',NOW(),NOW(),1);
INSERT INTO issues(`title`,`content`,`createdAt`,`lastReplyAt`,`createdBy`) 
VALUES ('Issue 5','Sample Issue 5',NOW(),NOW(),1);

INSERT INTO issues_replies(`iid`,`content`,`createdAt`,`createdBy`) 
VALUES (1,'Reply 1',NOW(),1);