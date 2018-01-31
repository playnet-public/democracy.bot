package votes

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/playnet-public/democracy.bot/pkg/helpers"
)

// Vote stores a primitive Vote object
type Vote struct {
	Guild       string
	ID          string
	CurrentID   string
	Title       string
	Description string
	Author      string
	Created     time.Time
	Expires     time.Time
	Pro         int
	Con         int
}

// Embed from Vote
func (v *Vote) Embed(s *discordgo.Session) *discordgo.MessageEmbed {
	author, err := s.User(v.Author)
	if err != nil {
		return nil
	}
	embed := helpers.NewEmbed().
		SetTitle(fmt.Sprintf("[Vote] %s", v.Title)).
		SetAuthor(author.Username, author.AvatarURL("100x100")).
		SetColor(0x587987).
		SetDescription(v.Description).
		SetTimestamp(v.Created).
		AddField(
			"Due Date",
			fmt.Sprintf("%s", v.Expires.UTC().Format("02-01-2006 - 15:04:05")),
			false,
		).
		AddField("Pro", fmt.Sprintf(":white_check_mark: - %d", v.Pro), true).
		AddField("Con", fmt.Sprintf(":negative_squared_cross_mark: - %d", v.Con), true)
	return embed.MessageEmbed
}
