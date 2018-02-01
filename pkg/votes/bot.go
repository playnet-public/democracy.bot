package votes

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type msgFunc func(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate)
type reactFunc func(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageReactionAdd)

// Bot for creating and managing votes
type Bot struct {
	Log *zap.Logger
	// cmd maps function
	messageHandlers map[string]msgFunc
	// message title/content maps function
	reactionHandlers map[string]reactFunc
}

// New bot with logger
func New(log *zap.Logger) *Bot {
	return &Bot{
		Log:              log,
		messageHandlers:  make(map[string]msgFunc),
		reactionHandlers: make(map[string]reactFunc),
	}
}

// AddMessageHandler to Bot
func (b *Bot) AddMessageHandler(cmd string, f msgFunc) {
	b.messageHandlers[cmd] = f
}

// AddReactionHandler to Bot
func (b *Bot) AddReactionHandler(title string, f reactFunc) {
	b.reactionHandlers[title] = f
}

// Ready Event Handler
func (b *Bot) Ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "democracy")
	s.State.TrackChannels = true
	s.State.MaxMessageCount = 100

	/*for _, g := range event.Guilds {
		b.ResetDemocracy(s, g)
	}*/

}

// MessageCreate Event Handler
func (b *Bot) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if m.Content == "!democracy" {
		s.ChannelMessageDelete(m.ChannelID, m.ID)
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, newInitEmbed())
		if err != nil {
			b.Log.Error("unable to send embed", zap.Error(err))
			return
		}
		return
	}
	if !strings.HasPrefix(m.Content, "!democracy ") {
		return
	}
	m.Content = strings.TrimPrefix(m.Content, "!democracy ")
	ch := b.getChannel(s, m.ChannelID)
	b.Log.Info("message event",
		zap.String("user", m.Author.ID),
		zap.String("message", m.Content),
		zap.String("channel", m.ChannelID),
	)
	for k, v := range b.messageHandlers {
		if strings.HasPrefix(m.Content, k) {
			v(ch, s, m)
		}
	}
}

// ReactionAdd Event Handler
func (b *Bot) ReactionAdd(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	if m.UserID == s.State.User.ID {
		return
	}
	ch := b.getChannel(s, m.ChannelID)
	b.Log.Info("reaction event",
		zap.String("user", m.UserID),
		zap.String("message", m.MessageID),
		zap.String("channel", m.ChannelID),
		zap.String("emoji", m.Emoji.Name),
	)
	var title string
	msg, err := s.ChannelMessage(m.ChannelID, m.MessageID)
	if err != nil {
		b.Log.Error("unable to find message", zap.Error(err))
		return
	}
	if len(msg.Embeds) > 0 {
		title = msg.Embeds[0].Title
	} else {
		title = msg.Content
	}

	for k, v := range b.reactionHandlers {
		if strings.HasPrefix(title, k) {
			v(ch, s, m)
		}
	}
}

