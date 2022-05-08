//Этот файл сгенерирован автоматически, не изменяйте его

package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var ( // для Author:
	Authors          []Author                                                       //основная структура данных для модели Authors
	currentIdAuthors uint32   = 1                                                   //начальный идентификатор для модели Authors
	publicKeyAuthors          = []string{"name", "birthyear", "deathyear", "books"} //массив всех публичных полей структуры
)

var ( // для Book:
	Books          []Book                                                           //основная структура данных для модели Books
	currentIdBooks uint32 = 1                                                       //начальный идентификатор для модели Books
	publicKeyBooks        = []string{"name", "publicyear", "annotation", "authors"} //массив всех публичных полей структуры
)

func InitRouter() http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/authors", CreateAuthors).Methods("POST")
	router.HandleFunc("/authors", GetAuthors).Methods("GET")
	router.HandleFunc("/authors/{Id}", GetAuthor).Methods("GET")
	router.HandleFunc("/authors/{Id}", UpdateAuthor).Methods("PUT")
	router.HandleFunc("/authors/{Id}", DeleteAuthor).Methods("DELETE")
	router.HandleFunc("/authors/find/", FindAuthor).Methods("GET")

	router.HandleFunc("/books", CreateBooks).Methods("POST")
	router.HandleFunc("/books", GetBooks).Methods("GET")
	router.HandleFunc("/books/{Id}", GetBook).Methods("GET")
	router.HandleFunc("/books/{Id}", UpdateBook).Methods("PUT")
	router.HandleFunc("/books/{Id}", DeleteBook).Methods("DELETE")
	router.HandleFunc("/books/find/", FindBook).Methods("GET")
	return router
}

//CreateAuthors создает новые записи в коллекции из JSON тела POST запроса.
//JSON должен быть массивом объектов [{}...]
func CreateAuthors(w http.ResponseWriter, r *http.Request) {
	//прочтем всё тело запроса
	rBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("не прочли тело запроса: %v", rBody)
		return
	}

	//срез для приема данных
	var newAuthor []Author

	//парсим в структуру тело запроса
	err = json.Unmarshal(rBody, &newAuthor)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("не расшифровали тело запроса: %s,\n Ошибка: %s", rBody, err)
		return
	}

	//дополняем коллекцию метаданными
	for i := range newAuthor {
		newAuthor[i].Id = currentIdAuthors //присвоим Id по порядку от глобального счетчика
		currentIdAuthors++
		newAuthor[i].modification = r.RemoteAddr + ";" + r.UserAgent() +
			";" + time.Now().Format("2006/01/02 15:05:04")
	}

	//добавляем новые данные в общий срез
	Authors = append(Authors, newAuthor...)
	w.WriteHeader(http.StatusCreated)
	printAll(w, newAuthor)
}

//GetAuthors отдает полный список Author коллекции в JSON
func GetAuthors(w http.ResponseWriter, r *http.Request) {
	printAll(w, Authors)
}

