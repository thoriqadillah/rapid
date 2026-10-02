package notification

import (
	"fmt"

	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

// Kind identifies the semantic notification accent and event type.
type Kind string

const (
	KindInfo    Kind = "info"
	KindSuccess Kind = "success"
	KindError   Kind = "error"
)

type Event struct {
	Kind    Kind
	Title   string
	Message string
}

const (
	popupWidth       = 400
	popupGap         = theme.SpacingSm
	popupRightMargin = theme.SpacingSm
	popupBottom      = theme.SpacingSm
	popupTimerMS     = 5000
	popupBorderWidth = 1
	accentWidth      = 3
	popupEnterMS     = 200
	popupSlideMS     = 200
	popupFadeOutMS   = 200
)

// MoveTarget is the position a popup should slide to.
type MoveTarget struct {
	X, Y, Width, Height int
}

// Popup is the QML NotificationItem equivalent. It is owned by Manager after
// Show returns; callers should close it, not delete it.
type Popup struct {
	*qt.QFrame

	titleLabel   *qt.QLabel
	messageLabel *qt.QLabel
	accent       *qt.QFrame
	timer        *qt.QTimer

	kind             Kind
	title            string
	message          string
	typeColor        *qt.QColor
	opened           bool
	dismissed        bool
	dismissCallbacks []func()

	effect      *qt.QGraphicsOpacityEffect
	posAnim     *qt.QPropertyAnimation
	opacityAnim *qt.QPropertyAnimation
	positioned  bool
}

func NewPopup(parent *qt.QWidget) *Popup {
	p := &Popup{
		QFrame:    qt.NewQFrame(parent),
		kind:      KindInfo,
		typeColor: theme.ColorInfo,
	}
	p.SetObjectName(*qt.NewQAnyStringView3("notificationPopup"))
	p.SetFrameStyle(0)
	p.SetFixedWidth(popupWidth)
	p.SetSizePolicy2(qt.QSizePolicy__Fixed, qt.QSizePolicy__Preferred)
	p.SetMouseTracking(true)
	p.SetAttribute(qt.WA_StyledBackground)
	p.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))

	p.accent = qt.NewQFrame3(p.QWidget, 0)
	p.accent.SetObjectName(*qt.NewQAnyStringView3("notificationAccent"))
	p.accent.SetFixedWidth(accentWidth)
	p.accent.SetSizePolicy2(qt.QSizePolicy__Fixed, qt.QSizePolicy__Expanding)
	p.accent.SetAttribute(qt.WA_StyledBackground)

	p.titleLabel = qt.NewQLabel3("")
	p.titleLabel.SetTextFormat(qt.PlainText)
	p.titleLabel.SetWordWrap(false)
	p.titleLabel.SetObjectName(*qt.NewQAnyStringView3("notificationTitle"))
	p.titleLabel.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Fixed)
	p.titleLabel.SetAttribute(qt.WA_TransparentForMouseEvents)

	p.messageLabel = qt.NewQLabel3("")
	p.messageLabel.SetTextFormat(qt.PlainText)
	p.messageLabel.SetWordWrap(true)
	p.messageLabel.SetAlignment(qt.AlignLeft | qt.AlignVCenter)
	p.messageLabel.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Preferred)
	p.messageLabel.SetAttribute(qt.WA_TransparentForMouseEvents)
	// QLabel's word-wrap sizeHint measures at its intrinsic width (narrower than
	// the fixed card), over-reporting height and leaving a band of empty space
	// under the message. Report the height at the real wrapped width instead.
	// ponytail: exact up to a +/-2px wrap boundary at the fixed card width.
	p.messageLabel.OnSizeHint(func(super func() *qt.QSize) *qt.QSize {
		width := popupWidth - 2*popupBorderWidth - accentWidth - 2*theme.SpacingMd
		return qt.NewQSize2(width, p.messageLabel.HeightForWidth(width))
	})

	column := qt.NewQVBoxLayout2()
	column.SetContentsMargins(theme.SpacingMd, theme.SpacingSm, theme.SpacingMd, theme.SpacingSm)
	column.SetSpacing(theme.SpacingXs)
	column.AddWidget(p.titleLabel.QWidget)
	column.AddWidget(p.messageLabel.QWidget)

	row := qt.NewQHBoxLayout2()
	row.SetContentsMargins(0, 0, 0, 0)
	row.SetSpacing(0)
	row.AddWidget(p.accent.QWidget)
	row.AddLayout(column.QLayout)
	p.SetLayout(row.QLayout)

	p.timer = qt.NewQTimer2(p.QObject)
	p.timer.SetSingleShot(true)
	p.timer.SetInterval(popupTimerMS)
	p.timer.OnTimeout(p.Close)

	p.OnEnterEvent(func(super func(*qt.QEnterEvent), event *qt.QEnterEvent) {
		super(event)
		p.timer.Stop()
	})
	p.OnLeaveEvent(func(super func(*qt.QEvent), event *qt.QEvent) {
		super(event)
		if p.opened && !p.dismissed {
			p.timer.SetInterval(popupTimerMS)
			p.timer.Start2()
		}
	})
	p.OnMousePressEvent(func(super func(*qt.QMouseEvent), event *qt.QMouseEvent) {
		super(event)
		p.Close()
	})
	p.applyStyle()
	return p
}

func (p *Popup) SetKind(kind Kind) {
	p.kind = kind
	p.SetTypeColor(colorForKind(kind))
}

