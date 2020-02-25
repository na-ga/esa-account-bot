package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

//
type MessageListener struct {
	slackClient        *slack.Client
	esaClient          *EsaClient
	repository         *Repository
	botID              string
	botName            string
	botUsageURL        string
	accountExpireMonth int
	channelID          string
}

//
func (s *MessageListener) Run() {
	rtm := s.slackClient.NewRTM()
	go rtm.ManageConnection()
	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			if err := s.handleMessageEvent(ev); err != nil {
				logger.Errorf("Failed to handle message: %s", err.Error())
				s.slackClient.PostMessage(s.channelID, slack.MsgOptionText(err.Error(), false)) // ignore post error
			}
		}
	}
}

// handleMessageEvent handles message events.
func (s *MessageListener) handleMessageEvent(ev *slack.MessageEvent) error {
	if ev.Channel != s.channelID {
		return nil
	}
	if !strings.HasPrefix(ev.Msg.Text, WrapUserNameInLink(s.botID)) {
		return nil
	}
	cmd := strings.Split(strings.TrimSpace(ev.Msg.Text), " ")
	if len(cmd) < 2 {
		return s.handleHelp(ev)
	}
	switch cmd[1] {
	case "admins":
		return s.handleAdmins(ev)
	case "invite":
		return s.handleInviteAccount(ev)
	case "delete":
		return s.handleDeleteAccount(ev)
	case "cleanup":
		return s.handleCleanupAccount(ev)
	default:
		return s.handleHelp(ev)
	}
}

//
func (s *MessageListener) handleAdmins(ev *slack.MessageEvent) error {
	var message string
	for _, name := range s.repository.GetAdminNames() {
		if message != "" {
			message += ", "
		}
		message += "@" + name // use plain text to not notify admins
	}
	ret := "Administrators:\n" + WrapTextInCodeBlock(message)
	if _, _, err := s.slackClient.PostMessage(ev.Channel, slack.MsgOptionAsUser(true), slack.MsgOptionText(ret, false)); err != nil {
		return fmt.Errorf("failed to post message: %s", err)
	}
	return nil
}

//
func (s *MessageListener) handleHelp(ev *slack.MessageEvent) error {
	messages := []string{
		fmt.Sprintf("%-40s : %s", "- @"+s.botName+" help", "利用可能なコマンド一覧を出力します。"),
		fmt.Sprintf("%-40s : %s", "- @"+s.botName+" admins", "承認を行える管理者一覧を出力します。"),
		fmt.Sprintf("%-40s : %s", "- @"+s.botName+" invite", "自身の Email 宛に招待リンクを送信します。管理者の承認が必要です。"),
		fmt.Sprintf("%-40s : %s", "- @"+s.botName+" invite [Email]", "指定した Email 宛に招待リンクを送信します。管理者の承認が必要です。"),
		fmt.Sprintf("%-40s : %s", "- @"+s.botName+" delete [ScreenName]", "指定した ScreenName のアカウントを削除します。管理者の承認が必要です。"),
		fmt.Sprintf("%-40s : %s", "- @"+s.botName+" cleanup", fmt.Sprintf("過去 %d ヶ月間アクセスしていないアカウントを削除します。管理者の承認が必要です。", s.accountExpireMonth)),
		fmt.Sprintf("%-40s : %s", "- @"+s.botName+" cleanup [Month]", "過去 Month ヶ月間アクセスしていないアカウントを削除します。管理者の承認が必要です。"),
	}
	ret := "Available commands:\n" + WrapTextsInCodeBlock(messages)
	if s.botUsageURL != "" {
		ret += "\nMore information: " + s.botUsageURL
	}
	if _, _, err := s.slackClient.PostMessage(ev.Channel, slack.MsgOptionAsUser(true), slack.MsgOptionText(ret, false)); err != nil {
		return fmt.Errorf("failed to post message: %s", err)
	}
	return nil
}

//
func (s *MessageListener) handleInviteAccount(ev *slack.MessageEvent) error {

	//
	user, err := s.slackClient.GetUserInfo(ev.User)
	if err != nil {
		return err
	}
	callback := Callback{
		ID:    s.repository.Callbacks().GenerateID(),
		Value: user.Profile.Email,
		OwnerUser: User{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Profile.Email,
		},
	}
	options := strings.Split(strings.TrimSpace(ev.Msg.Text), " ")[2:]
	if len(options) == 1 && options[0] != "" {
		callback.Value = RemoveMailtoMeta(options[0])
	}
	if err := s.repository.ValidEmail(callback.Value); err != nil {
		return err
	}

	//
	s.repository.Callbacks().Set(callback)
	organizations := s.repository.GetOrganizations()
	selectOrgOptions := make([]slack.AttachmentActionOption, len(organizations))
	for i, v := range organizations {
		selectOrgOptions[i] = slack.AttachmentActionOption{Text: v, Value: v}
	}
	text := "所属組織を選択してください"
	if callback.Value != callback.OwnerUser.Email {
		text = "招待するアカウントの所属組織を選択してください"
	}
	opts := []slack.MsgOption{
		slack.MsgOptionAsUser(true),
		slack.MsgOptionAttachments(slack.Attachment{
			Title:      DateTimePrefix() + "Select organization",
			Text:       text,
			Color:      ColorCodeBlue,
			CallbackID: callback.ID,
			Actions: []slack.AttachmentAction{
				{
					Name:    actionInviteSelectOrganization,
					Type:    "select",
					Options: selectOrgOptions,
					Style:   "primary",
				},
				{
					Name:  actionCancel,
					Text:  "Cancel",
					Type:  "button",
					Style: "danger",
				},
			},
		}),
	}
	if _, _, err := s.slackClient.PostMessage(ev.Channel, opts...); err != nil {
		return fmt.Errorf("failed to post message: %s", err)
	}
	return nil
}

