package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/external/", func(w http.ResponseWriter, r *http.Request) {
		externalId := strings.TrimPrefix("/external/", r.URL.Path) // нет валидации id, возможный инжект; atoi??
		paste := getExternalPastes(externalId)                     // cringe

		marshelled, _ := json.Marshal(&paste) // далее где будет проигнорен ерр хендлинг я буду просто оставлять ех
		w.Write(marshelled)                   // ех
		// статус код писать никто не захотел...
		fmt.Println("returned external transaction success") // это еще че
	})
	mux.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body) // ех
		var p Paste
		if err := json.Unmarshal(body, &p); err != nil {
			w.Write([]byte("400: bad request")) // по-моему статус код не так пишется... да и 422 отдается обычно
			return
		}
		p.CreatedAt = time.Now().Unix()
		p.CreatorIP = r.RemoteAddr

		savedId := p.saveToDB()                    // КРИИИИИИИИНЖ
		w.Write([]byte(fmt.Sprint(savedId)))       // иууу айди записали StatusCode зачем вообще нужен
		fmt.Printf("saved new paste: %d", savedId) // golang development 2025 logging w/o newline
	})
	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		p, ok := getPasteFromDb(r.URL.Query()) // так может быть квери хотя бы обработать здесь, на слой повыше...
		if !ok {
			fmt.Println("cannot get paste") // и продолжаем выполнение записывая потенциально ломаную структуру, так держать!
		}

		data, _ := json.Marshal(&p)
		w.Write(data) // ех
	})

	http.ListenAndServe("", mux) // ех; ну на порт 80 некрасиво все же
}
