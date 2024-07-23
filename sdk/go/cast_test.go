package sdk_test

import (
	"testing"

	sdk "github.com/RafaySystems/function-templates/sdk/go"
	"github.com/google/go-cmp/cmp"
)

var data = map[string]interface{}{
	"str1":  "value1",
	"int1":  2,
	"bool1": true,
	"slice1": []interface{}{
		"value2",
		3,
	},
	"float1": 3.14,
	"map1": map[string]interface{}{
		"str2":  "value4",
		"int2":  5,
		"bool2": false,
		"slice2": []interface{}{
			"value5",
			6,
		},
		"float2": 6.28,
		"map2": map[string]interface{}{
			"str3": "value7",
			"int3": 8,
		},
	},
}

func TestGetString(t *testing.T) {
	testcases := []struct {
		name        string
		input       []string
		expected    string
		expectedErr string
	}{
		{
			name:        "single key string",
			input:       []string{"str1"},
			expected:    "value1",
			expectedErr: "",
		},
		{
			name:        "nested key string",
			input:       []string{"map1", "str2"},
			expected:    "value4",
			expectedErr: "",
		},
		{
			name:        "nested nested key string",
			input:       []string{"map1", "map2", "str3"},
			expected:    "value7",
			expectedErr: "",
		},
		{
			name:        "empty keys",
			input:       []string{},
			expectedErr: "no keys provided",
		},
		{
			name:        "invalid key",
			input:       []string{"invalid"},
			expectedErr: "key invalid not found",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			o := sdk.Object(data)
			got, err := o.GetString(tc.input...)
			if err != nil {
				if err.Error() != tc.expectedErr {
					t.Errorf("expected error %s, got %s", tc.expectedErr, err.Error())
				}
			} else if tc.expectedErr == "" {
				if diff := cmp.Diff(got, tc.expected); diff != "" {
					t.Errorf("unexpected diff: %s", diff)
				}
			} else {
				t.Errorf("expected error %s, got nil", tc.expectedErr)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	testcases := []struct {
		name        string
		input       []string
		expected    int
		expectedErr string
	}{
		{
			name:        "single key int",
			input:       []string{"int1"},
			expected:    2,
			expectedErr: "",
		},
		{
			name:        "nested key int",
			input:       []string{"map1", "int2"},
			expected:    5,
			expectedErr: "",
		},
		{
			name:        "nested nested key int",
			input:       []string{"map1", "map2", "int3"},
			expected:    8,
			expectedErr: "",
		},
		{
			name:        "invalid key",
			input:       []string{"invalid"},
			expectedErr: "key invalid not found",
		},
		{
			name:        "invalid nested key",
			input:       []string{"map1", "invalid"},
			expectedErr: "key invalid not found",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			o := sdk.Object(data)
			got, err := o.GetInt(tc.input...)
			if err != nil {
				if err.Error() != tc.expectedErr {
					t.Errorf("expected error %s, got %s", tc.expectedErr, err.Error())
				}
			} else {
				if diff := cmp.Diff(got, tc.expected); diff != "" {
					t.Errorf("unexpected diff: %s", diff)
				}
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	testcases := []struct {
		name        string
		input       []string
		expected    bool
		expectedErr string
	}{
		{
			name:        "single key bool",
			input:       []string{"bool1"},
			expected:    true,
			expectedErr: "",
		},
		{
			name:        "nested key bool",
			input:       []string{"map1", "bool2"},
			expected:    false,
			expectedErr: "",
		},
		{
			name:        "invalid key",
			input:       []string{"invalid"},
			expectedErr: "key invalid not found",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			o := sdk.Object(data)
			got, err := o.GetBool(tc.input...)
			if err != nil {
				if err.Error() != tc.expectedErr {
					t.Errorf("expected error %s, got %s", tc.expectedErr, err.Error())
				}
			} else {
				if diff := cmp.Diff(got, tc.expected); diff != "" {
					t.Errorf("unexpected diff: %s", diff)
				}
			}
		})
	}
}

func TestGetFloat(t *testing.T) {
	testcases := []struct {
		name        string
		input       []string
		expected    float64
		expectedErr string
	}{
		{
			name:        "single key float",
			input:       []string{"float1"},
			expected:    3.14,
			expectedErr: "",
		},
		{
			name:        "nested key float",
			input:       []string{"map1", "float2"},
			expected:    6.28,
			expectedErr: "",
		},
		{
			name:        "invalid key",
			input:       []string{"invalid"},
			expectedErr: "key invalid not found",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			o := sdk.Object(data)
			got, err := o.GetFloat64(tc.input...)
			if err != nil {
				if err.Error() != tc.expectedErr {
					t.Errorf("expected error %s, got %s", tc.expectedErr, err.Error())
				}
			} else {
				if diff := cmp.Diff(got, tc.expected); diff != "" {
					t.Errorf("unexpected diff: %s", diff)
				}
			}
		})
	}
}

func TestGetStringMap(t *testing.T) {
	testcases := []struct {
		name        string
		input       []string
		expected    sdk.Object
		expectedErr string
	}{
		{
			name:        "single key map",
			input:       []string{"map1"},
			expected:    map[string]interface{}{"str2": "value4", "int2": 5, "bool2": false, "slice2": []interface{}{"value5", 6}, "float2": 6.28, "map2": map[string]interface{}{"str3": "value7", "int3": 8}},
			expectedErr: "",
		},
		{
			name:        "nested key map",
			input:       []string{"map1", "map2"},
			expected:    map[string]interface{}{"str3": "value7", "int3": 8},
			expectedErr: "",
		},
		{
			name:        "invalid key",
			input:       []string{"invalid"},
			expectedErr: "key invalid not found",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			o := sdk.Object(data)
			got, err := o.GetStringMap(tc.input...)
			if err != nil {
				if err.Error() != tc.expectedErr {
					t.Errorf("expected error %s, got %s", tc.expectedErr, err.Error())
				}
			} else {
				if diff := cmp.Diff(got, tc.expected); diff != "" {
					t.Errorf("unexpected diff: %s", diff)
				}
			}
		})
	}
}

func TestGetSlice(t *testing.T) {
	testcases := []struct {
		name        string
		input       []string
		expected    []interface{}
		expectedErr string
	}{
		{
			name:        "single key slice",
			input:       []string{"slice1"},
			expected:    []interface{}{"value2", 3},
			expectedErr: "",
		},
		{
			name:        "nested key slice",
			input:       []string{"map1", "slice2"},
			expected:    []interface{}{"value5", 6},
			expectedErr: "",
		},
		{
			name:        "invalid nested key",
			input:       []string{"map1", "invalid"},
			expectedErr: "key invalid not found",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			o := sdk.Object(data)
			got, err := o.GetSlice(tc.input...)
			if err != nil {
				if err.Error() != tc.expectedErr {
					t.Errorf("expected error %s, got %s", tc.expectedErr, err.Error())
				}
			} else {
				if diff := cmp.Diff(got, tc.expected); diff != "" {
					t.Errorf("unexpected diff: %s", diff)
				}
			}
		})
	}
}
