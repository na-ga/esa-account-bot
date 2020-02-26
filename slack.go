package main

import (
	"fmt"
	"strings"
	"time"
)

const (
	// actions
	actionInviteSelectOrganization = "inviteSelectOrganization"
	actionInviteConfirm            = "inviteConfirm"
	actionInviteApprove            = "inviteApprove"
	actionDeleteConfirm            = "deleteConfirm"
	actionDeleteApprove            = "deleteApprove"
	actionCleanupConfirm           = "cleanupConfirm"
	actionCleanupApprove           = "cleanupApprove"
	actionCancel                   = "cancel"
	actionReject                   = "reject"

	// color codes
	ColorCodeRed    = "#FF0000"
	ColorCodeOrange = "#FFA500"
	ColorCodeYellow = "#FFFF00"
	ColorCodeGreen  = "#00FF00"
	ColorCodeBlue   = "#0000FF"
)

// WrapTextInCodeBlock wraps a string into a code-block formatted string
func WrapTextInCodeBlock(text string) string {
	return fmt.Sprintf("```\n%s\n```", text)
}

// WrapTextsInCodeBlock wraps are strings into a code-block formatted string
func WrapTextsInCodeBlock(texts []string) string {
	return fmt.Sprintf("```\n%s\n```", strings.Join(texts, "\n"))
}

// WrapTextInInlineCodeBlock wraps a string into a inline-code-block formatted string
func WrapTextInInlineCodeBlock(text string) string {
	return fmt.Sprintf("`%s`", text)
}

// WrapUserNameInLink converts to a linkable user name
func WrapUserNameInLink(userName string) string {
	return fmt.Sprintf("<@%s>", userName)
}

// WrapUserGroupIDInLink converts to a linkable user group
func WrapUserGroupIDInLink(userGroupID string) string {
	return fmt.Sprintf("<!subteam^%s>", userGroupID)
}

// WrapTextInLink converts to a linkable string
func WrapTextInLink(des, link string) string {
	return fmt.Sprintf("<%s|%s>", link, des)
}

// RemoveMailtoMeta removes mailto meta and returns it.
func RemoveMailtoMeta(text string) string {
	prefix := "<mailto:"
	if strings.HasPrefix(text, prefix) {
		return strings.Split(text[len(prefix):], "|")[0]
	}
	return text
}

//
func DateTimePrefix() string {
	return time.Now().In(timeZone).Format("01/02 15:04") + " - "
}