//
func (s *MessageListener) handleDeleteAccount(ev *slack.MessageEvent) error {

	//
	user, err := s.slackClient.GetUserInfo(ev.User)
	if err != nil {
		return err
	}
	callback := Callback{
		ID:    s.repository.Callbacks().GenerateID(),
		Value: user.Profile.Email,
		OwnerUser: User{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Profile.Email,
		},
	}
	options := strings.Split(strings.TrimSpace(ev.Msg.Text), " ")[2:]
	if len(options) == 1 && options[0] != "" {
		callback.Value = options[0]
	}
	if callback.Value == "" {
		return fmt.Errorf("invalid ScreenName")
	}

	//
	s.repository.Callbacks().Set(callback)
	texts := []string{
		"Requester: " + WrapUserNameInLink(user.Name),
		"対象者のプロフィール: https://" + s.esaClient.GetTeamName() + ".esa.io/members/" + callback.Value,
	}
	opts := []slack.MsgOption{
		slack.MsgOptionAsUser(true),
		slack.MsgOptionAttachments(slack.Attachment{
			Title:      DateTimePrefix() + "Confirm",
			Text:       "アカウント削除申請の内容を確認してください\n" + WrapTextsInCodeBlock(texts),
			Color:      ColorCodeBlue,
			CallbackID: callback.ID,
			Actions: []slack.AttachmentAction{
				{
					Name:  actionDeleteConfirm,
					Text:  "OK, delete",
					Type:  "button",
					Style: "primary",
				},
				{
					Name:  actionCancel,
					Text:  "Cancel",
					Type:  "button",
					Style: "danger",
				},
			},
		}),
	}
	if _, _, err := s.slackClient.PostMessage(ev.Channel, opts...); err != nil {
		return fmt.Errorf("failed to post message: %s", err)
	}
	return nil
}

//
func (s *MessageListener) handleCleanupAccount(ev *slack.MessageEvent) error {

	//
	user, err := s.slackClient.GetUserInfo(ev.User)
	if err != nil {
		return err
	}

	var targetMonth int
	options := strings.Split(strings.TrimSpace(ev.Msg.Text), " ")[2:]
	if len(options) == 1 && options[0] != "" {
		targetMonth, err = strconv.Atoi(options[0])
		if err != nil || targetMonth < s.accountExpireMonth {
			return fmt.Errorf("invalid month, you must be at least %d: %d", s.accountExpireMonth, targetMonth)
		}
	} else {
		targetMonth = s.accountExpireMonth
	}

	// Search
	res, err := s.esaClient.ListAccount(QueryOptionSort("last_accessed"), QueryOptionOrder("asc"), QueryOptionPerPage(100))
	if err != nil {
		return fmt.Errorf("failed to get the target list that matches the conditions: %s", err.Error())
	}
	if len(res.Members) == 0 {
		ret := "No accounts matches the conditions"
		if _, _, err := s.slackClient.PostMessage(ev.Channel, slack.MsgOptionText(ret, false)); err != nil {
			return err
		}
		return nil
	}
	targets := make([]string, 0, len(res.Members))
	screenNames := make([]string, 0, len(res.Members))
	expireTime := time.Now().AddDate(0, -targetMonth, 0)
	for _, member := range res.Members {
		t, err := member.LastAccessedTime()
		if err != nil {
			logger.Errorf("account %s has unexpected last_accessed_at %s: %s", member.ScreenName, member.LastAccessedAt, err.Error())
			continue
		}
		if !expireTime.After(t) {
			logger.Debugf("No match condition: screenName=%s, lastAccess=%s, expire=%s", member.ScreenName, t, expireTime)
			break
		}
		screenNames = append(screenNames, member.ScreenName)
		targets = append(targets, fmt.Sprintf("- (%s) https://%s.esa.io/members/%s", member.LastAccessedAt[:10], s.esaClient.GetTeamName(), member.ScreenName))
	}
	if len(screenNames) == 0 {
		ret := "No accounts matches the conditions"
		if _, _, err := s.slackClient.PostMessage(ev.Channel, slack.MsgOptionText(ret, false)); err != nil {
			return err
		}
		return nil
	}

	//
	callback := Callback{
		ID:    s.repository.Callbacks().GenerateID(),
		Value: strings.Join(screenNames, ","),
		OwnerUser: User{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Profile.Email,
		},
	}

	//
	texts := []string{
		"Requester: " + WrapUserNameInLink(user.Name),
		fmt.Sprintf("Condition: 最終アクセス日時が %s 以前の期限切れアカウント (%d件) を削除します", expireTime.Format("2006/01/02"), len(screenNames)),
	}
	texts = append(texts, targets...)
	s.repository.Callbacks().Set(callback)
	opts := []slack.MsgOption{
		slack.MsgOptionAsUser(true),
		slack.MsgOptionAttachments(slack.Attachment{
			Title:      DateTimePrefix() + "Confirm",
			Text:       "期限切れアカウント削除申請の内容を確認してください\n" + WrapTextsInCodeBlock(texts),
			Color:      ColorCodeBlue,
			CallbackID: callback.ID,
			Actions: []slack.AttachmentAction{
				{
					Name:  actionCleanupConfirm,
					Text:  "OK, cleanup",
					Type:  "button",
					Style: "primary",
				},
				{
					Name:  actionCancel,
					Text:  "Cancel",
					Type:  "button",
					Style: "danger",
				},
			},
		}),
	}
	if _, _, err := s.slackClient.PostMessage(ev.Channel, opts...); err != nil {
		return fmt.Errorf("failed to post message: %s", err)
	}
	return nil
}
