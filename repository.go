package main

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/nlopes/slack"
)

//
type Repository struct {
	slackClient       *slack.Client
	callbacks         *CallbackMap
	admins            map[string]User
	allowEmailDomains map[string]struct{}
	organizationList  []string
}

//
type User struct {
	ID    string
	Name  string
	Email string
}

//
func NewRepository(slackClient *slack.Client, adminIDs []string, allowEmailDomains []string, organizations []string) (*Repository, error) {
	admins := make(map[string]User, len(adminIDs))
	for _, v := range adminIDs {
		user, err := slackClient.GetUserInfo(v)
		if err != nil {
			logger.Errorf("Failed to get admin user profile: %s", err.Error())
			continue
		}
		admin := User{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Profile.Email,
		}
		admins[v] = admin
	}
	if len(admins) == 0 {
		return nil, errors.New("empty admins")
	}
	if len(organizations) == 0 {
		organizations = []string{"Other"}
	}
	domains := make(map[string]struct{}, len(allowEmailDomains))
	for _, v := range allowEmailDomains {
		domains[v] = struct{}{}
	}
	return &Repository{
		callbacks:         NewCallbackMap(),
		slackClient:       slackClient,
		admins:            admins,
		allowEmailDomains: domains,
		organizationList:  organizations,
	}, nil
}

//
func (r *Repository) Callbacks() *CallbackMap {
	return r.callbacks
}

//
func (r *Repository) IsAdminUserID(userID string) bool {
	_, ok := r.admins[userID]
	return ok
}

//
func (r *Repository) GetAdminNames() []string {
	ret := make([]string, 0, len(r.admins))
	for _, admin := range r.admins {
		ret = append(ret, admin.Name)
	}
	return ret
}

//
func (r *Repository) GetOrganizations() []string {
	copied := make([]string, len(r.organizationList))
	copy(copied, r.organizationList)
	return copied
}

//
func (r *Repository) ValidEmail(email string) error {
	if !govalidator.IsEmail(email) {
		return fmt.Errorf("invalid email: %s", WrapTextInInlineCodeBlock(email))
	}
	if len(r.allowEmailDomains) == 0 {
		return nil
	}
	domain := strings.Split(email, "@")[1]
	if _, ok := r.allowEmailDomains[domain]; ok {
		return nil
	}
	var hint string
	for v := range r.allowEmailDomains {
		if hint != "" {
			hint += " or "
		}
		hint += WrapTextInInlineCodeBlock(v)
	}
	return fmt.Errorf("invalid email domain, you must use %s: %s", hint, WrapTextInInlineCodeBlock(email))
}

//
func NewCallbackMap() *CallbackMap {
	return &CallbackMap{
		values:  map[string]Callback{},
		timeNow: time.Now,
	}
}

//
type CallbackMap struct {
	mu      sync.Mutex
	values  map[string]Callback
	timeNow func() time.Time
}

//
type Callback struct {
	ID           string
	Value        string
	Organization string
	OwnerUser    User
}

//
func (cm *CallbackMap) GenerateID() string {
	return cm.timeNow().Format(time.RFC3339Nano)
}

//
func (cm *CallbackMap) Set(value Callback) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.values[value.ID] = value
}

//
func (cm *CallbackMap) Get(key string) (Callback, bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.cleanup()
	value, ok := cm.values[key]
	return value, ok
}

//
func (cm *CallbackMap) cleanup() {
	for key := range cm.values {
		t, err := time.Parse(time.RFC3339Nano, key)
		if err != nil {
			delete(cm.values, key)
			continue
		}
		if cm.timeNow().Sub(t) > time.Hour*24*7 { // 1 week
			delete(cm.values, key)
			continue
		}
	}
}
