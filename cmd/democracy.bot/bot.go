package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	flag "github.com/bborbe/flagenv"
	"github.com/playnet-public/democracy.bot/pkg/votes"

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

	dbHost     = flag.String("dbHost", "localhost", "db server host")
	dbUser     = flag.String("dbUser", "db", "db user")
	dbName     = flag.String("dbName", "db", "db name")
	dbPassword = flag.String("dbPassword", "dev", "db password")

	sentry *raven.Client
)

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

	voteHandler := votes.NewVoteHandler(log)
	err = voteHandler.InitDB(*dbHost, *dbName, *dbUser, *dbPassword)
	bot := votes.New(log)

	bot.AddMessageHandler("reset", bot.ResetDemocracy)
	bot.AddMessageHandler("vote", voteHandler.Vote)
	bot.AddReactionHandler("Vote created", voteHandler.React)
	bot.AddReactionHandler("[Vote]", voteHandler.React)

	log.Info("adding handlers")
	discord.AddHandler(bot.Ready)
	discord.AddHandler(bot.MessageCreate)
	discord.AddHandler(bot.ReactionAdd)

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
