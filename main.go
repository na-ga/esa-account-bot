package main

import (
	"net/http"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/nlopes/slack"
)

type (
	configuration struct {
		Port               string   `envconfig:"PORT" default:"3000"`
		ChannelID          string   `envconfig:"CHANNEL_ID" required:"true"`
		BotID              string   `envconfig:"BOT_ID" required:"true"`
		BotToken           string   `envconfig:"BOT_TOKEN" required:"true"`
		BotUsageURL        string   `envconfig:"BOT_USAGE_URL"`
		VerificationToken  string   `envconfig:"VERIFICATION_TOKEN" required:"true"`
		AllowEmailDomains  []string `envconfig:"ALLOW_EMAIL_DOMAINS"`
		EsaToken           string   `envconfig:"ESA_TOKEN" required:"true"`
		EsaTeamName        string   `envconfig:"ESA_TEAM_NAME" required:"true"`
		AdminIDs           []string `envconfig:"ADMIN_IDS" required:"true"`
		AdminGroupID       string   `envconfig:"ADMIN_GROUP_ID"`
		AccountExpireMonth int      `envconfig:"ACCOUNT_EXPIRE_MONTH" default:"6"`
		Organizations      []string `envconfig:"ORGANIZATIONS"`
	}
)

const (
	envPrefix = ""
)

func main() {

	// parse config
	var conf configuration
	if err := envconfig.Process(envPrefix, &conf); err != nil {
		logger.Errorf("Failed to process env var: %s", err)
		os.Exit(1)
	}
	accountExpireMonth := conf.AccountExpireMonth
	if accountExpireMonth < 1 {
		accountExpireMonth = 1
	}

	// setup
	logger.Infof("Start slack event listening")
	slackClient := slack.New(conf.BotToken)
	bot, err := slackClient.GetUserInfo(conf.BotID)
	if err != nil {
		logger.Errorf("Failed to get bot profile: %s", err)
		os.Exit(1)
	}
	esaClient, err := NewEsaClient(conf.EsaTeamName, conf.EsaToken)
	if err != nil {
		logger.Errorf("Failed to create esa client: %s", err)
		os.Exit(1)
	}
	repository, err := NewRepository(slackClient, conf.AdminIDs, conf.AllowEmailDomains, conf.Organizations)
	if err != nil {
		logger.Errorf("Failed to create repository: %s", err)
		os.Exit(1)
	}

	// listening slack event and response
	listener := &MessageListener{
		esaClient:          esaClient,
		slackClient:        slackClient,
		repository:         repository,
		channelID:          conf.ChannelID,
		botID:              conf.BotID,
		botName:            bot.Name,
		botUsageURL:        conf.BotUsageURL,
		accountExpireMonth: accountExpireMonth,
	}
	go listener.Run()

	// register handler to receive interactive message responses from slack (kicked by user action)
	auxMux := http.NewServeMux()
	auxMux.Handle("/interaction", InteractionHandler{
		esaClient:         esaClient,
		slackClient:       slackClient,
		repository:        repository,
		channelID:         conf.ChannelID,
		verificationToken: conf.VerificationToken,
		adminIDs:          conf.AdminIDs,
		adminGroupID:      conf.AdminGroupID,
	})
	auxMux.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	logger.Infof("Server listening on :%s", conf.Port)
	if err := http.ListenAndServe(":"+conf.Port, auxMux); err != nil {
		logger.Errorf("Failed to server listening: %s", err)
		os.Exit(1)
	}
	os.Exit(0)
}
