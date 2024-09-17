package parser

import (
	"reflect"
	"testing"
)

func TestFilterPreloads(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string][]string
		expected map[string][]string
	}{
		{
			name: "Simple case",
			input: map[string][]string{
				"User": {"Profile", "Profile.Addresses", "Profile.Cards"},
			},
			expected: map[string][]string{
				"User": {"Profile.Addresses", "Profile.Cards"},
			},
		},
		{
			name: "Nested case",
			input: map[string][]string{
				"User": {
					"Profile",
					"Profile.Addresses",
					"Profile.Addresses.City",
					"Profile.Cards",
					"Orders",
					"Orders.Items",
				},
			},
			expected: map[string][]string{
				"User": {
					"Profile.Addresses.City",
					"Profile.Cards",
					"Orders.Items",
				},
			},
		},
		{
			name: "Multiple models",
			input: map[string][]string{
				"User": {
					"Profile",
					"Profile.Addresses",
					"Orders",
				},
				"Order": {
					"Items",
					"Items.Product",
					"User",
				},
			},
			expected: map[string][]string{
				"User": {
					"Profile.Addresses",
					"Orders",
				},
				"Order": {
					"Items.Product",
					"User",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := filterPreloads(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("filterPreloads(%v) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}
