package codegen

import (
	"fmt"
	"strings"
	"text/template"
)

type strErrBuf struct {
	b   strings.Builder
	err error
}

// Appends a string to the buffer if no error is already present, or captures the provided error; once an error is set, subsequent calls have no effect.
func (m *strErrBuf) printlne(str string, err error) {
	if m.err == nil {
		if err == nil {
			_, m.err = fmt.Fprintln(&m.b, str)
		} else {
			m.err = err
		}
	}
}

// Appends a formatted string followed by a newline to the buffer if no prior error exists, and updates the error state if writing fails.
func (m *strErrBuf) printlnf(format string, args ...any) {
	if m.err == nil {
		_, m.err = fmt.Fprintf(&m.b, format, args...)
		m.b.WriteRune('\n')
	}
}

// "Returns the trimmed string content of the buffer and the stored error, providing the final output and any encountered error."
func (m *strErrBuf) output() (string, error) {
	return strings.TrimSpace(m.b.String()), m.err
}

// Executes the given template into the buffer using the provided data, storing the first encountered error in `m.err` and skipping subsequent executions if an error already exists.
func (m *strErrBuf) executeTemplate(t *template.Template, data any) {
	if m.err == nil {
		m.err = t.Execute(&m.b, data)
	}
}

// Parses the given template string into a named template and executes it with the provided data, appending the resulting output to the error buffer.
func (m *strErrBuf) execTemplS(name string, templ string, data any) {
	t := template.Must(template.New(name).Parse(templ))
	m.executeTemplate(t, data)
}
