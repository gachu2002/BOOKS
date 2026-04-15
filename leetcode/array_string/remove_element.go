package array_string

// Remove Element
// https://leetcode.com/problems/remove-element/
func RemoveElement(nums []int, val int) int {
	writeIndex := 0
	for _, num := range nums {
		if num == val {
			continue
		}
		nums[writeIndex] = num
		writeIndex++
	}

	return writeIndex
}
