package utils

import (
	"fmt"
	"strings"
)

const (
	DefaultTableWidth = 63
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

func (t *SimpleTable) Layout() {
	for i, c := range t.Columns {
		max := 0
		for _, r := range t.Records {
			if len(r.Fields[c.ID]) > max {
				max = len(r.Fields[c.ID])
			}
		}
		if len(c.Display) > max {
			max = len(c.Display)
		}
		t.Columns[i].Width = max
	}

	n := len(t.Columns)

	// extend last column to fit table width
	s := 0
	for i, c := range t.Columns {
		if i == n - 1 {
			s = s + 4 // left border, padding, right border
			if DefaultTableWidth - s > c.Width {
				t.Columns[i].Width = DefaultTableWidth - s
			}
		}
		s += c.Width + 3 // 2 padding, 1 left border
	}
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
	t.Layout()

	n := len(t.Columns)

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
			fmt.Print("┐\n")
		}
	}

	// print table head
	// │ SERVICE │ STATUS                                              │
	for i, c := range t.Columns {
		fmt.Print("│ ")
		fmt.Print(c.Display)
		fmt.Print(strings.Repeat(" ", c.Width - len(c.Display) + 1))
		if i == n - 1 {
			fmt.Print("│\n")
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
				fmt.Print("┤\n")
			}
		}
		for i, c := range t.Columns {
			fmt.Print("│ ")
			fmt.Print(r.Fields[c.ID])
			fmt.Print(strings.Repeat(" ", c.Width - len(r.Fields[c.ID]) + 1))
			if i == n - 1 {
				fmt.Print("│\n")
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
			fmt.Print("┘\n")
		}
	}
}

func (t *SimpleTable) PrintUpdate(x TableRecord) {
	for i, r := range t.Records {
		if r.Fields["service"] == x.Fields["service"] {
			t.Records[i].Fields["status"] = x.Fields["status"]
		}
	}
	fmt.Printf("\033[%dA", 2 * len(t.Records) + 3)
	t.Print()
}
