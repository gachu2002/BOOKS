package registry

import (
	"fmt"
	"slices"

	"leetcode/array_string"
)

// Question describes one problem and how to run its sample cases.
type Question struct {
	Slug     string
	Title    string
	Category string
	URL      string
	Run      func() error
}

var questions = map[string]Question{
	"merge-sorted-array": {
		Slug:     "merge-sorted-array",
		Title:    "Merge Sorted Array",
		Category: "array_string",
		URL:      "https://leetcode.com/problems/merge-sorted-array/",
		Run: func() error {
			nums1 := []int{1, 2, 3, 0, 0, 0}
			nums2 := []int{2, 5, 6}
			array_string.MergeSortedArray(nums1, 3, nums2, 3)
			want := []int{1, 2, 2, 3, 5, 6}
			if !slices.Equal(nums1, want) {
				return fmt.Errorf("sample failed: got %v want %v", nums1, want)
			}
			fmt.Printf("sample passed: nums1=%v\n", nums1)
			return nil
		},
	},
	"remove-element": {
		Slug:     "remove-element",
		Title:    "Remove Element",
		Category: "array_string",
		URL:      "https://leetcode.com/problems/remove-element/",
		Run: func() error {
			nums := []int{3, 2, 2, 3}
			k := array_string.RemoveElement(nums, 3)
			wantK := 2
			wantPrefix := []int{2, 2}
			if k != wantK {
				return fmt.Errorf("sample failed: got length %d want %d", k, wantK)
			}
			if !slices.Equal(nums[:k], wantPrefix) {
				return fmt.Errorf("sample failed: got prefix %v want %v", nums[:k], wantPrefix)
			}
			fmt.Printf("sample passed: k=%d nums=%v\n", k, nums[:k])
			return nil
		},
	},
}

func All() []Question {
	out := make([]Question, 0, len(questions))
	for _, q := range questions {
		out = append(out, q)
	}
	return out
}

func Get(slug string) (Question, bool) {
	q, ok := questions[slug]
	return q, ok
}
