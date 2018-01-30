package votes

import (
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
		if embed.Author.Name != m.UserID {
			v.log.Info("permission denied for undo", zap.String("expected", embed.Author.Name), zap.String("user", m.UserID))
			return
		}
		if embed.Title != "Vote created" {
			v.log.Info("not a vote create event")
			return
		}
		voteID := embed.Description
		err = s.ChannelMessageDelete(c.ID, voteID)
		if err != nil {
			v.log.Error("unable to delete vote", zap.Error(err))
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
}
