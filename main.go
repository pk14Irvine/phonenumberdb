package main

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "password"
	dbname   = "phone"
)

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=disable", host, port, user, password)
	db, err := sql.Open("postgres", psqlInfo)
	must(err)
	err = resetDB(db, dbname)
	must(err)
	db.Close()

	psqlInfo = fmt.Sprintf("%s dbname=%s", psqlInfo, dbname)
	db, err = sql.Open("postgres", psqlInfo)
	must(err)
	defer db.Close()

	must(db.Ping())
	must(createPhoneNumbersTable(db))

	_, err = insertPhone(db, "123 456 7891")
	must(err)
	_, err = insertPhone(db, "(123) 456 7892")
	must(err)
	_, err = insertPhone(db, "(123) 456-7893")
	must(err)
	_, err = insertPhone(db, "123-456-7894")
	must(err)
	_, err = insertPhone(db, "123-456-7890")
	must(err)
	_, err = insertPhone(db, "1234567892")
	must(err)
	_, err = insertPhone(db, "(123)456-7892")
	must(err)
	id, err := insertPhone(db, "1234567890")
	must(err)
	number, err := getPhone(db, id)
	must(err)
	fmt.Println("Number is...", number)

	phones, err := allPhones(db)
	must(err)
	for _, p := range phones {
		fmt.Printf("Working on... %+v\n", p)
		number := normalize(p.number)
		if number != p.number {
			fmt.Println("Updating or removing...", number)
			existing, err := findPhone(db, number)
			must(err)
			if existing != nil {
				must(deletePhone(db, p.id))
			} else {
				p.number = number
				must(updatePhone(db, p))
			}
		} else {
			fmt.Println("No changes required")
		}
	}
}

type phone struct {
	id     int
	number string
}

func deletePhone(db *sql.DB, id int) error {
	statement := `delete from phone_numbers where id = $1`
	_, err := db.Exec(statement, id)
	return err
}
func allPhones(db *sql.DB) ([]phone, error) {
	rows, err := db.Query("select id, value from phone_numbers")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ret []phone

	for rows.Next() {
		var p phone
		if err := rows.Scan(&p.id, &p.number); err != nil {
			return nil, err
		}
		ret = append(ret, p)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return ret, nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func getPhone(db *sql.DB, id int) (string, error) {
	var number string
	row := db.QueryRow("select * from phone_numbers where id=$1", id)
	err := row.Scan(&id, &number)
	if err != nil {
		return "", err
	}
	return number, nil
}

func findPhone(db *sql.DB, number string) (*phone, error) {
	var p phone
	row := db.QueryRow("select * from phone_numbers where value=$1", number)
	err := row.Scan(&p.id, &p.number)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return &p, nil
}

func updatePhone(db *sql.DB, p phone) error {
	statement := `update phone_numbers set value = $2 where id = $1`
	_, err := db.Exec(statement, p.id, p.number)
	return err
}

func insertPhone(db *sql.DB, phone string) (int, error) {
	statement := `insert into phone_numbers(value) values($1) returning id`
	var id int
	err := db.QueryRow(statement, phone).Scan((&id))
	if err != nil {
		return -1, err
	}
	return id, nil
}

func createPhoneNumbersTable(db *sql.DB) error {
	statement := `
		create table if not exists phone_numbers(
			id serial,
			value varchar(255)
		)`
	_, err := db.Exec(statement)
	return err
}

func resetDB(db *sql.DB, name string) error {
	_, err := db.Exec("DROP DATABASE IF EXISTS " + name)
	if err != nil {
		return err
	}
	return createDB(db, name)
}

func createDB(db *sql.DB, name string) error {
	_, err := db.Exec("CREATE DATABASE " + name)
	if err != nil {
		return err
	}

	return nil
}

func normalize(phone string) string {
	// We want these -012346789
	// sql.Register("postgres", pq.)
	re := regexp.MustCompile("\\D")
	return re.ReplaceAllString(phone, "")
}

// func normalize(phone string) string {
// 	// We want these -012346789
// 	var buf bytes.Buffer
// 	for _, ch := range phone {
// 		if ch >= '0' && ch <= '9' {
// 			buf.WriteRune(ch)
// 		}
// 	}
// 	return buf.String()
// }
