package votes

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// React Handler
func (v *VoteHandler) React(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	if m.Emoji.Name == "↩" {
		msg, err := s.ChannelMessage(m.ChannelID, m.MessageID)
		if err != nil {
			v.log.Error("unable to find message", zap.Error(err))
			return
		}
		if len(msg.Embeds) < 1 {
			v.log.Error("unable to find message embed", zap.Error(err))
			return
		}
		embed := msg.Embeds[0]
		isVote := false
		if embed.Title != "Vote created" {
			if !strings.HasPrefix(embed.Title, "[Vote]") {
				v.log.Info("not a vote create event")
				return
			}
			isVote = true
		}
		voteID := embed.Description
		vote, err := v.GetVote(c.GuildID, voteID)
		if err != nil {
			vote, err = v.GetVote(c.GuildID, m.MessageID)
			if err != nil {
				v.log.Error("unable to fetch vote from db", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("user", m.UserID), zap.Error(err))
				return
			}
		}
		if vote.Author != m.UserID {
			v.log.Info("permission denied for undo", zap.String("expected", vote.Author), zap.String("user", m.UserID))
			return
		}
		err = v.DeleteVote(vote)
		if err != nil {
			v.log.Error("unable to delete vote from db", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("author", vote.Author), zap.Error(err))
			return
		}
		err = s.ChannelMessageDelete(c.ID, vote.CurrentID)
		if err != nil {
			err = s.ChannelMessageDelete(c.ID, vote.ID)
			if err != nil {
				v.log.Error("unable to delete vote", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("author", vote.Author), zap.Error(err))
				return
			}
		}
		err = v.DeleteVoteEntries(vote)
		if err != nil {
			v.log.Error("unable to delete vote entries from db", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("author", vote.Author), zap.Error(err))
			return
		}
		if isVote {
			return
		}
		embed.Title = "Vote deleted"
		embed.Description = voteID
		embed.Fields = []*discordgo.MessageEmbedField{}
		undoMsg := discordgo.NewMessageEdit(m.ChannelID, m.MessageID)
		undoMsg.Embed = embed
		//s.ChannelMessageDelete(m.ChannelID, m.MessageID)
		_, err = s.ChannelMessageEditComplex(undoMsg)
		if err != nil {
			v.log.Error("unable to update message", zap.Error(err))
		}
		s.MessageReactionsRemoveAll(m.ChannelID, m.MessageID)
	}
	if m.Emoji.Name == "✅" {
		v.log.Info("updating vote", zap.String("guild", c.GuildID), zap.String("vote", m.MessageID), zap.String("user", m.UserID))
		vote, err := v.GetVote(c.GuildID, m.MessageID)
		if err != nil {
			v.log.Error("unable to fetch vote from db", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("user", m.UserID), zap.Error(err))
			return
		}
		err = v.AddVoteEntry(vote, m.UserID, true)
		if err != nil {
			v.log.Error("unable to write value to db", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("user", m.UserID), zap.Error(err))
			return
		}
		vote, err = v.GetVoteCount(vote)
		if err != nil {
			v.log.Error("unable to get vote entries", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.Error(err))
			return
		}
		edit := discordgo.NewMessageEdit(c.ID, m.MessageID)
		edit.Embed = vote.Embed(s)
		_, err = s.ChannelMessageEditComplex(edit)
		if err != nil {
			v.log.Error("unable to update vote", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("channel", c.ID), zap.String("user", m.UserID), zap.Error(err))
			return
		}
		err = s.MessageReactionRemove(c.ID, m.MessageID, m.Emoji.Name, m.UserID)
		if err != nil {
			v.log.Error("unable to remove reaction", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("channel", c.ID), zap.String("message", m.MessageID), zap.String("emoji", m.Emoji.Name), zap.String("user", m.UserID), zap.Error(err))
			return
		}
		/*err = s.MessageReactionsRemoveAll(editMsg.ChannelID, editMsg.ID)
		if err != nil {
			v.log.Error("unable to remove reaction", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("channel", c.ID), zap.String("user", m.UserID), zap.Error(err))
			return
		}
		err = s.MessageReactionAdd(editMsg.ChannelID, editMsg.ID, "✅")
		if err != nil {
			s.ChannelMessageDelete(editMsg.ChannelID, editMsg.ID)
			v.log.Error("unable to add emoji", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.Error(err))
			return
		}
		err = s.MessageReactionAdd(editMsg.ChannelID, editMsg.ID, "❎")
		if err != nil {
			s.ChannelMessageDelete(editMsg.ChannelID, editMsg.ID)
			v.log.Error("unable to add emoji", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.Error(err))
			return
		}*/
	}
	if m.Emoji.Name == "❎" {
		v.log.Info("updating vote", zap.String("guild", c.GuildID), zap.String("vote", m.MessageID), zap.String("user", m.UserID))
		vote, err := v.GetVote(c.GuildID, m.MessageID)
		if err != nil {
			v.log.Error("unable to fetch vote from db", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("user", m.UserID), zap.Error(err))
			return
		}
		err = v.AddVoteEntry(vote, m.UserID, false)
		if err != nil {
			v.log.Error("unable to write value to db", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("user", m.UserID), zap.Error(err))
			return
		}
		vote, err = v.GetVoteCount(vote)
		if err != nil {
			v.log.Error("unable to get vote entries", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.Error(err))
			return
		}
		edit := discordgo.NewMessageEdit(c.ID, m.MessageID)
		edit.Embed = vote.Embed(s)
		_, err = s.ChannelMessageEditComplex(edit)
		if err != nil {
			v.log.Error("unable to update vote", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("channel", c.ID), zap.String("user", m.UserID), zap.Error(err))
			return
		}
		err = s.MessageReactionRemove(c.ID, m.MessageID, m.Emoji.Name, m.UserID)
		if err != nil {
			v.log.Error("unable to remove reaction", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("channel", c.ID), zap.String("message", m.MessageID), zap.String("emoji", m.Emoji.Name), zap.String("user", m.UserID), zap.Error(err))
			return
		}
		/*err = s.MessageReactionsRemoveAll(editMsg.ChannelID, editMsg.ID)
		if err != nil {
			v.log.Error("unable to remove reaction", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.String("channel", c.ID), zap.String("user", m.UserID), zap.Error(err))
			return
		}
		err = s.MessageReactionAdd(editMsg.ChannelID, editMsg.ID, "✅")
		if err != nil {
			s.ChannelMessageDelete(editMsg.ChannelID, editMsg.ID)
			v.log.Error("unable to add emoji", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.Error(err))
			return
		}
		err = s.MessageReactionAdd(editMsg.ChannelID, editMsg.ID, "❎")
		if err != nil {
			s.ChannelMessageDelete(editMsg.ChannelID, editMsg.ID)
			v.log.Error("unable to add emoji", zap.String("guild", vote.Guild), zap.String("vote", vote.ID), zap.Error(err))
			return
		}*/
	}
}
