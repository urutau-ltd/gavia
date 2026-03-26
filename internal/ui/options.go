package ui

import "strings"

type SelectOption struct {
	Value string
	Label string
}

type AssignmentOption struct {
	Value    string
	Label    string
	Hint     string
	Selected bool
}

// BuildSelectOptions converts typed repository records into template-friendly choices.
func BuildSelectOptions[T any](
	items []T,
	valueOf func(T) string,
	labelOf func(T) string,
) []SelectOption {
	options := make([]SelectOption, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(valueOf(item))
		label := strings.TrimSpace(labelOf(item))
		if value == "" || label == "" {
			continue
		}

		options = append(options, SelectOption{
			Value: value,
			Label: label,
		})
	}

	return options
}

// FindSelectValueByLabel resolves a value from a human-readable label, used by defaults.
func FindSelectValueByLabel[T any](
	items []T,
	wantLabel string,
	valueOf func(T) string,
	labelOf func(T) string,
) string {
	wantLabel = strings.TrimSpace(wantLabel)
	if wantLabel == "" {
		return ""
	}

	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(labelOf(item)), wantLabel) {
			return strings.TrimSpace(valueOf(item))
		}
	}

	return ""
}

// BuildAssignmentOptions converts records into checkbox-friendly options.
func BuildAssignmentOptions[T any](
	items []T,
	selected map[string]struct{},
	valueOf func(T) string,
	labelOf func(T) string,
	hintOf func(T) string,
) []AssignmentOption {
	options := make([]AssignmentOption, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(valueOf(item))
		label := strings.TrimSpace(labelOf(item))
		if value == "" || label == "" {
			continue
		}

		_, isSelected := selected[value]
		options = append(options, AssignmentOption{
			Value:    value,
			Label:    label,
			Hint:     strings.TrimSpace(hintOf(item)),
			Selected: isSelected,
		})
	}

	return options
}