// ResetDemocracy for the provied guild
func (b *Bot) ResetDemocracy(c *discordgo.Channel, s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Content == "reset_handler" {
		return
	}
	g, err := s.Guild(c.GuildID)
	if err != nil {
		b.Log.Error("could not fetch guild", zap.String("guild", c.GuildID), zap.Error(err))
		return
	}
	errHandler := func(guild *discordgo.Guild, e string) {
		owner, err := s.UserChannelCreate(guild.OwnerID)
		if err != nil {
			b.Log.Error("could not contact owner", zap.String("guild", guild.ID), zap.String("owner", owner.ID))
			return
		}
		_, err = s.ChannelMessageSend(owner.ID, fmt.Sprintf("democracy.bot failed to initialize on your server %s. Please contact support. Error: %s", guild.Name, e))
		if err != nil {
			b.Log.Error("could not contact owner", zap.String("guild", guild.ID), zap.String("owner", owner.ID))
			return
		}
	}

	b.Log.Info("readying guild", zap.String("guild", g.ID), zap.Int("chanCount", len(g.Channels)))
	if len(g.Channels) < 1 {
		return
	}
	var oldChan *discordgo.Channel
	for _, c := range g.Channels {
		b.Log.Debug("looking for default channel", zap.String("guild", g.ID), zap.String("channelID", c.ID), zap.String("channel", c.Name))
		if c.Name == "democracy" {
			b.Log.Debug("caching default channel", zap.String("guild", g.ID), zap.String("channelID", c.ID), zap.String("channel", c.Name))
			oldChan = c
			b.Log.Debug("resetting default channel", zap.String("guild", g.ID), zap.String("channelID", c.ID), zap.String("channel", c.Name))
			_, err := s.ChannelDelete(c.ID)
			if err != nil {
				b.Log.Error("could not delete channel", zap.String("guild", g.ID), zap.String("cid", c.ID), zap.Error(err))
				errHandler(g, "could not delete channel")
				return
			}
		}
	}
	if oldChan == nil {
		b.Log.Error("could not find default channel", zap.String("guild", g.ID))
		errHandler(g, "could not find default channel")
		return
	}
	ch, err := s.GuildChannelCreate(g.ID, "democracy", "text")
	if err != nil {
		b.Log.Error("could not create channel", zap.String("guild", g.ID))
		errHandler(g, "could not create channel")
		return
	}
	_, err = s.ChannelEditComplex(ch.ID, &discordgo.ChannelEdit{
		Name:                 "democracy",
		Topic:                "For the people, by the people",
		Position:             oldChan.Position,
		ParentID:             oldChan.ParentID,
		PermissionOverwrites: oldChan.PermissionOverwrites,
	})
	if err != nil {
		b.Log.Error("could not update channel", zap.String("guild", g.ID))
		errHandler(g, "could not update channel")
		return
	}

	_, err = s.ChannelMessageSendEmbed(ch.ID, newInitEmbed())
	if err != nil {
		b.Log.Error("unable to send embed", zap.Error(err))
		errHandler(g, "unable to send embed")
		return
	}

	// Reload Handlers
	for _, f := range b.messageHandlers {
		m.Content = "reset_handler"
		f(ch, s, m)
	}
}

func (b *Bot) getChannel(s *discordgo.Session, current string) (ch *discordgo.Channel) {

	currentChannel, err := s.State.Channel(current)
	if err != nil {
		b.Log.Error("could not find channel")
		return nil
	}
	guild, err := s.Guild(currentChannel.GuildID)
	if err != nil {
		b.Log.Error("could not fetch democracy channel", zap.String("guild", currentChannel.GuildID), zap.Error(err))
		return
	}

	for _, c := range guild.Channels {
		if c.Name == "democracy" {
			ch = c
		}
	}
	if ch == nil {
		b.Log.Error("democracy channel not found", zap.String("guild", guild.ID))
	}
	return ch
}

/*func (b *Bot) getGuildChannel(s *discordgo.Session, guild *discordgo.Guild) (ch *discordgo.Channel) {
	for _, c := range guild.Channels {
		if c.Name == "democracy" {
			ch = c
		}
	}
	if ch == nil {
		b.Log.Error("democracy channel not found", zap.String("guild", guild.ID))
	}
	return ch
}*/

func newInitEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "Info Board",
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       0x587987,
		Description: "This discord server is ruled by the people.",
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "How this works",
				Value:  "Every 3 weeks there is the chance to name new admin candidates. One week later the vote takes place for 3 days.",
				Inline: false,
			},
			&discordgo.MessageEmbedField{
				Name:   "Who is allowed to participate",
				Value:  "Everybody.",
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Commands",
				Value:  "Here are all commands yet implemented.",
				Inline: false,
			},
			&discordgo.MessageEmbedField{
				Name:   "Reset Democracy Channel",
				Value:  "!democracy reset",
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Start Vote",
				Value:  "!democracy vote [title]|[text]",
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Show this Text",
				Value:  "!democracy",
				Inline: true,
			},
			//&discordgo.MessageEmbedField{
			//	Name:   "Start Distrust Vote",
			//	Value:  "!democracy distrust [reason]",
			//	Inline: true,
			//},
			//&discordgo.MessageEmbedField{
			//	Name:   "Suggest new Admin",
			//	Value:  "To do so, text the bot in private with '!democracy admin [user]'.",
			//	Inline: false,
			//},
		},
	}
}
