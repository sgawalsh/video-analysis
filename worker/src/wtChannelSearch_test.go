package main

import (
    "reflect"
    "testing"
)


func TestEvenlySpacedIndices(t *testing.T) {
	tests := []struct {
		name     string
		length   int
		n        int
		expected []int
	}{
		{
			name:     "empty list",
			length:   0,
			n:        5,
			expected: nil,
		},
		{
			name:     "single element",
			length:   1,
			n:        5,
			expected: []int{0},
		},
		{
			name:     "single sample",
			length:   10,
			n:        1,
			expected: []int{4},
		},
		{
			name:     "samples equal length",
			length:   5,
			n:        5,
			expected: []int{0, 1, 2, 3, 4},
		},
		{
			name:     "more samples than elements",
			length:   5,
			n:        10,
			expected: []int{0, 1, 2, 3, 4},
		},
		{
			name:     "normal sampling",
			length:   20,
			n:        5,
			expected: []int{0, 4, 9, 14, 19},
		},
		{
			name:     "non-even division",
			length:   10,
			n:        4,
			expected: []int{0, 3, 6, 9},
		},
		{
			name:     "two samples",
			length:   10,
			n:        2,
			expected: []int{0, 9},
		},
		{
			name:     "zero samples",
			length:   10,
			n:        0,
			expected: nil,
		},
		{
			name:     "negative samples",
			length:   10,
			n:        -1,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evenlySpacedIndices(tt.length, tt.n)

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("evenlySpacedIndices(%d, %d) = %v, want %v",
					tt.length, tt.n, got, tt.expected)
			}
		})
	}
}

func TestEvenlySpacedIndicesProperties(t *testing.T) {
	for length := 1; length <= 1000; length++ {
		for n := 1; n <= length+5; n++ {
			indices := evenlySpacedIndices(length, n)

			// Correct number of samples.
			expectedLen := min(length, n)
			if len(indices) != expectedLen {
				t.Fatalf("length=%d n=%d: got %d indices, want %d",
					length, n, len(indices), expectedLen)
			}

			// All indices are valid and strictly increasing.
			for i, index := range indices {
				if index < 0 || index >= length {
					t.Fatalf("out-of-bounds index: %d", index)
				}

				if i > 0 && index <= indices[i-1] {
					t.Fatalf("indices are not strictly increasing: %v", indices)
				}
			}

			// When sampling multiple elements, include both endpoints.
			if n >= 2 && n <= length {
				if indices[0] != 0 || indices[len(indices)-1] != length-1 {
					t.Fatalf("endpoints missing: %v", indices)
				}
			}
		}
	}
}