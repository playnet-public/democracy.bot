CREATE TABLE IF NOT EXISTS votes (
    vote_id         VARCHAR(50) PRIMARY KEY,
    guild_id        VARCHAR(50),
    current_id      VARCHAR(50),
    title           VARCHAR(50),
    description     VARCHAR(50),
    author          VARCHAR(30),
    created         DATE,
    expiration      DATE,
    pro             INTEGER,
    con             INTEGER
);
CREATE TABLE IF NOT EXISTS vote_entries (
    vote_id         VARCHAR(50) NOT NULL,
    guild_id        VARCHAR(50) NOT NULL,
    author          VARCHAR(50) NOT NULL,
    vote            BOOLEAN,
    primary key (vote_id, guild_id, author)
);
