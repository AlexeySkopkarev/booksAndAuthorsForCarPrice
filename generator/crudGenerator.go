//Рутинный генератор CRUD-а и функций поиска для всех объявленных в main.go моделей
//Создается файл crud.go
//
//
//

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
	"text/template"
)

type modelsType struct {
	name         string
	publicFields []string
}

var (
	//variable содержит описание переменных для хранения моделей и начальных идентификаторов
	variable = template.Must(template.New("variable").Parse(`
	var (// для {{.Model}}:
	{{.Model}}s            []{{.Model}} //основная структура данных для модели {{.Model}}s
	currentId{{.Model}}s   uint32   = 1 //начальный идентификатор для модели {{.Model}}s
	publicKey{{.Model}}s    = []string{ {{.KeysArray}} }//массив всех публичных полей структуры
)
`))

	//headerFile содержит заголовок файла
	headerFile = `//Этот файл сгенерирован автоматически, не изменяйте его

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
)`

	//initRouter описывает связь между путями и методами для Model
	initRouter = template.Must(template.New("initRouter").Parse(`
	router.HandleFunc("/{{.Path}}s",       Create{{.Model}}s).Methods("POST")
	router.HandleFunc("/{{.Path}}s",       Get{{.Model}}s).Methods("GET")
	router.HandleFunc("/{{.Path}}s/{Id}",  Get{{.Model}}).Methods("GET")
	router.HandleFunc("/{{.Path}}s/{Id}",  Update{{.Model}}).Methods("PUT")
	router.HandleFunc("/{{.Path}}s/{Id}",  Delete{{.Model}}).Methods("DELETE")
	router.HandleFunc("/{{.Path}}s/find/", Find{{.Model}}).Methods("GET")
`))

	//crudFunction описывает шаблон функций стандартного CRUDа для Model
	crudFunction = template.Must(template.New("crudFunction").Parse(`
//Create{{.Model}}s создает новые записи в коллекции из JSON тела POST запроса.
//JSON должен быть массивом объектов [{}...]
func Create{{.Model}}s(w http.ResponseWriter, r *http.Request) {
	//прочтем всё тело запроса
	rBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("не прочли тело запроса: %v", rBody)
		return
	}

	//срез для приема данных
	var new{{.Model}} []{{.Model}}

	//парсим в структуру тело запроса
	err = json.Unmarshal(rBody, &new{{.Model}})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("не расшифровали тело запроса: %s,\n Ошибка: %s", rBody, err)
		return
	}

	//дополняем коллекцию метаданными
	for i := range new{{.Model}} {
		new{{.Model}}[i].Id = currentId{{.Model}}s //присвоим Id по порядку от глобального счетчика
		currentId{{.Model}}s++
		new{{.Model}}[i].modification = r.RemoteAddr + ";" + r.UserAgent() +
			";" + time.Now().Format("2006/01/02 15:05:04")
	}

	//добавляем новые данные в общий срез
	{{.Model}}s = append({{.Model}}s, new{{.Model}}...)
	w.WriteHeader(http.StatusCreated)
	printAll(w, new{{.Model}})
}

//Get{{.Model}}s отдает полный список {{.Model}} коллекции в JSON
func Get{{.Model}}s(w http.ResponseWriter, r *http.Request) {
	printAll(w, {{.Model}}s)
}

//Get{{.Model}} выдает информацию в JSON о {{.Model}} по её Id
func Get{{.Model}}(w http.ResponseWriter, r *http.Request) {

	tmp, err := strconv.Atoi(mux.Vars(r)["Id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("get{{.Model}}: Id не прочли, ошибка: %s", err)
		return
	}
	id := uint32(tmp)
	for _, b := range {{.Model}}s {
		if b.Id == id { //до первого совпадения Id
			printAll(w, []{{.Model}}{b})
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

//Update{{.Model}} обновляет информацию о {{.Model}} в коллекции по её Id
func Update{{.Model}}(w http.ResponseWriter, r *http.Request) {

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

	for i := range {{.Model}}s {
		if {{.Model}}s[i].Id == id { //до первого совпадения Id

			//рефлексируем по структуре данных, по каждому найденному полю
			//проверяем карту обновленных данных и если нашли - обновляем поле в структуре

			val := reflect.ValueOf(&{{.Model}}s[i]).Elem() //адресуемое значение
			for ii := 0; ii < val.NumField(); ii++ {    //идем по полям структуры
				if tmp, ok := rBody[val.Type().Field(ii).Tag.Get("json")]; ok {
					//в карте нашли поле, его обновляем
					val.Field(ii).Set(reflect.ValueOf(tmp))
				}
			}
			{{.Model}}s[i].modification = r.RemoteAddr + ";" + r.UserAgent() +
				";" + time.Now().Format("2006/01/02 15:05:04")

			printAll(w, []{{.Model}}{ {{.Model}}s[i] }) //или просто w.WriteHeader(http.StatusOK)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

//Delete{{.Model}} удаляет {{.Model}} в коллекции по её Id
func Delete{{.Model}}(w http.ResponseWriter, r *http.Request) {

	tmp, err := strconv.Atoi(mux.Vars(r)["Id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Id не прочли, ошибка: %s", err)
		return
	}
	id := uint32(tmp)
	for i, b := range {{.Model}}s {
		if b.Id == id { //до первого совпадения Id
			{{.Model}}s = append({{.Model}}s[:i], {{.Model}}s[i+1:]...) //удаляем i элемент
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

//Find{{.Model}} находит все {{.Model}} по заданным текстовым параметрам - подстрокам.
//Неизвестные и некорректные ключи пропускаются
func Find{{.Model}}(w http.ResponseWriter, r *http.Request) {

	var found{{.Model}} []{{.Model}} //массив для найденных {{.Model}}

	param := r.URL.Query() //карта карт всех принятых параметров

	for _, key := range publicKey{{.Model}}s { // по всем возможным ключам
		if mapCurrentKey, exist := param[key]; exist { //если ключ в карте есть
			for _, value := range mapCurrentKey { //по всем значениям ключа (их может быть не один)
				value = strings.ToLower(value) //все будем сравнивать без учета регистра
				for i := range {{.Model}}s {         //по всем {{.Model}} коллекции
					val := reflect.ValueOf(&{{.Model}}s[i]).Elem() //адресуемое значение
					for ii := 0; ii < val.NumField(); ii++ { //идем по полям структуры
						if strings.EqualFold(val.Type().Field(ii).Tag.Get("json"), key) { //нашли поле в структуре, совпадающее по названию с ключом
							if strings.Contains(strings.ToLower(val.Field(ii).String()), value) { //в значении поля есть искомая подстрока
								if {{.Model}}s[i].popular < math.MaxUint8 {//если был запрошен в поиске - повысим популярность
		 							{{.Model}}s[1].popular++
								}
								found{{.Model}} = append(found{{.Model}}, {{.Model}}s[i]) //копим все найденные {{.Model}}
							}
						}
					}
				}
			}
		}
	}
	if len(found{{.Model}}) > 0 {
		printAll(w, found{{.Model}})
		return
	}
	w.WriteHeader(http.StatusNotFound)
	printAll(w, fmt.Sprintf("Ничего не нашли :(. Возможные ключи для поиска в {{.Model}}: %s", publicKey{{.Model}}s))
}
`))
)

