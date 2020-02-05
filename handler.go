package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"strings"

	"github.com/nlopes/slack"
)

// InteractionHandler handles interactive message response.
type InteractionHandler struct {
	esaClient         *EsaClient
	slackClient       *slack.Client
	repository        *Repository
	channelID         string
	verificationToken string
	adminIDs          []string
	adminGroupID      string
}

//
func (h InteractionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		logger.Errorf("Invalid method: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Errorf("Failed to read request body: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	body, err := url.QueryUnescape(string(buf)[8:])
	if err != nil {
		logger.Errorf("Failed to un-escape request body: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var message slack.InteractionCallback
	if err := json.Unmarshal([]byte(body), &message); err != nil {
		logger.Errorf("Failed to decode json message from slack: %s", body)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if message.Token != h.verificationToken {
		logger.Errorf("Invalid token: %s", message.Token)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if message.Channel.ID != h.channelID {
		logger.Errorf("Invalid channelId: %s", message.Channel.ID)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err := h.handle(w, message); err != nil {
		logger.Errorf("Failed to handle interactive messages: %s", err.Error())
		return
	}
}

//
func (h InteractionHandler) handle(w http.ResponseWriter, message slack.InteractionCallback) error {
	action := message.ActionCallback.AttachmentActions[0]
	switch action.Name {
	case actionInviteConfirm:
		return h.handleConfirm(w, message, actionInviteApprove)
	case actionInviteSelectOrganization:
		return h.handleInviteSelectOrganization(w, message)
	case actionInviteApprove:
		return h.handleInviteApprove(w, message)
	case actionDeleteConfirm:
		return h.handleConfirm(w, message, actionDeleteApprove)
	case actionDeleteApprove:
		return h.handleDeleteApprove(w, message)
	case actionCleanupConfirm:
		return h.handleConfirm(w, message, actionCleanupApprove)
	case actionCleanupApprove:
		return h.handleCleanupApprove(w, message)
	case actionCancel:
		return h.handleCancel(w, message)
	case actionReject:
		return h.handleReject(w, message)
	default:
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("invalid action was submitted: %s", action.Name)
	}
}

//
func (h InteractionHandler) handleCancel(w http.ResponseWriter, message slack.InteractionCallback) error {
	cb, ok := h.repository.Callbacks().Get(message.CallbackID)
	if !ok {
		text := ":x: Request has expired: " + message.CallbackID
		return h.responseError(w, message.OriginalMessage, text)
	}
	original := message.OriginalMessage
	if cb.OwnerUser.ID != message.User.ID {
		text := fmt.Sprintf(":warning: %s does not have cancel permission", WrapUserNameInLink(message.User.Name))
		return h.responseHint(w, original, text)
	}
	text := fmt.Sprintf(":x: %s canceled the request", WrapUserNameInLink(message.User.Name))
	return h.responseWarning(w, original, text)
}

//
func (h InteractionHandler) handleReject(w http.ResponseWriter, message slack.InteractionCallback) error {
	cb, ok := h.repository.Callbacks().Get(message.CallbackID)
	if !ok {
		text := ":x: Request has expired: " + message.CallbackID
		return h.responseError(w, message.OriginalMessage, text)
	}
	original := message.OriginalMessage
	if !h.repository.IsAdminUserID(message.User.ID) && cb.OwnerUser.ID != message.User.ID {
		text := fmt.Sprintf(":warning: %s does not have reject permission", WrapUserNameInLink(message.User.Name))
		return h.responseHint(w, original, text)
	}
	text := fmt.Sprintf(":x: %s rejected the request", WrapUserNameInLink(message.User.Name))
	return h.responseWarning(w, original, text)
}

//
func (h InteractionHandler) handleConfirm(w http.ResponseWriter, message slack.InteractionCallback, nextAction string) error {
	cb, ok := h.repository.Callbacks().Get(message.CallbackID)
	if !ok {
		text := ":x: Request has expired: " + message.CallbackID
		return h.responseError(w, message.OriginalMessage, text)
	}
	original := message.OriginalMessage
	if cb.OwnerUser.ID != message.User.ID {
		text := fmt.Sprintf(":warning: %s does not have confirm permission", WrapUserNameInLink(message.User.Name))
		return h.responseHint(w, original, text)
	}
	h.setSuccessToLastAttachment(original.Attachments, "")
	var admins string
	if h.adminGroupID != "" {
		admins = WrapUserGroupIDInLink(h.adminGroupID)
	} else {
		for _, v := range h.adminIDs {
			if admins == "" {
				admins = WrapUserGroupIDInLink(v)
			} else {
				admins += " " + WrapUserGroupIDInLink(v)
			}
		}
	}
	original.Attachments = append(original.Attachments, slack.Attachment{
		Title:      DateTimePrefix() + "Review",
		Text:       ":pray: 管理者 " + admins + " の承認が必要です",
		Color:      ColorCodeBlue,
		CallbackID: cb.ID,
		Actions: []slack.AttachmentAction{
			{
				Name:  nextAction,
				Text:  "Approve",
				Type:  "button",
				Style: "primary",
			},
			{
				Name:  actionReject,
				Text:  "Reject",
				Type:  "button",
				Style: "danger",
			},
		},
	})
	return h.response(w, &original)
}

//
func (h InteractionHandler) handleInviteSelectOrganization(w http.ResponseWriter, message slack.InteractionCallback) error {
	cb, ok := h.repository.Callbacks().Get(message.CallbackID)
	if !ok {
		text := ":x: Request has expired: " + message.CallbackID
		return h.responseError(w, message.OriginalMessage, text)
	}
	original := message.OriginalMessage
	if cb.OwnerUser.ID != message.User.ID {
		text := fmt.Sprintf(":warning: %s does not have select organization permission", WrapUserNameInLink(message.User.Name))
		return h.responseHint(w, original, text)
	}
	cb.Organization = message.ActionCallback.AttachmentActions[0].SelectedOptions[0].Value
	if cb.Organization == "" {
		text := ":warning: organization is required"
		return h.responseHint(w, original, text)
	}
	h.repository.Callbacks().Set(cb)
	texts := []string{
		"Requester: " + WrapUserNameInLink(cb.OwnerUser.Name),
		"招待メール送信先: " + cb.Value,
		"対象者の所属組織: " + cb.Organization,
	}
	original.Attachments = []slack.Attachment{
		{
			Title:      DateTimePrefix() + "Confirm",
			Text:       "アカウント招待申請の内容を確認してください\n" + WrapTextsInCodeBlock(texts),
			Color:      ColorCodeBlue,
			CallbackID: cb.ID,
			Actions: []slack.AttachmentAction{
				{
					Name:  actionInviteConfirm,
					Text:  "OK, invite",
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
		},
	}
	return h.response(w, &original)
}

//
func (h InteractionHandler) handleInviteApprove(w http.ResponseWriter, message slack.InteractionCallback) error {

	// Check
	cb, ok := h.repository.Callbacks().Get(message.CallbackID)
	if !ok {
		text := ":x: Request has expired: " + message.CallbackID
		return h.responseError(w, message.OriginalMessage, text)
	}
	original := message.OriginalMessage
	if !h.repository.IsAdminUserID(message.User.ID) {
		text := fmt.Sprintf(":warning: %s does not have approve permission", WrapUserNameInLink(message.User.Name))
		return h.responseHint(w, original, text)
	}
	text := fmt.Sprintf(":white_check_mark: %s approved the request", WrapUserNameInLink(message.User.Name))
	if err := h.responseSuccess(w, original, text); err != nil {
		return fmt.Errorf("failed to write message: %s", err.Error())
	}

	// interactive message は 3 秒以内に応答する必要があるため、メイン処理は非同期で行う
	go func() {
		logger.Infof("Starting invite account for %s", cb.Value)
		original.Attachments = append(original.Attachments, slack.Attachment{
			Color: ColorCodeBlue,
			Title: DateTimePrefix() + "Execute",
			Text:  ":car: Starting invite account ...",
		})
		h.slackClient.UpdateMessage(message.Channel.ID, message.MessageTs, slack.MsgOptionAttachments(original.Attachments...))
		if err := h.esaClient.InviteAccount(cb.Value); err != nil {
			logger.Errorf("Failed to invite account for %s: %s", cb.Value, err.Error())
			h.setErrorToLastAttachment(original.Attachments, fmt.Sprintf(":x: Failed to invite account for %s: %s", WrapTextInInlineCodeBlock(cb.Value), err.Error()))
			h.slackClient.UpdateMessage(message.Channel.ID, message.MessageTs, slack.MsgOptionAttachments(original.Attachments...))
			return
		}
		logger.Infof("Invitation email has been sent to %s", cb.Value)
		h.setSuccessToLastAttachment(original.Attachments, ":+1: 招待メールを確認し 72 時間以内にアカウント登録を行なってください")
		h.slackClient.UpdateMessage(message.Channel.ID, message.MessageTs, slack.MsgOptionAttachments(original.Attachments...))
	}()
	return nil
}

//
func (h InteractionHandler) handleDeleteApprove(w http.ResponseWriter, message slack.InteractionCallback) error {

	// Check
	cb, ok := h.repository.Callbacks().Get(message.CallbackID)
	if !ok {
		text := ":x: Request has expired: " + message.CallbackID
		return h.responseError(w, message.OriginalMessage, text)
	}
	original := message.OriginalMessage
	if !h.repository.IsAdminUserID(message.User.ID) {
		text := fmt.Sprintf(":warning: %s does not have approve permission", WrapUserNameInLink(message.User.Name))
		return h.responseHint(w, original, text)
	}
	text := fmt.Sprintf(":white_check_mark: %s approved the request", WrapUserNameInLink(message.User.Name))
	if err := h.responseSuccess(w, original, text); err != nil {
		return fmt.Errorf("failed to write message: %s", err.Error())
	}

	// interactive message は 3 秒以内に応答する必要があるため、メイン処理は非同期で行う
	go func() {
		logger.Infof("Starting invite account for %s", cb.Value)
		original.Attachments = append(original.Attachments, slack.Attachment{
			Color: ColorCodeBlue,
			Title: DateTimePrefix() + "Execute",
			Text:  ":car: Starting delete account ...",
		})
		h.slackClient.UpdateMessage(message.Channel.ID, message.MessageTs, slack.MsgOptionAttachments(original.Attachments...))
		if err := h.esaClient.DeleteAccount(cb.Value); err != nil {
			logger.Errorf("Failed to delete account %s: %s", cb.Value, err.Error())
			h.setErrorToLastAttachment(original.Attachments, fmt.Sprintf(":x: Failed to delete account %s: %s", WrapTextInInlineCodeBlock(cb.Value), err.Error()))
			h.slackClient.UpdateMessage(message.Channel.ID, message.MessageTs, slack.MsgOptionAttachments(original.Attachments...))
			return
		}
		logger.Infof("Account %s has been deleted", cb.Value)
		results := []string{
			fmt.Sprintf("対象アカウント %s を削除しました", cb.Value),
			fmt.Sprintf("- https://%s.esa.io/team?keyword=%s", h.esaClient.GetTeamName(), cb.Value),
		}
		h.setSuccessToLastAttachment(original.Attachments, fmt.Sprintf(":+1: Account has been deleted\n%s", WrapTextsInCodeBlock(results)))
		h.slackClient.UpdateMessage(message.Channel.ID, message.MessageTs, slack.MsgOptionAttachments(original.Attachments...))
	}()
	return nil
}

//
func (h InteractionHandler) handleCleanupApprove(w http.ResponseWriter, message slack.InteractionCallback) error {

	// Check
	cb, ok := h.repository.Callbacks().Get(message.CallbackID)
	if !ok {
		text := ":x: Request has expired: " + message.CallbackID
		return h.responseError(w, message.OriginalMessage, text)
	}
	original := message.OriginalMessage
	if !h.repository.IsAdminUserID(message.User.ID) {
		text := fmt.Sprintf(":warning: %s does not have approve permission", WrapUserNameInLink(message.User.Name))
		return h.responseHint(w, original, text)
	}
	text := fmt.Sprintf(":white_check_mark: %s approved the request", WrapUserNameInLink(message.User.Name))
	if err := h.responseSuccess(w, original, text); err != nil {
		return fmt.Errorf("failed to write message: %s", err.Error())
	}

	// interactive message は 3 秒以内に応答する必要があるため、メイン処理は非同期で行う
	go func() {
		logger.Infof("Starting delete expired account (%s)", cb.Value)
		original.Attachments = append(original.Attachments, slack.Attachment{
			Color: ColorCodeBlue,
			Title: DateTimePrefix() + "Execute",
			Text:  ":car: Starting delete expired account ...",
		})
		h.slackClient.UpdateMessage(message.Channel.ID, message.MessageTs, slack.MsgOptionAttachments(original.Attachments...))
		targets := strings.Split(cb.Value, ",")
		results := make([]string, 0, len(targets)+1)
		results = append(results, fmt.Sprintf("期限切れアカウント (%d件) を削除しました", len(targets)))
		for _, target := range targets {
			logger.Infof("Try to delete expired account (%s)", target)
			if err := h.esaClient.DeleteAccount(target); err != nil {
				logger.Errorf("Failed to delete expired account %s: %s", target, err.Error())
				h.setErrorToLastAttachment(original.Attachments, fmt.Sprintf(":x: Failed to delete expired account %s: %s", WrapTextInInlineCodeBlock(target), err.Error()))
				h.slackClient.UpdateMessage(message.Channel.ID, message.MessageTs, slack.MsgOptionAttachments(original.Attachments...))
				return
			}
			results = append(results, fmt.Sprintf("- https://%s.esa.io/team?keyword=%s", h.esaClient.GetTeamName(), target))
		}
		logger.Infof("Expired account has been deleted (%s)", cb.Value)
		h.setSuccessToLastAttachment(original.Attachments, fmt.Sprintf(":+1: Expired account has been deleted\n%s", WrapTextsInCodeBlock(results)))
		h.slackClient.UpdateMessage(message.Channel.ID, message.MessageTs, slack.MsgOptionAttachments(original.Attachments...))
	}()
	return nil
}

//
func (h InteractionHandler) response(w http.ResponseWriter, msg *slack.Message) error {
	w.Header().Add("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	msg.ReplaceOriginal = true
	return json.NewEncoder(w).Encode(msg)
}

//
func (h InteractionHandler) responseHint(w http.ResponseWriter, msg slack.Message, text string) error {
	in := msg.Attachments
	if last := len(in) - 1; last >= 0 {
		in[last].Color = ColorCodeOrange
		in[last].Fields = append(in[last].Fields, slack.AttachmentField{Value: text})
	}
	return h.response(w, &msg)
}

//
func (h InteractionHandler) responseWarning(w http.ResponseWriter, msg slack.Message, text string) error {
	h.setWarningToLastAttachment(msg.Attachments, text)
	return h.response(w, &msg)
}

//
func (h InteractionHandler) responseError(w http.ResponseWriter, msg slack.Message, text string) error {
	h.setErrorToLastAttachment(msg.Attachments, text)
	return h.response(w, &msg)
}

//
func (h InteractionHandler) responseSuccess(w http.ResponseWriter, msg slack.Message, text string) error {
	h.setSuccessToLastAttachment(msg.Attachments, text)
	return h.response(w, &msg)
}

//
func (h InteractionHandler) setToLastAttachment(in []slack.Attachment, color string, text string) {
	if last := len(in) - 1; last >= 0 {
		in[last].Color = color
		in[last].Actions = []slack.AttachmentAction{}
		in[last].Fields = []slack.AttachmentField{}
		if text != "" {
			in[last].Text = text
		}
	}
}

//
func (h InteractionHandler) setErrorToLastAttachment(in []slack.Attachment, text string) {
	h.setToLastAttachment(in, ColorCodeRed, text)
}

//
func (h InteractionHandler) setWarningToLastAttachment(in []slack.Attachment, text string) {
	h.setToLastAttachment(in, ColorCodeYellow, text)
}

//
func (h InteractionHandler) setSuccessToLastAttachment(in []slack.Attachment, text string) {
	h.setToLastAttachment(in, ColorCodeGreen, text)
}
