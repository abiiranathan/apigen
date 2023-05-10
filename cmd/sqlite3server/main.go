package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SqlResult struct {
	Columns []string
	Data    [][]string
	Error   string
}

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s db port\n", os.Args[0])
		os.Exit(1)
	}

	// Open the SQLite3 database file
	db, err := sql.Open("sqlite3", os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// Create the TCP listener
	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", os.Args[2]))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating listener: %v\n", err)
		os.Exit(1)
	}
	defer ln.Close()

	fmt.Println("Listening on", ln.Addr())

	// Handle incoming connections
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error accepting connection: %v\n", err)
			continue
		}
		// Handle each connection in a separate goroutine
		go handleConnection(conn, db)
	}
}

func handleConnection(conn net.Conn, db *sql.DB) {
	fmt.Println("Connection from", conn.RemoteAddr())
	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		var buf bytes.Buffer
		for {
			chunk, err := reader.ReadString(';')
			if err != nil && err != io.EOF {
				fmt.Fprintf(os.Stderr, "error reading query: %v\n", err)
				break
			}

			if err == io.EOF {
				return
			}

			buf.WriteString(chunk)
			if strings.Contains(chunk, ";") {
				break
			}
		}

		query := strings.TrimSpace(buf.String())
		buf.Reset()

		result, err := execQuery(db, conn, query)
		if err != nil {
			result = &SqlResult{
				Error: err.Error(),
			}
		}
		sendQueryResult(conn, result)
	}
}

func execQuery(db *sql.DB, conn net.Conn, query string) (*SqlResult, error) {
	// Execute the query and send the results back to the client
	ctx := context.Background()
	ctx, cancelFunc := context.WithDeadline(ctx, time.Now().Add(time.Second*5))
	defer cancelFunc()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error(): %q error=%v\n", query, err)
		return nil, err
	}
	defer rows.Close()

	// error ignored since rows are not closed
	columns, _ := rows.Columns()

	columnCount := len(columns)
	values := make([]interface{}, columnCount)
	valuePtrs := make([]interface{}, columnCount)
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Send rows to the client
	data := [][]string{}
	for rows.Next() {
		err = rows.Scan(valuePtrs...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error scanning row: %v\n", err)
			return nil, err
		}

		rowData := []string{}
		for _, value := range values {
			if value == nil {
				rowData = append(rowData, "NULL")
			} else {
				rowData = append(rowData, fmt.Sprintf("%v", value))
			}
		}
		data = append(data, rowData)
	}
	return &SqlResult{Columns: columns, Data: data}, nil
}

// Serialize and send the result as raw bytes
func sendQueryResult(conn net.Conn, data *SqlResult) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	encoder.Encode(data)
	_, err := io.Copy(conn, &buffer)
	if err != nil {
		fmt.Printf("io.Copy() error: %v\n", err)
		return
	}
}
