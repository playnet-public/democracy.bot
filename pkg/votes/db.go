package votes

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"

	// Using PostgreSQL
	_ "github.com/lib/pq"
	"github.com/pkg/errors"

	"go.uber.org/zap"
)

// InitDB for persisting votes
func (v *VoteHandler) InitDB(host, name, user, password string) error {
	v.log.Info("connecting db", zap.String("host", host), zap.String("db", name))
	db, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, password, host, name))
	if err != nil {
		v.log.Error("unable to open db", zap.String("host", host), zap.String("db", name), zap.String("user", user), zap.Error(err))
		return err
	}
	v.db = db
	v.log.Info("database connected", zap.String("host", host), zap.String("db", name))
	return nil
}

// ReadVotes for guild
func (v *VoteHandler) ReadVotes(guild string) ([]Vote, error) {
	v.log.Info("fetching votes", zap.String("guild", guild))
	vote := Vote{
		Guild: guild,
	}

	votes := []Vote{}

	rows, err := v.db.Query("select vote_id, title, description, author, created, expiration from votes where guild_id = $1", guild)
	if err != nil {
		v.log.Error("error querying rows", zap.String("guild", guild), zap.Error(err))
		return votes, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		err := rows.Scan(&vote.ID, &vote.Title, &vote.Description, &vote.Author, &vote.Created, &vote.Expires)
		if err != nil {
			v.log.Error("could not scan row", zap.String("guild", guild), zap.Error(err))
			continue
		}
		votes = append(votes, vote)
		count = count + 1
	}
	v.log.Info("finished reading votes", zap.Int("count", count))
	err = rows.Err()
	if err != nil {
		v.log.Error("error reading rows", zap.String("guild", guild), zap.Error(err))
		return votes, err
	}

	return votes, nil
}

// GetVote by (current) ID
func (v *VoteHandler) GetVote(guild, id string) (Vote, error) {
	v.log.Info("fetching vote", zap.String("guild", guild), zap.String("vote", id))
	vote := Vote{
		Guild: guild,
	}

	votes := []Vote{}

	rows, err := v.db.Query("select vote_id, title, description, author, created, expiration from votes where guild_id = $1 and current_id = $2", guild, id)
	if err != nil {
		v.log.Error("error querying rows", zap.String("guild", guild), zap.String("vote", id), zap.Error(err))
		return vote, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		err := rows.Scan(&vote.ID, &vote.Title, &vote.Description, &vote.Author, &vote.Created, &vote.Expires)
		if err != nil {
			v.log.Error("could not scan row", zap.String("guild", guild), zap.String("vote", id), zap.Error(err))
			continue
		}
		votes = append(votes, vote)
		count = count + 1
	}
	v.log.Info("finished reading votes", zap.Int("count", count))
	if len(votes) != 1 {
		v.log.Info("received invalid vote count", zap.String("guild", guild), zap.String("vote", id), zap.Int("count", len(votes)))
		return vote, errors.New("invalid vote count")
	}
	err = rows.Err()
	if err != nil {
		v.log.Error("error reading rows", zap.String("guild", guild), zap.String("vote", id), zap.Error(err))
		return vote, err
	}

	return votes[0], nil
}

// InsertVote to guild
func (v *VoteHandler) InsertVote(vote Vote) error {
	v.log.Info("inserting vote", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("author", vote.Author))
	query := "INSERT INTO votes(guild_id, vote_id, current_id, title, description, author, created, expiration) VALUES($1,$2,$3,$4,$5,$6,$7,$8)"
	stmt, err := v.db.Prepare(query)
	if err != nil {
		v.log.Error("error preparing insert", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.Error(err), zap.String("query", query))
		return err
	}
	res, err := stmt.Exec(vote.Guild, vote.ID, vote.ID, vote.Title, vote.Description, vote.Author, vote.Created, vote.Expires)
	if err != nil {
		v.log.Error("error executing insert", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.Error(err))
		return err
	}
	rowCnt, err := res.RowsAffected()
	if err != nil {
		v.log.Error("error getting affected rows", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.Error(err))
		return err
	}
	v.log.Info("finished insert", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("author", vote.Author), zap.Int64("affected", rowCnt))

	return nil
}

