package main

import (
	"database/sql"
	"github.com/gdamore/tcell/v2"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rivo/tview"
)

var (
	app   *tview.Application
	pages *tview.Pages
	db    *sql.DB
)

func main() {
	app = tview.NewApplication()

	var err error

	db, err = sql.Open("mysql", "root:root@/analytics")

	if err != nil {
		panic(err)
	}

	sites := readTable("select * from page_views limit 100", db)

	table := createTable(sites, "page_views")
	filter := createFilter(sites, "page_views")

	pages = tview.NewPages().
		AddPage("table", table, true, true).
		AddPage("filter", filter, true, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlF:
			pages.SwitchToPage("filter")
		}

		return event
	})

	//
	//flex := tview.NewFlex().
	//	AddItem(table, 0, 1, false).
	//	AddItem(filter, 0, 1, false)

	//table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
	//	if event.Key() == tcell.KeyCtrlF {
	//		setFocus(filter, app)
	//	}
	//	return event
	//})

	if err := app.SetRoot(pages, true).SetFocus(table).Run(); err != nil {
		panic(err)
	}

	//renderTable(sites, app)
	//renderFilter(sites, app)

	//fmt.Println(sites.Columns)
	//fmt.Println(sites.Rows)
}

type TableData struct {
	Columns []string
	Rows    []map[string]string
}

func setFocus(p tview.Primitive, app *tview.Application) {
	go func() {
		app.QueueUpdateDraw(func() {
			app.SetFocus(p)
		})
	}()
}

func createFilter(tableData TableData, tableName string) *tview.Form {
	form := tview.NewForm()

	for i := 0; i < len(tableData.Columns); i++ {
		form.
			AddInputField(tableData.Columns[i], "", 20, nil, nil)
	}

	form.
		AddButton("Filter", func() {
			filters := make(map[string]string, len(tableData.Columns))

			for i := 0; i < len(tableData.Columns); i++ {
				formItem := form.GetFormItem(i)

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

			tableData := readTable(query, db)
			table := createTable(tableData, tableName)

			pages.AddPage("results", table, true, true).
				SwitchToPage("results")
		}).
		SetBorder(true).SetTitle("Filter").SetTitleAlign(tview.AlignLeft)

	return form

	//if err := app.SetRoot(form, true).EnableMouse(true).Run(); err != nil {
	//	panic(err)
	//}

	//fmt.Println(form.GetFormItem(0).(*tview.InputField).GetText())
}

func createTable(tableData TableData, tableName string) *tview.Table {
	table := tview.NewTable().
		SetBorders(true)

	for c := 0; c < len(tableData.Columns)-1; c++ {
		table.SetCell(0, c,
			tview.NewTableCell(tableData.Columns[c]).SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter))
	}

	for r := 0; r < len(tableData.Rows)-1; r++ {
		for c := 0; c < len(tableData.Columns)-1; c++ {
			color := tcell.ColorWhite

			table.SetCell(r+1, c,
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
				row, _ := table.GetSelection()

				// Because of the header row
				row--

				query := "delete from " + tableName + " where id = " + tableData.Rows[row]["id"]

				_, err := db.Query(query)

				if err != nil {
					panic(err.Error())
				}

				newTableData := readTable("select * from "+tableName+" limit 100", db)
				newTable := createTable(newTableData, tableName)

				pages.AddPage("table", newTable, true, true).
					SwitchToPage("table")
			}

			if buttonLabel == "Cancel" {
				pages.SwitchToPage("table")
			}
		})

	table.Select(0, 0).SetFixed(1, 1).SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			app.Stop()
		}
		if key == tcell.KeyEnter {
			table.SetSelectable(true, false)
		}
	}).SetSelectedFunc(func(row int, column int) {
		pages.AddPage("action_modal", modal, true, true).
			SwitchToPage("action_modal")
		//table.GetCell(row, column).SetTextColor(tcell.ColorRed)
		//table.SetSelectable(false, false)
	})

	return table
	//if err := app.SetRoot(table, true).SetFocus(table).Run(); err != nil {
	//	panic(err)
	//}
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
