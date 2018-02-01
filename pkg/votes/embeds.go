package votes

import (
	"fmt"
	"time"

	"github.com/playnet-public/democracy.bot/pkg/helpers"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

func newVoteEmbed(title, description string, author *discordgo.User) *discordgo.MessageEmbed {

	embed := helpers.NewEmbed().
		SetTitle(fmt.Sprintf("[Vote] %s", title)).
		SetAuthor(author.Username, author.AvatarURL("100x100")).
		SetColor(0x587987).
		SetDescription(description).
		SetTimestamp(time.Now()).
		AddField(
			"Due Date",
			fmt.Sprintf("%s", time.Now().AddDate(0, 0, 3).UTC().Format("02-01-2006 - 15:04:05")),
			false,
		).
		AddField("Pro", ":white_check_mark:", true).
		AddField("Con", ":negative_squared_cross_mark:", true)
	return embed.MessageEmbed
}

func newVoteSuccessEmbed(s *discordgo.Session, c string, desc string, author *discordgo.User) error {
	feedbackEmbed, err := s.ChannelMessageSendEmbed(c, &discordgo.MessageEmbed{
		Title:       "Vote created",
		Description: desc,
		Author: &discordgo.MessageEmbedAuthor{
			Name:    author.ID,
			IconURL: author.AvatarURL("100x100"),
		},
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "User",
				Value:  author.Username,
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Undo",
				Value:  "Press :leftwards_arrow_with_hook:",
				Inline: true,
			},
		},
	})
	err = s.MessageReactionAdd(c, feedbackEmbed.ID, "↩")
	if err != nil {
		err = errors.Wrap(err, "unable to add emoji")
		s.ChannelMessageDelete(c, feedbackEmbed.ID)
		return err
	}
	return nil
}

func newVoteFailedEmbed(s *discordgo.Session, c string, err string, author *discordgo.User) error {
	_, e := s.ChannelMessageSendEmbed(c, &discordgo.MessageEmbed{
		Title: "Vote failed",
		Author: &discordgo.MessageEmbedAuthor{
			Name:    author.ID,
			IconURL: author.AvatarURL("100x100"),
		},
		Description: fmt.Sprintf("Error: %s", err),
	})
	return e
}
