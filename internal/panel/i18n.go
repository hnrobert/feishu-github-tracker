package panel

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const localeCookie = "fgt_locale"

//go:embed locales/*.json
var localesFS embed.FS

var messages = loadMessages()

func loadMessages() map[string]map[string]string {
	locales := map[string]map[string]string{}
	for _, name := range []string{"zh-CN", "en-US"} {
		data, err := localesFS.ReadFile("locales/" + name + ".json")
		if err != nil {
			panic("read panel locale " + name + ": " + err.Error())
		}
		catalog := map[string]string{}
		if err := json.Unmarshal(data, &catalog); err != nil {
			panic("parse panel locale " + name + ": " + err.Error())
		}
		locales[name] = catalog
	}
	return locales
}

func localeFrom(r *http.Request) string {
	if c, err := r.Cookie(localeCookie); err == nil && validLocale(c.Value) {
		return c.Value
	}
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Accept-Language")), "en") {
		return "en-US"
	}
	return "zh-CN"
}

func validLocale(locale string) bool {
	_, ok := messages[locale]
	return ok
}

func translate(data ViewData, key string) string {
	return message(data.Locale, key)
}

func (a *App) message(r *http.Request, key string, args ...any) string {
	return fmt.Sprintf(message(localeFrom(r), key), args...)
}

func message(locale, key string) string {
	if value := messages[locale][key]; value != "" {
		return value
	}
	if value := messages["zh-CN"][key]; value != "" {
		return value
	}
	return key
}

func alternateLocale(locale string) string {
	if locale == "en-US" {
		return "zh-CN"
	}
	return "en-US"
}