func main() {

	//прочтем файл main, узнаем какие есть модели и какие у них поля,
	//сложим в структуру models{name,publicFields}

	node, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("кодогенерация: не распарсили main-файл, ошибка: %s", err)
	}
	var models []modelsType

	for _, decl := range node.Decls {
		genD, ok := decl.(*ast.GenDecl)
		if ok { //это генеральная декларация (импорты, типы, константы и переменные)
			for _, spec := range genD.Specs {
				currType, ok := spec.(*ast.TypeSpec)
				if ok { // и это описание типа
					currStruct, ok := currType.Type.(*ast.StructType)
					if ok { //и этот тип - структура
						if genD.Doc != nil { //у него есть комменты
							for _, comment := range genD.Doc.List {
								if comment.Text == "//for generate" { //в том числе нужный коммент
									var strField []string //соберем поля с разрешенным тегом "доступны для поиска"
									for _, field := range currStruct.Fields.List {
										if strings.Contains(field.Tag.Value, "find:\"yes\"") {
											temp := "\"" + strings.ToLower(field.Names[0].Name) + "\"" //подготовим особым образом элемент
											strField = append(strField, temp)
										}
									}
									models = append(models, modelsType{currType.Name.Name, strField})
								}
							}
						}
					}
				}

			}
		}
	}
	if len(models) == 0 {
		log.Fatalln("не нашли ни одной структуры моделей")
	}

	//создаем файл для генерации кода
	f, err := os.Create("crud.go")
	if err != nil {
		log.Fatalf("файл не создался, ошибка: %s", err)
	}

	defer f.Close()

	//пишем заголовок файла
	_, err = fmt.Fprintln(f, headerFile)
	if err != nil {
		log.Fatalf("заголовок файла не записался, ошибка: %s", err)
	}

	//пишем переменные для хранения моделей, начальных ID, перечень полей структур, которые используем для поиска
	for _, model := range models {
		err = variable.Execute(f, struct {
			Model     string
			KeysArray string
		}{model.name, strings.Join(model.publicFields, ",")})
		if err != nil {
			log.Fatalf("шаблон variable.%s не построился, ошибка: %s", model.name, err)
		}

	}

	//генерируем все пути
	_, err = fmt.Fprintln(f, `func InitRouter() http.Handler{
	router := mux.NewRouter()`)
	if err != nil {
		log.Fatalf("заголовок InitRouter не записался, ошибка: %s", err)
	}

	for _, model := range models {
		err = initRouter.Execute(f, struct {
			Model string
			Path  string
		}{model.name, strings.ToLower(model.name)})
		if err != nil {
			log.Fatalf("шаблон initRouter.%s не построился, ошибка: %s", model.name, err)
		}
	}

	_, err = fmt.Fprintln(f, `	return router}`)
	if err != nil {
		log.Fatalf("конец InitRouter не записался, ошибка: %s", err)
	}

	//генерируем функции
	for _, model := range models {
		err = crudFunction.Execute(f, struct{ Model string }{model.name})
		if err != nil {
			log.Fatalf("шаблон crudFunction.%s не построился, ошибка: %s", model.name, err)
		}
	}
}
