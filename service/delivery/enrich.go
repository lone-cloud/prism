package delivery

import (
	"fmt"
	"regexp"
	"strings"
)

var phoneRegex = regexp.MustCompile(`(?:\+?1[\s.-]?)?\(?\d{3}\)?[\s.-]?\d{3}[\s.-]?\d{4}`)
var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

func enrichActions(notif Notification) Notification {
	if len(notif.Actions) > 0 {
		return notif
	}

	seen := make(map[string]bool)

	for _, num := range phoneRegex.FindAllString(notif.Message, -1) {
		if seen[num] {
			continue
		}
		seen[num] = true
		notif.Actions = append(notif.Actions, Action{
			ID:       fmt.Sprintf("call-%s", num),
			Label:    fmt.Sprintf("Call %s", num),
			Endpoint: fmt.Sprintf("tel:%s", num),
		})
	}

	for _, addr := range emailRegex.FindAllString(notif.Message, -1) {
		if seen[addr] {
			continue
		}
		seen[addr] = true
		local := addr
		if i := strings.Index(addr, "@"); i != -1 {
			local = addr[:i]
		}
		notif.Actions = append(notif.Actions, Action{
			ID:       fmt.Sprintf("email-%s", addr),
			Label:    fmt.Sprintf("Email %s", local),
			Endpoint: fmt.Sprintf("mailto:%s", addr),
		})
	}

	return notif
}
