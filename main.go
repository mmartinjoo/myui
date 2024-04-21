package main

import (
	"database/sql"
	"github.com/gdamore/tcell/v2"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rivo/tview"
)

var (
	app          *tview.Application
	pages        *tview.Pages
	previewTable *tview.Table
	db           *sql.DB
	filter       *tview.Form
)

func main() {
	app = tview.NewApplication()

	previewTable = tview.NewTable().
		SetBorders(true)

	filter = tview.NewForm()

	login := createLogin()

	pages = tview.NewPages().
		AddPage("login", login, true, true).
		AddPage("table", previewTable, true, false).
		AddPage("filter", filter, true, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlF:
			pages.SwitchToPage("filter")
		}

		return event
	})

	if err := app.SetRoot(pages, true).SetFocus(login).Run(); err != nil {
		panic(err)
	}
}

type TableData struct {
	Columns []string
	Rows    []map[string]string
}

func createFilter(tableData TableData, tableName string) {
	filter.Clear(false)

	for i := 0; i < len(tableData.Columns); i++ {
		filter.
			AddInputField(tableData.Columns[i], "", 20, nil, nil)
	}

	filter.
		AddButton("Filter", func() {
			filters := make(map[string]string, len(tableData.Columns))

			for i := 0; i < len(tableData.Columns); i++ {
				formItem := filter.GetFormItem(i)

				val := formItem.(*tview.InputField).GetText()

				if val == "" {
					continue
				}

				filters[tableData.Columns[i]] = val
			}

			query := "select * from " + tableName

			i := 0

			for column, value := range filters {
				if i == 0 {
					query += " where "
				}

				if i > 0 && i < len(filters) {
					query += " and "
				}

				if value == "NULL" {
					query += column + " IS NULL "
				} else {
					query += column + " = '" + value + "' "
				}

				i++
			}

			query += " limit 100"

			newTableData := readTable(query, db)
			createTable(newTableData, tableName)

			pages.SwitchToPage("table")
		}).
		SetBorder(true).SetTitle("Filter").SetTitleAlign(tview.AlignLeft)
}

func createTable(tableData TableData, tableName string) {
	previewTable.Clear()

	for c := 0; c < len(tableData.Columns)-1; c++ {
		previewTable.SetCell(0, c,
			tview.NewTableCell(tableData.Columns[c]).SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter).SetSelectable(false))
	}

	for r := 0; r < len(tableData.Rows)-1; r++ {
		for c := 0; c < len(tableData.Columns)-1; c++ {
			color := tcell.ColorWhite

			previewTable.SetCell(r+1, c,
				tview.NewTableCell(tableData.Rows[r][tableData.Columns[c]]).
					SetTextColor(color).
					SetAlign(tview.AlignCenter))
		}
	}

	modal := tview.NewModal().
		SetText("What do you want to do with the row?").
		AddButtons([]string{"Update", "Delete", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Update" {
				panic("udpate")
			}

			if buttonLabel == "Delete" {
				row, _ := previewTable.GetSelection()

				// Because of the header row
				row--

				query := "delete from " + tableName + " where id = " + tableData.Rows[row]["id"]

				_, err := db.Query(query)

				if err != nil {
					panic(err.Error())
				}

				newTableData := readTable("select * from "+tableName+" limit 100", db)
				createTable(newTableData, tableName)

				pages.SwitchToPage("table")
			}

			if buttonLabel == "Cancel" {
				pages.SwitchToPage("table")
			}
		})

	previewTable.Select(0, 0).SetFixed(1, 1).SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			app.Stop()
		}
		if key == tcell.KeyEnter {
			previewTable.SetSelectable(true, false)
		}
	}).SetSelectedFunc(func(row int, column int) {
		pages.AddPage("action_modal", modal, true, true).
			SwitchToPage("action_modal")
	})
}

func readTable(query string, db *sql.DB) TableData {
	rows, err := db.Query(query)

	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	// Make a slice for the values
	values := make([]sql.RawBytes, len(columns))

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	var results []map[string]string

	// Fetch rows
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(scanArgs...)
		if err != nil {
			panic(err.Error()) // proper error handling instead of panic in your app
		}

		// Now do something with the data.
		// Here we just print each column as a string.
		var value string
		row := make(map[string]string)

		for i, col := range values {
			// Here we can check if the value is nil (NULL value)
			if col == nil {
				value = "NULL"
			} else {
				value = string(col)
			}

			row[columns[i]] = value
		}

		results = append(results, row)
	}

	if err = rows.Err(); err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	return TableData{Columns: columns, Rows: results}
}

func createLogin() *tview.Form {
	form := tview.NewForm()

	form.
		AddInputField("Username", "", 20, nil, nil).
		AddPasswordField("Password", "", 20, '*', nil).
		AddInputField("Database", "", 20, nil, nil).
		AddButton("Login", func() {
			var err error

			db, err = sql.Open("mysql", "root:root@/analytics")

			if err != nil {
				panic(err)
			}

			sites := readTable("select * from page_views limit 100", db)

			createTable(sites, "page_views")
			createFilter(sites, "page_views")

			pages.SwitchToPage("table")
		})

	return form
}
