package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"strings"

	"github.com/tealeg/xlsx"
)

func main() {
	p, err := Dir(os.Args[1])

	if err != nil {
		log.Fatalln(err)
	}

	buf := &bytes.Buffer{}

	if err := p.Write(buf); err != nil {
		log.Fatalln(err)
	}

	s := buf.String()
	fmt.Println("out:", s)
}

// Parser for transformation
type Parser struct {
	inputs []map[string]string // Data input
	tmpl   *template.Template  // SQL transformation template
}

// New create a new instance of the SQL transformation parser
func New(inputs []map[string]string, sql string) (*Parser, error) {
	tmpl, err := template.New("sql").Parse(sql)

	if err != nil {
		return nil, err
	}

	if len(inputs) == 0 {
		return nil, fmt.Errorf("no input found")
	}

	return &Parser{inputs, tmpl}, nil
}

// Files maps the inputs files to the parser
// .sql files are map as templates
// .csv, .xlsx is transformed to a dictionary
func Files(files []string) (*Parser, error) {
	var inputs []map[string]string
	var tmpl string
	var err error

	for _, file := range files {
		ext := path.Ext(file)

		switch ext {
		case ".csv":
			if inputs, err = readCSV(file); err != nil {
				return nil, err
			}

		case ".sql":
			if tmpl, err = readSQL(file); err != nil {
				return nil, err
			}
		case ".xlsx":
			if inputs, err = readXLSX(file); err != nil {
				return nil, err
			}
		}
	}

	return New(inputs, tmpl)
}

// Dir maps a directory to the parser
func Dir(dir string) (*Parser, error) {
	fi, err := ioutil.ReadDir(dir)

	if err != nil {
		return nil, err
	}

	files := []string{}

	for _, file := range fi {
		files = append(files, path.Join(dir, file.Name()))
	}

	return Files(files)
}

func (p *Parser) Write(w io.Writer) error {
	for _, data := range p.inputs {
		if err := p.tmpl.Execute(w, data); err != nil {
			return err
		}
		if _, err := w.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	return nil
}

func readSQL(file string) (string, error) {
	arr, err := ioutil.ReadFile(file)
	return string(arr), err
}
func readCSV(file string) ([]map[string]string, error) {
	var header []string
	var inputs []map[string]string

	f, err := os.Open(file)

	if err != nil {
		return inputs, err
	}

	r := csv.NewReader(f)

	for {
		record, err := r.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			return inputs, err
		}

		if header == nil {
			header = record
		} else {
			entry := map[string]string{}

			for i, val := range record {
				entry[header[i]] = val
			}

			inputs = append(inputs, entry)
		}
	}

	return inputs, nil
}
func readXLSX(file string) ([]map[string]string, error) {
	var inputs []map[string]string
	var headers []string

	xlFile, err := xlsx.OpenFile(file)
	if err != nil {
		return inputs, err
	}

	for ri, row := range xlFile.Sheets[0].Rows {
		// for ci, cell := range row.Cells {

		if ri == 0 {
			for _, cell := range row.Cells {
				headers = append(headers, cell.Value)
			}
		} else {
			entry := map[string]string{}
			hasVal := false

			for i, key := range headers {
				if i >= len(row.Cells) {
					entry[key] = ""
				} else {
					val := row.Cells[i].Value
					entry[key] = strings.TrimSpace(val)
					hasVal = len(val) > 0
				}
			}
			if hasVal {
				inputs = append(inputs, entry)
			}
		}

		// fmt.Printf("%v.%v - %#v\n", ri, ci, cell)
		// }
	}

	return inputs, nil
}