// UpdateVote to guild
func (v *VoteHandler) UpdateVote(id string, vote Vote) error {
	v.log.Info("updating vote", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("author", vote.Author))
	query := "UPDATE votes SET current_id = $2 WHERE vote_id = $1"
	stmt, err := v.db.Prepare(query)
	if err != nil {
		v.log.Error("error preparing update", zap.String("guild", vote.Guild), zap.String("currentID", vote.CurrentID), zap.String("vote", vote.ID), zap.Error(err), zap.String("query", query))
		return err
	}
	res, err := stmt.Exec(id, vote.CurrentID)
	if err != nil {
		v.log.Error("error executing update", zap.String("guild", vote.Guild), zap.String("currentID", vote.CurrentID), zap.String("vote", vote.ID), zap.Error(err))
		return err
	}
	rowCnt, err := res.RowsAffected()
	if err != nil {
		v.log.Error("error getting affected rows", zap.String("guild", vote.Guild), zap.String("currentID", vote.CurrentID), zap.String("vote", vote.ID), zap.Error(err))
		return err
	}
	v.log.Info("finished update", zap.Int64("affected", rowCnt))

	return nil
}

// GetVoteCount for vote
func (v *VoteHandler) GetVoteCount(vote Vote) (Vote, error) {
	v.log.Info("fetching vote entries", zap.String("guild", vote.Guild), zap.String("vote", vote.ID))
	rows, err := v.db.Query("select author, vote from vote_entries where guild_id = $1 and vote_id = $2", vote.Guild, vote.ID)
	if err != nil {
		v.log.Error("error querying rows", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.Error(err))
		return vote, err
	}
	defer rows.Close()
	count := 0
	entry := struct {
		author string
		vote   bool
	}{}
	for rows.Next() {
		err := rows.Scan(&entry.author, &entry.vote)
		if err != nil {
			v.log.Error("could not scan row", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.Error(err))
			continue
		}
		if entry.vote {
			vote.Pro = vote.Pro + 1
		} else {
			vote.Con = vote.Con + 1
		}
		count = count + 1
	}
	v.log.Info("finished reading votes", zap.Int("count", count))
	err = rows.Err()
	if err != nil {
		v.log.Error("error reading rows", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.Error(err))
		return vote, err
	}

	return vote, nil
}

// AddVoteEntry for user
func (v *VoteHandler) AddVoteEntry(vote Vote, author string, value bool) error {
	v.log.Info("adding vote entry", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("author", author))
	query := "INSERT INTO vote_entries(vote_id, guild_id, author, vote) VALUES($1,$2,$3,$4)"
	stmt, err := v.db.Prepare(query)
	if err != nil {
		v.log.Error("error preparing insert", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.Error(err), zap.String("query", query))
		return err
	}
	res, err := stmt.Exec(vote.ID, vote.Guild, author, value)
	var rowCnt int64
	if err != nil {
		pge, ok := err.(*pq.Error)
		if ok {
			if pge.Code.Name() != "unique_violation" {
				v.log.Error("error executing insert", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("author", author), zap.Bool("value", value), zap.String("dbErrorName", pge.Code.Name()), zap.String("dbErrorClass", pge.Code.Class().Name()), zap.Error(err))
				return err
			}
			err = v.UpdateVoteEntry(vote, author, value)
			if err != nil {
				v.log.Error("error executing insert", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("author", author), zap.Bool("value", value), zap.Error(err))
				return err
			}
		} else {
			v.log.Error("error executing insert", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("author", author), zap.Bool("value", value), zap.Error(err))
			return err
		}
	} else {
		rowCnt, err = res.RowsAffected()
		if err != nil {
			v.log.Error("error getting affected rows", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.Error(err))
			return err
		}
	}
	v.log.Info("finished entry insert", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("author", author), zap.Bool("value", value), zap.Int64("affected", rowCnt))
	return nil
}

// UpdateVoteEntry for user
func (v *VoteHandler) UpdateVoteEntry(vote Vote, author string, value bool) error {
	v.log.Info("updating vote entry", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("author", author))
	query := "UPDATE vote_entries SET vote = $3 WHERE vote_id = $1 AND author = $2"
	stmt, err := v.db.Prepare(query)
	if err != nil {
		v.log.Error("error preparing update", zap.String("guild", vote.Guild), zap.String("currentID", vote.CurrentID), zap.String("vote", vote.ID), zap.Error(err), zap.String("query", query))
		return err
	}
	res, err := stmt.Exec(vote.ID, author, value)
	if err != nil {
		v.log.Error("error executing update", zap.String("guild", vote.Guild), zap.String("currentID", vote.CurrentID), zap.String("vote", vote.ID), zap.Error(err))
		return err
	}
	rowCnt, err := res.RowsAffected()
	if err != nil {
		v.log.Error("error getting affected rows", zap.String("guild", vote.Guild), zap.String("currentID", vote.CurrentID), zap.String("vote", vote.ID), zap.Error(err))
		return err
	}
	v.log.Info("finished update", zap.Int64("affected", rowCnt))

	return nil
}
