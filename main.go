package main

//go:generate go run .\generator\crudGenerator.go
//go:generate go fmt

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

//Структуры модели. Тег json есть у полей, доступных для маршалинга в тело ответа,
//тег find="yes" есть у полей, по которым пользователь может осуществлять поиск.
//Для добавления моделей в проект достаточно ниже описать новую структуру
//с комментом "//for generate" и запустить генератор для формирования
//соответствующих переменных, путей и ручек.
//поля Id, popular, modification - обязательные (служебные)
//

//for generate
type Author struct {
	Id            uint32   `json:"id" find:"no"`            //уникальный идентификатор
	Name          string   `json:"name" find:"yes"`         //имя автора
	BirthYear     string   `json:"birthYear" find:"yes"`    //год рождения
	DeathYear     string   `json:"deathYear" find:"yes"`    //год смерти (если есть)
	Books         string   `json:"books" find:"yes"`        //перечисление всех его книг в строку, заполняется автоматически
	BooksQuantity uint32   `json:"booksQuantity" find:"no"` //количество книг автора в библиотеке
	booksId       []uint32 `find:"no"`                      //массив идентификаторов всех его книг, заполняется автоматически по совпадению имени
	popular       uint8    `find:"no"`                      //рейтинг автора от 0 до 255 - фактически количество запросов в поиске
	modification  string   `find:"no"`                      //дата-время и автор последней модификации
}

//for generate
type Book struct {
	Id           uint32 `json:"id" find:"no"`          //уникальный идентификатор
	Name         string `json:"name" find:"yes"`       //название книги
	PublicYear   string `json:"publicYear" find:"yes"` //год публикации
	Annotation   string `json:"annotation" find:"yes"` //краткое содержание
	Authors      string `json:"authors" find:"yes"`    //строка с именем автора, авторов (через ";")
	popular      uint8  `find:"no"`                    //рейтинг книги от 0 до 255 - фактически количество запросов в поиске
	modification string `find:"no"`                    //дата-время и автор последней модификации
}

func main() {

	router := InitRouter() // инициализируем роутер

	go librarian() //запускаем библиотекаря

	fmt.Println("Server is listening...")

	err := http.ListenAndServe(":8181", router)
	if err != nil {
		log.Fatalf("сервер не стартанул, ошибка: %s", err)
	}

}

//printAll формирует и посылает тело в JSON
func printAll(w http.ResponseWriter, body interface{}) {

	//складываем в JSON входящий срез
	wBody, err := json.Marshal(body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("не смогли закодировать в JSON: %v,\nошибка: %s", body, err)
		return
	}

	//формируем тело ответа
	_, err = w.Write(wBody)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("не отдали тело ответа: %v,\nошибка: %s", wBody, err)
	}
}
