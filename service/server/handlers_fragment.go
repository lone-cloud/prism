package server

import (
	"bytes"
	"net/http"

	"prism/service/integration/signal"
	"prism/service/notification"
	"prism/service/util"
)

type AppListItem struct {
	AppName  string
	Channels []ChannelState
}

type ChannelState struct {
	Channel       string
	Label         string
	Active        bool
	Toggleable    bool
	Subscriptions []SubscriptionItem
}

type SubscriptionItem struct {
	ID       string
	Tooltip  string
	Hostname string
}

func (s *Server) handleFragmentApps(w http.ResponseWriter, r *http.Request) {
	apps, err := s.store.GetAllApps()
	if err != nil {
		util.LogAndError(w, s.logger, "Internal server error", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	data := s.buildAppListData(apps)

	var buf bytes.Buffer
	if err := s.fragmentTmpl.ExecuteTemplate(&buf, "app-list.html", data); err != nil {
		util.LogAndError(w, s.logger, "Failed to execute template", http.StatusInternalServerError, err)
		return
	}

	_, _ = w.Write(buf.Bytes())
}

func (s *Server) buildAppListData(apps []notification.App) []AppListItem {
	if len(apps) == 0 {
		return nil
	}

	signalEnabled := s.integrations.Signal != nil && s.integrations.Signal.IsEnabled()
	telegramEnabled := s.integrations.Telegram != nil && s.integrations.Telegram.IsEnabled()
	telegramBotName := ""
	if telegramEnabled {
		handlers := s.integrations.Telegram.GetHandlers()
		if handlers != nil {
			client := handlers.GetClient()
			if client != nil {
				if bot, err := client.GetMe(); err == nil && bot != nil && bot.Username != "" {
					telegramBotName = "@" + bot.Username
				}
			}
		}
	}

	items := make([]AppListItem, 0, len(apps))
	for _, app := range apps {
		channels := []ChannelState{}

		if signalEnabled {
			var signalSub *notification.Subscription
			for i := range app.Subscriptions {
				if app.Subscriptions[i].Channel == notification.ChannelSignal {
					signalSub = &app.Subscriptions[i]
					break
				}
			}

			state := ChannelState{
				Channel:    string(notification.ChannelSignal),
				Label:      notification.ChannelSignal.Label(),
				Active:     signalSub != nil,
				Toggleable: true,
			}

			if signalSub != nil && signalSub.Signal != nil {
				state.Subscriptions = []SubscriptionItem{{
					ID:      signalSub.ID,
					Tooltip: signal.FormatPhoneNumber(signalSub.Signal.Account),
				}}
			}

			channels = append(channels, state)
		}

		if telegramEnabled {
			var telegramSub *notification.Subscription
			for i := range app.Subscriptions {
				if app.Subscriptions[i].Channel == notification.ChannelTelegram {
					telegramSub = &app.Subscriptions[i]
					break
				}
			}

			state := ChannelState{
				Channel:    string(notification.ChannelTelegram),
				Label:      notification.ChannelTelegram.Label(),
				Active:     telegramSub != nil,
				Toggleable: true,
			}

			if telegramSub != nil {
				state.Subscriptions = []SubscriptionItem{{
					ID:      telegramSub.ID,
					Tooltip: telegramBotName,
				}}
			}

			channels = append(channels, state)
		}

		webPushSubs := []SubscriptionItem{}
		for _, sub := range app.Subscriptions {
			if sub.Channel == notification.ChannelWebPush && sub.WebPush != nil {
				webPushSubs = append(webPushSubs, SubscriptionItem{
					ID:      sub.ID,
					Tooltip: sub.WebPush.Endpoint,
				})
			}
		}

		if len(webPushSubs) > 0 {
			channels = append(channels, ChannelState{
				Channel:       string(notification.ChannelWebPush),
				Label:         notification.ChannelWebPush.Label(),
				Active:        true,
				Toggleable:    false,
				Subscriptions: webPushSubs,
			})
		}

		item := AppListItem{
			AppName:  app.AppName,
			Channels: channels,
		}
		items = append(items, item)
	}
	return items
}
