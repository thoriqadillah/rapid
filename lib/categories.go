package lib

type Category string

const (
	CategoryAudio       Category = "audio"
	CategoryApplication Category = "application"
	CategoryCompressed  Category = "compressed"
	CategoryDocument    Category = "document"
	CategoryImage       Category = "image"
	CategoryUnknown     Category = "unknown"
	CategoryVideo       Category = "video"
)

func ToCategory(s string) Category {
	switch s {
	case "audio":
		return CategoryAudio
	case "application":
		return CategoryApplication
	case "compressed":
		return CategoryCompressed
	case "document":
		return CategoryDocument
	case "image":
		return CategoryImage
	case "unknown":
		return CategoryUnknown
	case "video":
		return CategoryVideo
	default:
		return CategoryUnknown
	}
}

func (c Category) String() string {
	str := string(c)
	if str == "" {
		return "unknown"
	}

	return str
}

func (c Category) Label() string {
	switch c {
	case CategoryAudio:
		return "Audio"
	case CategoryApplication:
		return "Application"
	case CategoryCompressed:
		return "Compressed"
	case CategoryDocument:
		return "Document"
	case CategoryImage:
		return "Image"
	case CategoryUnknown:
		return "Unknown"
	case CategoryVideo:
		return "Video"
	default:
		return "Unknown"
	}
}
