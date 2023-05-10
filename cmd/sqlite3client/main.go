package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/jedib0t/go-pretty/v6/table"
)

type SqlResult struct {
	Columns []string
	Data    [][]string
	Error   string
}

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s addr port\n", os.Args[0])
		os.Exit(1)
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", os.Args[1], os.Args[2]))
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("Welcome to the Sqlite3Client! Enter your SQL queries below. Type \"exit\" or \"q\" to quit.")

	// Read a query from the user
	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 "> ",
		HistoryFile:            "/tmp/readline-multiline",
		DisableAutoSaveHistory: true,
	})

	if err != nil {
		panic(err)
	}
	defer rl.Close()

	for {
		query := readQueryFromPrompt(rl)
		if query == "exit" || query == "q" {
			break
		}

		// Send the query to the server
		err = sendQuery(conn, query)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// Read the result from the server
		result, err := readResult(conn)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// Print the error
		if result.Error != "" {
			fmt.Println(result.Error)
			continue
		}

		// Print the result
		fmt.Println(formatTable(result.Columns, result.Data))
	}
}

func readQueryFromPrompt(rl *readline.Instance) string {
	var query string
	var cmds []string
	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// Handle exit commands
		if line == "exit" || line == "q" {
			return line
		}

		cmds = append(cmds, line)
		if !strings.HasSuffix(line, ";") {
			rl.SetPrompt("... ")
			continue
		}

		query = strings.Join(cmds, " ")
		//lint:ignore SA4006 Command is actually used.
		cmds = cmds[:0]

		rl.SetPrompt("> ")
		rl.SaveHistory(query)
		break
	}
	return query
}

func sendQuery(conn net.Conn, query string) error {
	_, err := conn.Write([]byte(query))
	return err
}

func readResult(conn net.Conn) (*SqlResult, error) {
	decoder := gob.NewDecoder(conn)
	var result SqlResult
	err := decoder.Decode(&result)

	if err != nil {
		return nil, err
	}
	// Return the result
	return &result, nil

}

func formatTable(columnNames []string, rows [][]string) string {
	// Create a new table object
	tbl := table.NewWriter()
	tbl.SetStyle(table.StyleLight)

	// Set the column headers
	header := make(table.Row, len(columnNames))
	for i, name := range columnNames {
		header[i] = name
	}
	tbl.AppendHeader(header)

	// Add the rows of data
	for _, row := range rows {
		tblRow := table.Row{}
		for _, v := range row {
			tblRow = append(tblRow, v)
		}
		tbl.AppendRow(tblRow)
	}

	// Render the table to a string
	var buffer bytes.Buffer
	tbl.SetOutputMirror(&buffer)
	tbl.Render()
	return buffer.String()
}
