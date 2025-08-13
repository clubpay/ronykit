package i18n

import (
	"context"
	"io"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	ctxLangKey  = struct{}{}
	defaultLang = language.English
	bundles     = make(map[language.Tag]Bundle)
)

func init() {
	InitLanguage(defaultLang)
}

type Bundle struct {
	p *message.Printer
}

func GetBundle(lang language.Tag) Bundle {
	b, ok := bundles[lang]
	if !ok {
		b = bundles[defaultLang]
	}

	return b
}

func InitLanguage(lang language.Tag) Bundle {
	bundles[lang] = Bundle{
		p: message.NewPrinter(lang),
	}

	return bundles[lang]
}

func InjectContext(ctx context.Context, lang language.Tag) context.Context {
	return context.WithValue(ctx, ctxLangKey, lang)
}

func SetDefaultLanguage(lang language.Tag) {
	defaultLang = lang
}

func (b Bundle) Sprintf(msg message.Reference, args ...any) string {
	return b.p.Sprintf(msg, args...)
}

func (b Bundle) Fprintf(w io.Writer, key message.Reference, args ...any) (int, error) {
	return b.p.Fprintf(w, key, args...)
}

func Text(msg message.Reference, args ...any) string {
	return GetBundle(defaultLang).Sprintf(msg, args...)
}

func TextCtx(ctx context.Context, msg message.Reference, args ...any) string {
	lang := ctx.Value(ctxLangKey).(language.Tag)

	return GetBundle(lang).Sprintf(msg, args...)
}

func TextLang(lang language.Tag, msg message.Reference, args ...any) string {
	return GetBundle(lang).Sprintf(msg, args...)
}

func Txt(msg message.Reference, args ...any) string {
	return Text(msg, args...)
}

func TxtX(ctx context.Context, msg message.Reference, args ...any) string {
	return TextCtx(ctx, msg, args...)
}

func TxtL(lang language.Tag, msg message.Reference, args ...any) string {
	return TextLang(lang, msg, args...)
}