func (p *Popup) Kind() Kind {
	return p.kind
}

func (p *Popup) SetTitle(v string) {
	p.title = v
	p.titleLabel.SetText(v)
	p.AdjustSize()
}

func (p *Popup) Title() string {
	return p.title
}

func (p *Popup) SetMessage(v string) {
	p.message = v
	p.messageLabel.SetText(v)
	p.AdjustSize()
}

func (p *Popup) Message() string {
	return p.message
}

func (p *Popup) SetTypeColor(color *qt.QColor) {
	if color == nil {
		color = theme.ColorInfo
	}
	p.typeColor = color
	p.applyStyle()
}

func (p *Popup) TypeColor() *qt.QColor {
	return p.typeColor
}

func (p *Popup) OnDismissed(fn func()) {
	if fn != nil {
		p.dismissCallbacks = append(p.dismissCallbacks, fn)
	}
}

func (p *Popup) Open() {
	if p == nil {
		return
	}
	p.opened = true
	p.dismissed = false
	p.AdjustSize()
	if p.effect == nil {
		p.effect = qt.NewQGraphicsOpacityEffect2(p.QObject)
		p.effect.SetOpacity(0)
		p.SetGraphicsEffect(p.effect.QGraphicsEffect)
	}
	p.Show()
	p.Raise()
	p.timer.SetInterval(popupTimerMS)
	p.timer.Start2()
}

// MoveTo slides the popup to target. The first call also slides it in from
// the right while fading in; later calls slide vertically to the new spot as
// others dismiss.
func (p *Popup) MoveTo(target MoveTarget) {
	if p == nil {
		return
	}
	to := qt.NewQPoint2(target.X, target.Y)
	if !p.positioned {
		p.positioned = true
		from := qt.NewQPoint2(target.X+target.Width+popupGap, target.Y)
		p.slideTo(from, to)
		p.fadeTo(0, 1, popupEnterMS, qt.QEasingCurve__OutCubic, nil)
		return
	}
	p.slideTo(p.Pos(), to)
}

func (p *Popup) slideTo(from, to *qt.QPoint) {
	if p.posAnim != nil {
		p.posAnim.Stop()
	}
	p.posAnim = qt.NewQPropertyAnimation2(qt.UnsafeNewQObject(p.UnsafePointer()), []byte("pos"))
	p.posAnim.SetParent(p.QObject)
	p.posAnim.SetDuration(popupSlideMS)
	p.posAnim.SetStartValue(qt.NewQVariant24(from))
	p.posAnim.SetEndValue(qt.NewQVariant24(to))
	p.posAnim.SetEasingCurve(qt.NewQEasingCurve3(qt.QEasingCurve__InOutCubic))
	p.posAnim.Start()
}

func (p *Popup) fadeTo(from, to float64, duration int, easing qt.QEasingCurve__Type, done func()) {
	if p.opacityAnim != nil {
		p.opacityAnim.Stop()
	}
	p.opacityAnim = qt.NewQPropertyAnimation2(qt.UnsafeNewQObject(p.effect.UnsafePointer()), []byte("opacity"))
	p.opacityAnim.SetParent(p.QObject)
	p.opacityAnim.SetDuration(duration)
	p.opacityAnim.SetStartValue(qt.NewQVariant9(from))
	p.opacityAnim.SetEndValue(qt.NewQVariant9(to))
	p.opacityAnim.SetEasingCurve(qt.NewQEasingCurve3(easing))
	if done != nil {
		p.opacityAnim.OnFinished(done)
	}
	p.opacityAnim.Start()
}

func (p *Popup) Close() {
	if p == nil || p.dismissed {
		return
	}
	p.dismissed = true
	p.opened = false
	if p.timer != nil {
		p.timer.Stop()
	}
	if p.effect != nil {
		pos := p.Pos()
		to := qt.NewQPoint2(pos.X()+p.Width()+popupGap, pos.Y())
		p.slideTo(pos, to)
		p.fadeTo(1, 0, popupFadeOutMS, qt.QEasingCurve__InCubic, p.finishClose)
		return
	}
	p.finishClose()
}

func (p *Popup) finishClose() {
	p.Hide()
	p.DeleteLater()
	for _, fn := range p.dismissCallbacks {
		if fn != nil {
			fn()
		}
	}
}

func (p *Popup) Timer() *qt.QTimer {
	return p.timer
}

func (p *Popup) applyStyle() {
	if p == nil {
		return
	}
	card := theme.ColorSurface
	border := theme.ColorBorder
	text := theme.ColorText

	p.SetStyleSheet(fmt.Sprintf(`
		QFrame#notificationPopup {
			background-color: %s;
			border: 1px solid %s;
			border-radius: %dpx;
		}
		QFrame#notificationAccent {
			background-color: %s;
			border: none;
			border-radius: 1px;
		}
		QLabel#notificationTitle {
			font-weight: 500;
		}
		QLabel {
			background: transparent;
			border: none;
			color: %s;
			font-size: %dpx;
		}
	`,
		theme.CssColor(card),
		theme.CssColor(border),
		theme.RadiusSm,
		theme.CssColor(p.typeColor),
		theme.CssColor(text),
		theme.TextSize))
}

func colorForKind(kind Kind) *qt.QColor {
	switch kind {
	case KindError:
		return theme.ColorDanger
	case KindSuccess:
		return theme.ColorSuccess
	default:
		return theme.ColorInfo
	}
}
