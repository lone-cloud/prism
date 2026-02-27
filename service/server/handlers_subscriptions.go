package server

import (
	"fmt"
	"net/http"

	"prism/service/subscription"
	"prism/service/util"

	"github.com/go-chi/chi/v5"
)

type subscriptionFormData struct {
	Channel string
	GroupID string
	ChatID  string
}

func (s *Server) handleCreateSubscription(w http.ResponseWriter, r *http.Request) {
	appName := chi.URLParam(r, "appName")
	if appName == "" {
		http.Error(w, "Missing app parameter", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		util.JSONError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	form := subscriptionFormData{
		Channel: r.FormValue("channel"),
		GroupID: r.FormValue("group_id"),
		ChatID:  r.FormValue("chat_id"),
	}

	channel := subscription.Channel(form.Channel)
	if !s.publisher.IsValidChannel(channel) {
		util.SetToast(w, fmt.Sprintf("Invalid or unavailable channel: %s", form.Channel), "error")
		s.handleFragmentApps(w, r)
		return
	}

	app, err := s.store.GetApp(appName)
	if err != nil {
		util.LogAndError(w, s.logger, "Failed to load app subscriptions", http.StatusInternalServerError, err)
		return
	}
	if app != nil {
		for _, existingSub := range app.Subscriptions {
			if existingSub.Channel == channel {
				util.SetToast(w, fmt.Sprintf("%s already enabled", channel.Label()), "error")
				s.handleFragmentApps(w, r)
				return
			}
		}
	}

	sub := subscription.Subscription{
		AppName: appName,
		Channel: channel,
	}

	switch channel {
	case subscription.ChannelSignal:
		if s.integrations.Signal == nil || !s.integrations.Signal.IsEnabled() {
			util.SetToast(w, "Signal not configured", "error")
			s.handleFragmentApps(w, r)
			return
		}
		client := s.integrations.Signal.Handlers.Client
		account, err := client.GetLinkedAccount()
		if err != nil || account == nil {
			util.SetToast(w, "Signal not linked - configure its integration below", "error")
			s.handleFragmentApps(w, r)
			return
		}

		if form.GroupID != "" {
			sub.Signal = &subscription.SignalSubscription{
				GroupID: form.GroupID,
				Account: account.Number,
			}
		} else {
			cachedGroup, err := s.integrations.Signal.Groups.Get(appName)
			if err != nil {
				util.LogAndError(w, s.logger, "Failed to check for cached Signal group", http.StatusInternalServerError, err)
				return
			}

			if cachedGroup != nil && cachedGroup.Account == account.Number {
				sub.Signal = cachedGroup
			} else {
				signalSub, err := s.integrations.Signal.Sender.CreateDefaultSignalSubscription(appName)
				if err != nil {
					util.LogAndError(w, s.logger, "Failed to create Signal subscription", http.StatusInternalServerError, err)
					return
				}
				sub.Signal = signalSub
			}
		}

	case subscription.ChannelTelegram:
		if s.integrations.Telegram == nil || !s.integrations.Telegram.IsEnabled() {
			util.SetToast(w, "Telegram not configured", "error")
			s.handleFragmentApps(w, r)
			return
		}
		chatID := s.integrations.Telegram.Handlers.GetChatID()
		if chatID == 0 {
			util.SetToast(w, "Telegram not linked - configure its integration below", "error")
			s.handleFragmentApps(w, r)
			return
		}
		if form.ChatID != "" {
			chatID = 0
			fmt.Sscanf(form.ChatID, "%d", &chatID)
		}
		sub.Telegram = &subscription.TelegramSubscription{
			ChatID: fmt.Sprintf("%d", chatID),
		}

	default:
		util.JSONError(w, "Unsupported channel for manual subscription", http.StatusBadRequest)
		return
	}

	if _, err := s.store.AddSubscription(sub); err != nil {
		util.LogAndError(w, s.logger, "Failed to create subscription", http.StatusInternalServerError, err)
		return
	}

	util.SetToast(w, fmt.Sprintf("%s enabled", channel.Label()), "success")
	s.handleFragmentApps(w, r)
}

func (s *Server) handleDeleteSubscription(w http.ResponseWriter, r *http.Request) {
	subscriptionID := chi.URLParam(r, "subscriptionId")
	if subscriptionID == "" {
		http.Error(w, "Missing subscription ID", http.StatusBadRequest)
		return
	}

	sub, err := s.store.GetSubscription(subscriptionID)
	if err != nil {
		util.LogAndError(w, s.logger, "Failed to get subscription", http.StatusInternalServerError, err)
		return
	}

	if err := s.store.DeleteSubscription(subscriptionID); err != nil {
		util.LogAndError(w, s.logger, "Failed to delete subscription", http.StatusInternalServerError, err)
		return
	}

	message := fmt.Sprintf("%s channel disabled", sub.Channel.Label())
	if sub.Channel == subscription.ChannelWebPush {
		message = fmt.Sprintf("%s channel deleted", sub.Channel.Label())
	}
	util.SetToast(w, message, "success")
	s.handleFragmentApps(w, r)
}
