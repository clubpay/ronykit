package mcp

type Option = func(*bundle)

func WithName(name string) Option {
	return func(b *bundle) {
		b.name = name
	}
}

func WithTitle(title string) Option {
	return func(b *bundle) {
		b.title = title
	}
}

func WithWebsiteURL(url string) Option {
	return func(b *bundle) {
		b.websiteURL = url
	}
}

func WithInstructions(instructions string) Option {
	return func(b *bundle) {
		b.instructions = instructions
	}
}
