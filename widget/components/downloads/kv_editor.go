package downloads

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"rapid/lib/helpers/iter"
	"rapid/lib/reactive"
	"rapid/widget/theme"
	"rapid/widget/ui"

	qt "github.com/mappu/miqt/qt6"
)

// KV is one key/value pair of an editor row.
type KV struct {
	Key, Value string
}

type kvRow struct {
	key, value *ui.RTextField
	host       *qt.QWidget
}

type rows []*kvRow

func (r rows) value() []KV {
	out := make([]KV, 0, len(r))
	for _, row := range r {
		key := strings.TrimSpace(row.key.Text())
		if key == "" {
			continue
		}
		out = append(out, KV{
			Key:   key,
			Value: strings.TrimSpace(row.value.Text()),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type KVEditor struct {
	*qt.QWidget

	rows     rows
	rowsBox  *qt.QVBoxLayout
	kvMode   *qt.QWidget
	jsonArea *ui.RTextArea
	jsonSw   *ui.RSwitch
	value    reactive.Model[[]KV]
	jsonErr  reactive.Model[string]
}

func NewKVEditor(parent *qt.QWidget, title string) *KVEditor {
	e := &KVEditor{
		QWidget: qt.NewQWidget3(parent, 0),
		value:   reactive.NewSignal([]KV(nil)),
		jsonErr: reactive.NewSignal(""),
	}

	jsonValue := reactive.NewWritableComputed(
		func() string {
			kvs := e.value.Get()
			return toJSON(kvs)
		},
		func(json string) {
			e.jsonErr.Set("")
			rows, err := toRows(json)
			if err != nil {
				e.jsonErr.Set("Invalid JSON")
				return
			}
			e.setRows(rows)
		},
	)

	box := qt.NewQVBoxLayout2()
	box.SetContentsMargins(0, 0, 0, 0)
	box.SetSpacing(theme.SpacingSm)

	// Title row: title on the left, JSON label + switch on the right. The
	// switch exists even when the title is empty (QML always shows it).
	titleRow := qt.NewQHBoxLayout2()
	titleRow.SetContentsMargins(0, 0, 0, 0)
	titleRow.SetSpacing(theme.SpacingSm)
	titleRow.AddWidget2(styledLabel(title, theme.ColorTextMuted).QWidget, 1)
	titleRow.AddWidget(smallLabel("JSON", theme.ColorTextMuted).QWidget)
	e.jsonSw = ui.NewRSwitch(false)
	e.jsonSw.OnToggled(e.onToggle)
	titleRow.AddWidget(e.jsonSw.QWidget)
	box.AddLayout(titleRow.QLayout)

	e.OnDestroyed(reactive.Effect(func() {
		jsonErr := e.jsonErr.Get()
		_, hasEmpty := iter.Find(e.value.Get(), func(kv KV) bool {
			return kv.Key == "" || kv.Value == ""
		})

		e.jsonSw.SetDisabled(jsonErr != "" || hasEmpty)
	}))

	// Rows mode: one widget per row, then divider/"+" footer.
	rows := qt.NewQWidget3(nil, 0)
	e.rowsBox = qt.NewQVBoxLayout2()
	e.rowsBox.SetContentsMargins(0, 0, 0, 0)
	e.rowsBox.SetSpacing(theme.SpacingSm)
	rows.SetLayout(e.rowsBox.QLayout)

	divider := qt.NewQWidget3(nil, 0)
	divider.SetFixedHeight(1)
	divider.SetStyleSheet("background-color: " + theme.CssColor(theme.WithAlpha(theme.ColorBorder, 102)) + ";")
	plus := ui.NewRButtonIcon("MdiLightPlus.svg", ui.GhostVariant, false)
	plus.OnClicked(func() { e.addRow(KV{}) })
	footerRow := qt.NewQHBoxLayout2()
	footerRow.SetContentsMargins(0, 0, 0, 0)
	footerRow.SetSpacing(theme.SpacingMd)
	footerRow.AddWidget2(divider, 1)
	footerRow.AddWidget(plus.QWidget)

	e.kvMode = qt.NewQWidget3(nil, 0)
	kvBox := qt.NewQVBoxLayout2()
	kvBox.SetContentsMargins(0, 0, 0, 0)
	kvBox.SetSpacing(theme.SpacingSm)
	kvBox.AddWidget(rows)
	kvBox.AddLayout(footerRow.QLayout)
	e.kvMode.SetLayout(kvBox.QLayout)
	box.AddWidget(e.kvMode)

	e.jsonArea = ui.NewRTextArea()
	e.jsonArea.SetFixedHeight(180)
	e.jsonArea.SetMonospace(true)
	e.jsonArea.SetPlaceholder("{}")
	e.jsonArea.Bind(jsonValue)
	e.jsonArea.BindError(e.jsonErr)
	box.AddWidget(e.jsonArea.QWidget)

	box.AddStretch()
	e.SetLayout(box.QLayout)
	e.setMode(false)
	e.ensureRow()
	return e
}

func (e *KVEditor) Bind(model reactive.Model[[]KV]) {
	modelValue, cleanup := reactive.Bind(model)
	e.value = modelValue
	unsub := reactive.Effect(func() {
		value := modelValue.Get()
		if !slices.Equal(value, e.rows.value()) {
			e.setRows(value)
		}
	})
	e.OnDestroyed(func() {
		unsub()
		cleanup()
	})
}

func (e *KVEditor) setRows(rows []KV) {
	for i := len(rows); i < len(e.rows); i++ {
		e.dropRow(i)
	}

	for _, kv := range rows {
		e.addRow(kv)
	}

	e.value.Set(rows)
}

func (e *KVEditor) dropRow(i int) {
	r := e.rows[i]
	if item := e.rowsBox.TakeAt(i); item != nil {
		item.Delete()
	}
	r.host.DeleteLater()
	e.rows = append(e.rows[:i], e.rows[i+1:]...)
}

func (e *KVEditor) addRow(kv KV) {
	r := &kvRow{
		key:   ui.NewRTextField(),
		value: ui.NewRTextField(),
	}

	r.key.SetPlaceholder("Key")
	r.value.SetPlaceholder("Value")
	r.key.OnTextChanged(func(string) { e.value.Set(e.rows.value()) })
	r.value.OnTextChanged(func(string) { e.value.Set(e.rows.value()) })
	r.key.SetText(kv.Key)
	r.value.SetText(kv.Value)

	remove := ui.NewRButtonIcon("MaterialSymbolsLightCloseRounded.svg", ui.GhostVariant, false)

	r.host = qt.NewQWidget3(nil, 0)
	box := qt.NewQHBoxLayout2()
	box.SetContentsMargins(0, 0, 0, 0)
	box.SetSpacing(theme.SpacingMd)
	box.AddWidget2(r.key.QWidget, 1)
	box.AddWidget2(r.value.QWidget, 1)
	box.AddWidget(remove.QWidget)
	r.host.SetLayout(box.QLayout)

	e.rows = append(e.rows, r)
	remove.OnClicked(func() {
		i := iter.FindIndex(e.rows, func(row *kvRow) bool { return row == r })
		e.handleDelete(i)
	})

	e.rowsBox.AddWidget(r.host)
}

func (e *KVEditor) handleDelete(i int) {
	if len(e.rows) <= 1 {
		e.rows[0].key.SetText("")
		e.rows[0].value.SetText("")
		return
	}

	e.dropRow(i)
	e.value.Set(e.rows.value())
}

func (e *KVEditor) ensureRow() {
	if len(e.rows) == 0 {
		e.addRow(KV{})
	}
}

func (e *KVEditor) onToggle(on bool) {
	e.setMode(on)
	e.ensureRow()
}

func (e *KVEditor) setMode(useJson bool) {
	if useJson {
		e.kvMode.SetVisible(false)
		e.jsonArea.SetVisible(true)
	} else {
		e.jsonArea.SetVisible(false)
		e.kvMode.SetVisible(true)
	}

	// Top-level windows only grow on layout changes, so without an explicit
	// re-fit the dialog keeps the taller JSON height after switching back to
	// rows. UpdateGeometry is required: AdjustSize alone reads the stale hint.
	if top := e.QWidget.Window(); top != nil && top.Inherits("QDialog") {
		e.QWidget.UpdateGeometry()
		top.AdjustSize()
	}
}

func toJSON(kvs []KV) string {
	m := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		m[kv.Key] = kv.Value
	}

	b, _ := json.MarshalIndent(m, "", "  ")
	return string(b)
}

func toRows(text string) ([]KV, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, err
	}
	out := make([]KV, 0, len(raw))
	for k, v := range raw {
		switch t := v.(type) {
		case string:
			out = append(out, KV{Key: k, Value: t})
		case float64:
			out = append(out, KV{Key: k, Value: strconv.FormatFloat(t, 'g', -1, 64)})
		case bool:
			out = append(out, KV{Key: k, Value: strconv.FormatBool(t)})
		case nil:
			out = append(out, KV{Key: k, Value: ""})
		default:
			return nil, fmt.Errorf("%s: value must be a string, number or boolean", k)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}
