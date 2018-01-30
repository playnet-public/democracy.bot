package votes

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/playnet-public/democracy.bot/pkg/helpers"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// VoteHandler for primitive votes
type VoteHandler struct {
	log *zap.Logger
}

// NewVoteHandler for channel
func NewVoteHandler(log *zap.Logger) *VoteHandler {
	return &VoteHandler{
		log: log,
	}
}

// Vote Message Handler
func (v *VoteHandler) Vote(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) {
	var r result
	r = newResult("failed creating vote", "unable to create vote")

	// Send callback with the final result
	defer func() { v.MessageCallback(s, m, r) }()

	m.Content = strings.TrimPrefix(m.Content, "vote ")
	vote := strings.Split(m.Content, "|")
	if len(vote) < 2 {
		r = newResult(
			"invalid vote",
			"Invalid vote text. Please follow this schema: '!democracy vote [title]|[text]'",
		)
		return
	}

	embed := newVoteEmbed(vote[0], vote[1], m.Author)
	voteEmbed, err := s.ChannelMessageSendEmbed(c.ID, embed)
	if err != nil {
		r = newResult(
			"unable to send embed",
			"unable to send embed",
			err,
		)
		return
	}
	err = s.MessageReactionAdd(c.ID, voteEmbed.ID, "✅")
	if err != nil {
		s.ChannelMessageDelete(c.ID, voteEmbed.ID)
		r = newResult(
			"unable to add emoji",
			"unable to add emoji ✅",
			err,
		)
		return
	}
	err = s.MessageReactionAdd(c.ID, voteEmbed.ID, "❎")
	if err != nil {
		s.ChannelMessageDelete(c.ID, voteEmbed.ID)
		r = newResult(
			"unable to add emoji",
			"unable to add emoji ❎",
			err,
		)
		return
	}
	r = newResult("", voteEmbed.ID)
}

// MessageCallback for handling errors and success messages
func (v *VoteHandler) MessageCallback(s *discordgo.Session, m *discordgo.MessageCreate, r result) {
	var err error
	if r.err != nil {
		v.log.Error(r.err.Error(), zap.String("msg", m.Content), zap.Error(r.err))
		err = newVoteSuccessEmbed(s, m.ChannelID, r.response, m.Author)
	} else {
		v.log.Info("vote success", zap.String("msg", m.Content))
		err = newVoteSuccessEmbed(s, m.ChannelID, r.response, m.Author)
	}
	if err != nil {
		v.log.Error("failed to create callback embed", zap.Error(err))
		return
	}

	s.ChannelMessageDelete(m.ChannelID, m.ID)
}

type result struct {
	err      error
	response string
	embed    *helpers.Embed
}

func newResult(err, resp string, errs ...error) result {
	r := result{}
	if err != "" {
		if len(errs) > 0 {
			r.err = errors.Wrap(errors.New(err), errs[0].Error())
		} else {
			r.err = errors.New(err)
		}
	}
	r.response = resp
	return r
}
