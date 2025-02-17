// SPDX-FileCopyrightText: 2022 Risk.Ident GmbH <contact@riskident.com>
//
// SPDX-License-Identifier: GPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify it
// under the terms of the GNU General Public License as published by the
// Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for
// more details.
//
// You should have received a copy of the GNU General Public License along
// with this program.  If not, see <http://www.gnu.org/licenses/>.

package config

import (
	"bytes"
	"encoding"
	"text/template"

	"github.com/invopop/jsonschema"
	"github.com/spf13/pflag"
)

var FuncsMap template.FuncMap

type Template template.Template

// Ensure the type implements the interfaces
var _ pflag.Value = &Template{}
var _ encoding.TextUnmarshaler = &Template{}
var _ jsonSchemaInterface = Template{}

func (t *Template) Template() *template.Template {
	return (*template.Template)(t)
}

func (t *Template) String() string {
	return t.Template().Root.String()
}

func (t *Template) Set(value string) error {
	parsed, err := template.New("").Funcs(FuncsMap).Parse(value)
	if err != nil {
		return err
	}
	*t = Template(*parsed)
	return nil
}

func (Template) Type() string {
	return "template"
}

func (t *Template) UnmarshalText(text []byte) error {
	return t.Set(string(text))
}

func (t *Template) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (Template) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:  "string",
		Title: "Go template",
	}
}

func (t *Template) Render(data any) (string, error) {
	var buf bytes.Buffer
	if err := t.Template().Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
