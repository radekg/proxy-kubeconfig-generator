package configuration

import "strings"

type NamespaceSelectorLabels struct {
	Values []string
}

func (i *NamespaceSelectorLabels) String() string {
	if i.Values == nil {
		return "<not set>"
	}
	return strings.Join(i.Values, ", ")
}

func (i *NamespaceSelectorLabels) Set(value string) error {
	i.Values = append(i.Values, value)
	return nil
}
