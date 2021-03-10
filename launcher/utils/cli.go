package utils

import "fmt"

type SimpleTable struct {
	Columns []TableColumn
	Records []TableRecord
}

type TableColumn struct {
	ID string
	Display string
}

type TableRecord struct {
	Fields map[string]string
}

func (t *SimpleTable) Print() {
	fmt.Println("")
}

func (t *SimpleTable) PrintUpdate(r TableRecord) {
	fmt.Printf("%s", r.Fields)
}
