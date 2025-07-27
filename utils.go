package main

import (
	"bytes"
	"regexp"
	"text/template"
)

func matchSubexp(re *regexp.Regexp, s string) (bool, map[string]string) {
	match := re.MatchString(s)
	if !match {
		return false, nil
	}
	submatch := re.FindStringSubmatch(s)
	subexp := re.SubexpNames()
	result := make(map[string]string)
	for i, name := range subexp {
		if i != 0 && name != "" {
			result[name] = submatch[i]
		}
	}
	return true, result
}

func executeTmpl(tmpl *template.Template, data any) (string, error) {
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
