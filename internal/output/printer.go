package output

import (
	"fmt"
	"strings"
)

func PrintTable(headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	sep := make([]string, len(headers))
	for i, w := range widths {
		sep[i] = strings.Repeat("-", w)
	}

	printRow(headers, widths)
	printRow(sep, widths)
	for _, row := range rows {
		printRow(row, widths)
	}
}

func printRow(cells []string, widths []int) {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		if i < len(widths) {
			parts[i] = fmt.Sprintf("%-*s", widths[i], cell)
		} else {
			parts[i] = cell
		}
	}
	fmt.Println(strings.Join(parts, "  "))
}

func PrintKV(pairs [][2]string) {
	maxKey := 0
	for _, p := range pairs {
		if len(p[0]) > maxKey {
			maxKey = len(p[0])
		}
	}
	for _, p := range pairs {
		fmt.Printf("%-*s  %s\n", maxKey, p[0], p[1])
	}
}