//GetAuthor выдает информацию в JSON о Author по её Id
func GetAuthor(w http.ResponseWriter, r *http.Request) {

	tmp, err := strconv.Atoi(mux.Vars(r)["Id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("getAuthor: Id не прочли, ошибка: %s", err)
		return
	}
	id := uint32(tmp)
	for _, b := range Authors {
		if b.Id == id { //до первого совпадения Id
			printAll(w, []Author{b})
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

//UpdateAuthor обновляет информацию о Author в коллекции по её Id
func UpdateAuthor(w http.ResponseWriter, r *http.Request) {

	//читаем тело запроса в карту - не знаем, сколько и каких полей может встретиться

	var rBody map[string]string

	//тело запроса
	dec := json.NewDecoder(r.Body)
	for dec.More() {
		err := dec.Decode(&rBody)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("не прочли тело запроса, ошибка: %s", err)
			return
		}
	}

	//выясняем id обновляемой карточки
	tmp, err := strconv.Atoi(mux.Vars(r)["Id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Id не прочли, ошибка: %s", err)
		return
	}
	id := uint32(tmp)

	for i := range Authors {
		if Authors[i].Id == id { //до первого совпадения Id

			//рефлексируем по структуре данных, по каждому найденному полю
			//проверяем карту обновленных данных и если нашли - обновляем поле в структуре

			val := reflect.ValueOf(&Authors[i]).Elem() //адресуемое значение
			for ii := 0; ii < val.NumField(); ii++ {   //идем по полям структуры
				if tmp, ok := rBody[val.Type().Field(ii).Tag.Get("json")]; ok {
					//в карте нашли поле, его обновляем
					val.Field(ii).Set(reflect.ValueOf(tmp))
				}
			}
			Authors[i].modification = r.RemoteAddr + ";" + r.UserAgent() +
				";" + time.Now().Format("2006/01/02 15:05:04")

			printAll(w, []Author{Authors[i]}) //или просто w.WriteHeader(http.StatusOK)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

//DeleteAuthor удаляет Author в коллекции по её Id
func DeleteAuthor(w http.ResponseWriter, r *http.Request) {

	tmp, err := strconv.Atoi(mux.Vars(r)["Id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Id не прочли, ошибка: %s", err)
		return
	}
	id := uint32(tmp)
	for i, b := range Authors {
		if b.Id == id { //до первого совпадения Id
			Authors = append(Authors[:i], Authors[i+1:]...) //удаляем i элемент
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

//FindAuthor находит все Author по заданным текстовым параметрам - подстрокам.
//Неизвестные и некорректные ключи пропускаются
func FindAuthor(w http.ResponseWriter, r *http.Request) {

	var foundAuthor []Author //массив для найденных Author

	param := r.URL.Query() //карта карт всех принятых параметров

	for _, key := range publicKeyAuthors { // по всем возможным ключам
		if mapCurrentKey, exist := param[key]; exist { //если ключ в карте есть
			for _, value := range mapCurrentKey { //по всем значениям ключа (их может быть не один)
				value = strings.ToLower(value) //все будем сравнивать без учета регистра
				for i := range Authors {       //по всем Author коллекции
					val := reflect.ValueOf(&Authors[i]).Elem() //адресуемое значение
					for ii := 0; ii < val.NumField(); ii++ {   //идем по полям структуры
						if strings.EqualFold(val.Type().Field(ii).Tag.Get("json"), key) { //нашли поле в структуре, совпадающее по названию с ключом
							if strings.Contains(strings.ToLower(val.Field(ii).String()), value) { //в значении поля есть искомая подстрока
								if Authors[i].popular < math.MaxUint8 { //если был запрошен в поиске - повысим популярность
									Authors[1].popular++
								}
								foundAuthor = append(foundAuthor, Authors[i]) //копим все найденные Author
							}
						}
					}
				}
			}
		}
	}
	if len(foundAuthor) > 0 {
		printAll(w, foundAuthor)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	printAll(w, fmt.Sprintf("Ничего не нашли :(. Возможные ключи для поиска в Author: %s", publicKeyAuthors))
}

//CreateBooks создает новые записи в коллекции из JSON тела POST запроса.
//JSON должен быть массивом объектов [{}...]
func CreateBooks(w http.ResponseWriter, r *http.Request) {
	//прочтем всё тело запроса
	rBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("не прочли тело запроса: %v", rBody)
		return
	}

	//срез для приема данных
	var newBook []Book

	//парсим в структуру тело запроса
	err = json.Unmarshal(rBody, &newBook)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("не расшифровали тело запроса: %s,\n Ошибка: %s", rBody, err)
		return
	}

	//дополняем коллекцию метаданными
	for i := range newBook {
		newBook[i].Id = currentIdBooks //присвоим Id по порядку от глобального счетчика
		currentIdBooks++
		newBook[i].modification = r.RemoteAddr + ";" + r.UserAgent() +
			";" + time.Now().Format("2006/01/02 15:05:04")
	}

	//добавляем новые данные в общий срез
	Books = append(Books, newBook...)
	w.WriteHeader(http.StatusCreated)
	printAll(w, newBook)
}

//GetBooks отдает полный список Book коллекции в JSON
func GetBooks(w http.ResponseWriter, r *http.Request) {
	printAll(w, Books)
}

//GetBook выдает информацию в JSON о Book по её Id
func GetBook(w http.ResponseWriter, r *http.Request) {

	tmp, err := strconv.Atoi(mux.Vars(r)["Id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("getBook: Id не прочли, ошибка: %s", err)
		return
	}
	id := uint32(tmp)
	for _, b := range Books {
		if b.Id == id { //до первого совпадения Id
			printAll(w, []Book{b})
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

//UpdateBook обновляет информацию о Book в коллекции по её Id
func UpdateBook(w http.ResponseWriter, r *http.Request) {

	//читаем тело запроса в карту - не знаем, сколько и каких полей может встретиться

	var rBody map[string]string

	//тело запроса
	dec := json.NewDecoder(r.Body)
	for dec.More() {
		err := dec.Decode(&rBody)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("не прочли тело запроса, ошибка: %s", err)
			return
		}
	}

	//выясняем id обновляемой карточки
	tmp, err := strconv.Atoi(mux.Vars(r)["Id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Id не прочли, ошибка: %s", err)
		return
	}
	id := uint32(tmp)

	for i := range Books {
		if Books[i].Id == id { //до первого совпадения Id

			//рефлексируем по структуре данных, по каждому найденному полю
			//проверяем карту обновленных данных и если нашли - обновляем поле в структуре

			val := reflect.ValueOf(&Books[i]).Elem() //адресуемое значение
			for ii := 0; ii < val.NumField(); ii++ { //идем по полям структуры
				if tmp, ok := rBody[val.Type().Field(ii).Tag.Get("json")]; ok {
					//в карте нашли поле, его обновляем
					val.Field(ii).Set(reflect.ValueOf(tmp))
				}
			}
			Books[i].modification = r.RemoteAddr + ";" + r.UserAgent() +
				";" + time.Now().Format("2006/01/02 15:05:04")

			printAll(w, []Book{Books[i]}) //или просто w.WriteHeader(http.StatusOK)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

//DeleteBook удаляет Book в коллекции по её Id
func DeleteBook(w http.ResponseWriter, r *http.Request) {

	tmp, err := strconv.Atoi(mux.Vars(r)["Id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Id не прочли, ошибка: %s", err)
		return
	}
	id := uint32(tmp)
	for i, b := range Books {
		if b.Id == id { //до первого совпадения Id
			Books = append(Books[:i], Books[i+1:]...) //удаляем i элемент
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

//FindBook находит все Book по заданным текстовым параметрам - подстрокам.
//Неизвестные и некорректные ключи пропускаются
func FindBook(w http.ResponseWriter, r *http.Request) {

	var foundBook []Book //массив для найденных Book

	param := r.URL.Query() //карта карт всех принятых параметров

	for _, key := range publicKeyBooks { // по всем возможным ключам
		if mapCurrentKey, exist := param[key]; exist { //если ключ в карте есть
			for _, value := range mapCurrentKey { //по всем значениям ключа (их может быть не один)
				value = strings.ToLower(value) //все будем сравнивать без учета регистра
				for i := range Books {         //по всем Book коллекции
					val := reflect.ValueOf(&Books[i]).Elem() //адресуемое значение
					for ii := 0; ii < val.NumField(); ii++ { //идем по полям структуры
						if strings.EqualFold(val.Type().Field(ii).Tag.Get("json"), key) { //нашли поле в структуре, совпадающее по названию с ключом
							if strings.Contains(strings.ToLower(val.Field(ii).String()), value) { //в значении поля есть искомая подстрока
								if Books[i].popular < math.MaxUint8 { //если был запрошен в поиске - повысим популярность
									Books[1].popular++
								}
								foundBook = append(foundBook, Books[i]) //копим все найденные Book
							}
						}
					}
				}
			}
		}
	}
	if len(foundBook) > 0 {
		printAll(w, foundBook)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	printAll(w, fmt.Sprintf("Ничего не нашли :(. Возможные ключи для поиска в Book: %s", publicKeyBooks))
}
