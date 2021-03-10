package utils

import (
	"fmt"
	"strings"
)

const (
	DefaultTableWidth = 68
)

type SimpleTable struct {
	Columns []TableColumn
	Records []TableRecord
}

type TableColumn struct {
	ID string
	Display string
	Width int
}

type TableRecord struct {
	Fields map[string]string
}

// Print the table like
// ┌─────────┬─────────────────────────────────────────────────────┐
// │ SERVICE │ STATUS                                              │
// ├─────────┼─────────────────────────────────────────────────────┤
// │ lndbtc  │ Syncing 34.24% (610000/1781443)                     │
// ├─────────┼─────────────────────────────────────────────────────┤
// │ lndltc  │ Syncing 12.17% (191000/1568645)                     │
// └─────────┴─────────────────────────────────────────────────────┘
func (t *SimpleTable) Print() {
	for _, c := range t.Columns {
		max := 0
		for _, r := range t.Records {
			if len(r.Fields[c.ID]) > max {
				max = len(r.Fields[c.ID])
			}
		}
		if len(c.Display) > max {
			max = len(c.Display)
		}
		c.Width = max
	}

	n := len(t.Columns)

	// extend last column to fit table width
	s := 0
	for i, c := range t.Columns {
		if i == n - 1 {
			s = s + 1 // right border
			if DefaultTableWidth - s > c.Width {
				c.Width = DefaultTableWidth - s
			}
		}
		s += c.Width + 3 // 2 padding, 1 left border
	}

	// print tale top border
	// ┌─────────┬─────────────────────────────────────────────────────┐
	for i, c := range t.Columns {
		if i == 0 {
			fmt.Print("┌")
		} else {
			fmt.Print("┬")
		}
		fmt.Print(strings.Repeat("─", c.Width + 2))
		if i == n - 1 {
			fmt.Print("┐")
		}
	}

	// print table head
	// │ SERVICE │ STATUS                                              │
	for i, c := range t.Columns {
		fmt.Print("│ ")
		fmt.Print(c.Display)
		if i == n - 1 {
			fmt.Print("│")
		}
	}

	// print table records
	// ├─────────┼─────────────────────────────────────────────────────┤
	// │ lndbtc  │ Syncing 34.24% (610000/1781443)                     │
	for _, r := range t.Records {
		for i, c := range t.Columns {
			if i == 0 {
				fmt.Print("├")
			} else {
				fmt.Print("┼")
			}
			fmt.Print(strings.Repeat("─", c.Width + 2))
			if i == n - 1 {
				fmt.Print("┤")
			}
		}
		for i, c := range t.Columns {
			fmt.Print("│ ")
			fmt.Print(r.Fields[c.ID])
			if i == n - 1 {
				fmt.Print("│")
			}
		}
	}

	// print table bottom border
	// └─────────┴─────────────────────────────────────────────────────┘
	for i, c := range t.Columns {
		if i == 0 {
			fmt.Print("└")
		} else {
			fmt.Print("┴")
		}
		fmt.Print(strings.Repeat("─", c.Width + 2))
		if i == n - 1 {
			fmt.Print("┘")
		}
	}
}

func (t *SimpleTable) PrintUpdate(r TableRecord) {
	fmt.Printf("%s\n", r.Fields)
}
