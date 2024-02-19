package provider

import (
	"fmt"
	"math"
)

const sepHeader = "Continued from previous comment.\n<details><summary>Show Output</summary>\n\n" +
	"```diff\n"

const sepFooter = "\n```\n</details>" +
	"\n<br>\n\n**Warning**: Output length greater than max comment size. Continued in next comment."

func sepHeaderId(headerTagID string) string {
	return fmt.Sprintf("%s\n", headerTagID) + sepHeader
}

// SplitComment splits comment into a slice of comments that are under maxSize.
// It appends sepEnd to all comments that have a following comment.
// It prepends sepStart to all comments that have a preceding comment.
func SplitComment(comment string, maxSize int, sepEnd string, sepStart string) []string {
	if len(comment) <= maxSize {
		return []string{comment}
	}

	maxWithSep := maxSize - len(sepEnd) - len(sepStart)
	var comments []string
	numComments := int(math.Ceil(float64(len(comment)) / float64(maxWithSep)))
	for i := 0; i < numComments; i++ {
		upTo := min(len(comment), (i+1)*maxWithSep)
		portion := comment[i*maxWithSep : upTo]
		if i < numComments-1 {
			portion += sepEnd
		}
		if i > 0 {
			portion = sepStart + portion
		}
		comments = append(comments, portion)
	}
	return comments
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
