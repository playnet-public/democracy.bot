package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	flag "github.com/bborbe/flagenv"

	"github.com/bwmarrin/discordgo"
	raven "github.com/getsentry/raven-go"
	"github.com/golang/glog"
	"github.com/kolide/kit/version"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	app    = "democracy.bot"
	appKey = "democracy.bot"
)

var (
	maxprocsPtr = flag.Int("maxprocs", runtime.NumCPU(), "max go procs")
	sentryDsn   = flag.String("sentrydsn", "", "sentry dsn key")
	dbgPtr      = flag.Bool("debug", false, "debug printing")
	versionPtr  = flag.Bool("version", true, "show or hide version info")

	apiToken     = flag.String("apiToken", "", "discord api token")
	port         = flag.String("port", "80", "auth server port")
	callback     = flag.String("callback", "http://localhost/callback", "oauth callback url")
	clientID     = flag.String("clientID", "", "oauth client id")
	clientSecret = flag.String("clientSecret", "", "oauth client secret")

	sentry *raven.Client
)

type bot struct {
	log *zap.Logger

	demoChan *discordgo.Channel
}

func main() {
	flag.Parse()

	if *versionPtr {
		fmt.Printf("-- PlayNet %s --\n", app)
		version.PrintFull()
	}
	runtime.GOMAXPROCS(*maxprocsPtr)

	// prepare glog
	defer glog.Flush()
	glog.CopyStandardLogTo("info")

	var zapFields []zapcore.Field
	// hide app and version information when debugging
	if !*dbgPtr {
		zapFields = []zapcore.Field{
			zap.String("app", appKey),
			zap.String("version", version.Version().Version),
		}
	}

	// prepare zap logging
	log := newLogger(*dbgPtr).With(zapFields...)
	defer log.Sync()
	log.Info("preparing")

	var err error

	// prepare sentry error logging
	sentry, err = raven.New(*sentryDsn)
	if err != nil {
		panic(err)
	}
	err = raven.SetDSN(*sentryDsn)
	if err != nil {
		panic(err)
	}
	errs := make(chan error)

	// catch system interrupts
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errs <- fmt.Errorf("%s", <-c)
	}()

	// catch errors and throw fatal //TODO: is this good?
	go func() {
		ret := <-errs
		if ret != nil {
			log.Fatal(ret.Error())
		}
	}()

	// run main code
	log.Info("starting")
	raven.CapturePanicAndWait(func() {
		if err := do(log); err != nil {
			log.Fatal("fatal error encountered", zap.Error(err))
			raven.CaptureErrorAndWait(err, map[string]string{"isFinal": "true"})
			errs <- err
		}
	}, nil)
	log.Info("finished")
}

func do(log *zap.Logger) error {
	if *apiToken == "" {
		log.Error("no api token provided")
		return errors.New("no api token provided")
	}
	log.Info("creating discord session")
	discord, err := discordgo.New("Bot " + *apiToken)
	if err != nil {
		log.Error("failed to create discord session", zap.Error(err))
		return err
	}

	bot := &bot{log: log}

	log.Info("adding handlers")
	discord.AddHandler(bot.ready)
	discord.AddHandler(bot.messageCreate)
	discord.AddHandler(bot.reactionAdd)

	err = discord.Open()
	if err != nil {
		log.Error("failed to open discord session", zap.Error(err))
	}

	log.Info("running")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	discord.Close()

	return nil
}

func (b *bot) ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "democracy")
	s.State.TrackChannels = true
	s.State.MaxMessageCount = 100
}

