package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/url"
	"strconv"
	"time"
)

type Paste struct {
	Id        int    `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt int64
	CreatorIP string
}

// сплошной кринж, не закрываем коннекшены, если хотя бы раз не получится acquire-нуть бд (а sqlite - синхронная), то сразу ложим поток
func getDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./db.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	return db
}

/*
в общем и целом плохо, заместо вот этого бд-кринжа сделать какой то генератор коннекшенов / транзакций из единого пула
логику с добавлением в пасты и ченджлогом разбить
конны, транзы нужно закрывать
и при ошибках отменять контекст чтобы он не потiк
*/
func (p Paste) saveToDB() int {
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second)) // отменять контекст никто не захотел,
	// привет context leak
	db := getDB()           // вот это ебучий кринж
	conn, _ := db.Conn(ctx) // да да это тоже

	/*
		вот все квери которые здесь есть это ну плохо
		без quoting'а, без санитайзинга ни че го здесь не делается
		привет sql-иньекциям передаем
		лучше бы squirell здесь был
	*/

	query := fmt.Sprintf("insert into pastes (title, body, created_at) values ('%s', '%s', %d)", p.Title, p.Body, time.Now().Unix())
	queryGetInsertedPaste := fmt.Sprintf("select id from pastes where title='%s'", p.Title) // obsolete, можно в первом query сделать returning id
	queryChngLog := "insert into changelog (paste_id, creator_ip, paste_body_len) values (%d, '%s', %d)"

	tx, err := conn.BeginTx(ctx, nil) // сначала берем коннекшен, от него транзакцию..
	// ну ладно, это было бы ок если бд была бы в синглтоне
	// а коннекшены с транзами просто ЗАКРЫВАЛИСЬ

	if err != nil {
		log.Fatal(err) // ех; да ну мы просто положим поток
	}

	tx.Exec(query)                                                   // ех
	insertedIdRes := tx.QueryRow(fmt.Sprintf(queryGetInsertedPaste)) // obsolete (см коммент выше)
	var insertedId int
	insertedIdRes.Scan(&insertedId)                                          // ех
	tx.Exec(fmt.Sprintf(queryChngLog, insertedId, p.CreatorIP, len(p.Body))) // ех;

	if err := tx.Commit(); err != nil {
		// ТАК ТЫ ХОТЯ БЫ РОЛЛБЕКНИ Я ХЗ И ЗАКРОЙ...
		return -1
	}

	return insertedId
}

func getPasteFromDb(params url.Values) (Paste, bool) {
	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	db := getDB()
	conn, _ := db.Conn(ctx)

	// здесь все то же самое

	var query string

	// --

	id, err := strconv.Atoi(params.Get("id"))
	if err != nil {
		query = fmt.Sprintf("select title, body from pastes where title like '%s'", params.Get("title"))
	} else {
		query = fmt.Sprintf("select title, body from pastes where id=%d", id)
	}

	// что? че я щас прочитал... как обычно sql-иньекции, ищем основываясь есть ли у нас айдишник...
	// ну хоть ех присутствует...
	// только применено оно не по назначению

	// --

	res := conn.QueryRowContext(ctx, query)
	var p Paste
	if err := res.Scan(&p.Title, &p.Body); err != nil {
		return p, false
	}
	return p, true // может быть будем тогда возвращать *Paste, error.... ай неважно, ну бд мы опять не закрыли
}
