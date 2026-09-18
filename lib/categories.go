package lib

type Category string

const (
	Audio       Category = "audio"
	Application Category = "application"
	Compressed  Category = "compressed"
	Document    Category = "document"
	Image       Category = "image"
	Unknown     Category = "unknown"
	Video       Category = "video"
)

func (c Category) String() string {
	return string(c)
}

func (c Category) Label() string {
	switch c {
	case Audio:
		return "Audio"
	case Application:
		return "Application"
	case Compressed:
		return "Compressed"
	case Document:
		return "Document"
	case Image:
		return "Image"
	case Unknown:
		return "Unknown"
	case Video:
		return "Video"
	default:
		return "Unknown"
	}
}