func (b *bot) init(s *discordgo.Session, m *discordgo.MessageCreate, c *discordgo.Channel) bool {
	b.log.Info("running init")

	guild, err := s.Guild(c.GuildID)
	var oldChan *discordgo.Channel
	for _, c := range guild.Channels {
		if c.Name == "democracy" {
			oldChan = c
			_, err := s.ChannelDelete(c.ID)
			if err != nil {
				b.log.Error("could not delete channel", zap.String("cid", c.ID), zap.Error(err))
				return false
			}
		}
	}
	b.demoChan, err = s.GuildChannelCreate(guild.ID, "democracy", "text")
	if err != nil {
		b.log.Error("could not create channel")
		return false
	}
	_, err = s.ChannelEditComplex(b.demoChan.ID, &discordgo.ChannelEdit{
		Name:                 "democracy",
		Topic:                "For the people, by the people",
		Position:             oldChan.Position,
		ParentID:             oldChan.ParentID,
		PermissionOverwrites: oldChan.PermissionOverwrites,
	})
	if err != nil {
		b.log.Error("could not update channel")
		return false
	}

	embed := &discordgo.MessageEmbed{
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
				Value:  "!democracy init",
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Start Vote",
				Value:  "!democracy vote [title]|[text]",
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

	_, err = s.ChannelMessageSendEmbed(b.demoChan.ID, embed)
	if err != nil {
		b.log.Error("unable to send embed", zap.Error(err))
		return false
	}
	return true
}

func (b *bot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	s.State.TrackChannels = true
	s.State.MaxMessageCount = 100

	if strings.HasPrefix(m.Content, "!democracy") {
		m.Content = strings.TrimPrefix(m.Content, "!democracy ")
		b.log.Info("handling message", zap.String("channel", m.ChannelID), zap.String("msg", m.Content))

		c, err := s.State.Channel(m.ChannelID)
		if err != nil {
			b.log.Error("could not find channel")
			return
		}

		if b.demoChan == nil {
			guild, err := s.Guild(c.GuildID)
			if err != nil {
				b.log.Error("could not fetch democracy channel", zap.String("guild", c.GuildID), zap.Error(err))
				return
			}
			for _, c := range guild.Channels {
				if c.Name == "democracy" {
					b.demoChan = c
				}
			}
		}

		if m.Content == "init" {
			b.init(s, m, c)
			s.ChannelMessageDelete(c.ID, m.ID)
		}

		if strings.HasPrefix(m.Content, "vote") {
			err = b.vote(s, m, c)
			s.ChannelMessageDelete(c.ID, m.ID)
			if err != nil {
				b.log.Error("unable to send embed", zap.Error(err))
				s.ChannelMessageSendEmbed(c.ID, &discordgo.MessageEmbed{
					Title:       "Vote failed",
					Description: fmt.Sprintf("Error: %s", err.Error()),
				})
				return
			}
		}
	}
}

func (b *bot) reactionAdd(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	if m.UserID == s.State.User.ID {
		return
	}

	b.log.Info("reaction event",
		zap.String("user", m.UserID),
		zap.String("message", m.MessageID),
		zap.String("channel", m.ChannelID),
		zap.String("emoji", m.Emoji.Name),
	)

	if m.Emoji.Name == "↩" {
		msg, err := s.ChannelMessage(m.ChannelID, m.MessageID)
		if err != nil {
			b.log.Error("unable to find message", zap.Error(err))
			return
		}
		if len(msg.Embeds) < 1 {
			b.log.Error("unable to find message embed", zap.Error(err))
			return
		}
		embed := msg.Embeds[0]
		if embed.Author.Name != m.UserID {
			b.log.Info("permission denied for undo", zap.String("expected", embed.Author.Name), zap.String("user", m.UserID))
			return
		}
		if embed.Title != "Vote created" {
			b.log.Info("not a vote create event")
			return
		}
		voteID := embed.Description
		err = s.ChannelMessageDelete(b.demoChan.ID, voteID)
		if err != nil {
			b.log.Error("unable to delete vote", zap.Error(err))
			return
		}
		embed.Title = "Vote deleted"
		embed.Description = voteID
		embed.Fields = []*discordgo.MessageEmbedField{}
		undoMsg := discordgo.NewMessageEdit(m.ChannelID, m.MessageID)
		undoMsg.Embed = embed
		//s.ChannelMessageDelete(m.ChannelID, m.MessageID)
		s.ChannelMessageEditComplex(undoMsg)
		s.MessageReactionsRemoveAll(m.ChannelID, m.MessageID)
	}
}

func (b *bot) vote(s *discordgo.Session, m *discordgo.MessageCreate, c *discordgo.Channel) error {
	m.Content = strings.TrimPrefix(m.Content, "vote ")
	vote := strings.Split(m.Content, "|")
	if len(vote) < 2 {
		b.log.Error("invalid vote", zap.String("msg", m.Content))
		return errors.New("Invalid vote text. Please follow this schema: '!democracy vote [title]|[text]'")
	}
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("[Vote] %s", vote[0]),
		Author: &discordgo.MessageEmbedAuthor{
			Name:    m.Author.Username,
			IconURL: m.Author.AvatarURL("100x100"),
		},
		Color:       0x587987,
		Description: vote[1],
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "Due Date",
				Value:  fmt.Sprintf("%s", time.Now().AddDate(0, 0, 3).UTC().Format("02-01-2006 - 15:04:05")),
				Inline: false,
			},
			&discordgo.MessageEmbedField{
				Name:   "Pro",
				Value:  ":white_check_mark:",
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Con",
				Value:  ":negative_squared_cross_mark:",
				Inline: true,
			},
		},
	}

	voteEmbed, err := s.ChannelMessageSendEmbed(b.demoChan.ID, embed)
	if err != nil {
		b.log.Error("unable to send embed", zap.Error(err))
		return err
	}
	err = s.MessageReactionAdd(b.demoChan.ID, voteEmbed.ID, "✅")
	if err != nil {
		b.log.Error("unable to add emoji", zap.Error(err))
		s.ChannelMessageDelete(b.demoChan.ID, voteEmbed.ID)
		return err
	}
	err = s.MessageReactionAdd(b.demoChan.ID, voteEmbed.ID, "❎")
	if err != nil {
		b.log.Error("unable to add emoji", zap.Error(err))
		s.ChannelMessageDelete(b.demoChan.ID, voteEmbed.ID)
		return err
	}
	feedbackEmbed, err := s.ChannelMessageSendEmbed(c.ID, &discordgo.MessageEmbed{
		Title:       "Vote created",
		Description: voteEmbed.ID,
		Author: &discordgo.MessageEmbedAuthor{
			Name:    m.Author.ID,
			IconURL: m.Author.AvatarURL("100x100"),
		},
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name:   "User",
				Value:  m.Author.Username,
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "Undo",
				Value:  "Press :leftwards_arrow_with_hook:",
				Inline: true,
			},
		},
	})
	err = s.MessageReactionAdd(c.ID, feedbackEmbed.ID, "↩")
	if err != nil {
		b.log.Error("unable to add emoji", zap.Error(err))
		s.ChannelMessageDelete(c.ID, feedbackEmbed.ID)
		return err
	}
	return nil
}

//TODO: Move this to playnet common libs
func newLogger(dbg bool) *zap.Logger {
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})

	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)
	consoleConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoder := zapcore.NewConsoleEncoder(consoleConfig)
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
	)
	logger := zap.New(core)
	if dbg {
		logger = logger.WithOptions(
			zap.AddCaller(),
			zap.AddStacktrace(zap.ErrorLevel),
		)
	} else {
		logger = logger.WithOptions(
			zap.AddStacktrace(zap.FatalLevel),
		)
	}
	return logger
}
