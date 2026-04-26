package locales

import (
	"embed"
	_ "embed"
	"encoding/json"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed *.json
var localeFS embed.FS

var bundle *i18n.Bundle

func Init() (error, []string) {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	files := []string{
		"ru.json",
		"en.json",
	}

	for _, f := range files {
		_, err := bundle.LoadMessageFileFS(localeFS, f)
		if err != nil {
			return err, nil
		}
	}

	tags := bundle.LanguageTags()
	langs := make([]string, 0, len(tags))
	for _, lang := range tags {
		base, _ := lang.Base()
		langs = append(langs, base.String())
	}
	return nil, langs
}

func T(lang string, messageID string) string {
	localizer := i18n.NewLocalizer(bundle, lang)

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: messageID,
	})

	if err != nil {
		return ""
	}

	return msg
}
