package server

import (
	"bytes"
	"net/http"

	"prism/service/integration/signal"
	"prism/service/subscription"
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

func (s *Server) buildAppListData(apps []subscription.App) []AppListItem {
	if len(apps) == 0 {
		return nil
	}

	signalEnabled := s.integrations.IsSignalLinked()
	telegramEnabled := s.integrations.IsTelegramLinked()
	telegramBotName := ""
	if telegramEnabled {
		handlers := s.integrations.Telegram.Handlers
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
			var signalSub *subscription.Subscription
			for i := range app.Subscriptions {
				if app.Subscriptions[i].Channel == subscription.ChannelSignal {
					signalSub = &app.Subscriptions[i]
					break
				}
			}

			state := ChannelState{
				Channel:    string(subscription.ChannelSignal),
				Label:      subscription.ChannelSignal.Label(),
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
			var telegramSub *subscription.Subscription
			for i := range app.Subscriptions {
				if app.Subscriptions[i].Channel == subscription.ChannelTelegram {
					telegramSub = &app.Subscriptions[i]
					break
				}
			}

			state := ChannelState{
				Channel:    string(subscription.ChannelTelegram),
				Label:      subscription.ChannelTelegram.Label(),
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
			if sub.Channel == subscription.ChannelWebPush && sub.WebPush != nil {
				webPushSubs = append(webPushSubs, SubscriptionItem{
					ID: sub.ID,
				})
			}
		}

		if len(webPushSubs) > 0 {
			channels = append(channels, ChannelState{
				Channel:       string(subscription.ChannelWebPush),
				Label:         subscription.ChannelWebPush.Label(),
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
