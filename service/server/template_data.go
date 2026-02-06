package server

import "html/template"

type AppListItem struct {
	AppName           string
	Channel           string
	ChannelBadge      string
	ChannelConfigured bool
	Tooltip           string
	Hostname          string
	ChannelOptions    []SelectOption
}

type SelectOption struct {
	Value    string
	Label    string
	Selected bool
}

type IntegrationData struct {
	Name          string
	StatusClass   string
	StatusText    string
	StatusTooltip string
	Content       template.HTML
	Open          bool
	PollAttrs     string
}
