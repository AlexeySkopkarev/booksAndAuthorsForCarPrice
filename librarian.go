package main

import (
	"reflect"
	"strings"
	"time"
)

//librarian просматривает библиотеку и делает анализ, обобщение и исправление данных, но чаще - просто спит ;)
//Цель - вести актуальный реестр книг, авторов книг библиотеки и реестр всех книг каждого автора
func librarian() {

	for {

		//===============================================================================================
		//Должностная обязанность: "Борьба с дублями". Алгоритм:
		//для каждой книги проверяем совпадение имен, если имя совпадает,
		//то сравниваем последовательно остальные публичные поля;
		//если в обеих карточках все непустые поля совпадают, то запускаем процедуру сливания.
		//Сливание происходит по правилу:
		//в старую (с меньшим ID) карточку заливаются ненулевые поля новой (с большим ID), более актуальной, карточки,
		//последняя передается "в другой отдел" на удаление самой и всех ее вхождений; дата старой карточки обновляется;
		for i := range Books {
			for ii := i + 1; ii < len(Books); ii++ {
				if Books[i].Name == Books[ii].Name && Books[i].Name != "" {
					match := true //флаг совпадения
					//рефлексируем по структуре
					val1 := reflect.ValueOf(&Books[i]).Elem()
					val2 := reflect.ValueOf(&Books[ii]).Elem()
					for iii := 0; iii < val1.NumField(); iii++ { //по всем полям структуры
						if reflect.VisibleFields(reflect.TypeOf(&Books[i]).Elem())[iii].Type.String() == "string" && //пропускаем не строки
							reflect.VisibleFields(reflect.TypeOf(&Books[i]).Elem())[iii].Name != "modification" { //пропускаем уникальное поле modification
							match = match && (strings.EqualFold(val1.Field(iii).String(), val2.Field(iii).String()) || //логическое умножение - совпадают или
								val1.Field(iii).String() == "" || val2.Field(iii).String() == "") //одно из них - пустое
						}
					}
					if match { //считаем, что дубль, сливаем карточки
						for iii := 0; iii < val1.NumField(); iii++ {
							if reflect.VisibleFields(reflect.TypeOf(&Books[i]).Elem())[iii].Type.String() == "string" && //пропускаем не строки
								reflect.VisibleFields(reflect.TypeOf(&Books[i]).Elem())[iii].Name != "modification" && //пропускаем уникальное поле modification
								val2.Field(iii).String() != "" { //непустые поля
								val1.Field(iii).Set(val2.Field(iii)) //копируем из карточки с более поздним ID
							}
						}
						Books[i].modification = "local;librarian;" + time.Now().Format("2006/01/02 15:05:04") //обновляем поле модификации
						Books[ii].Name = ""                                                                   //помечаем карточку как "плохую" - зануляем имя
					}
				}
			}
		}

		//===============================================================================================
		//Должностная обязанность: "Удаление плохих карточек".
		//Цель - удаление карточек с пустым обязательным полем Name.
		//Такие поля могут создаваться пользователем, но не имеют смысла.
		//За один проход составляем срез индексов хороших карточек, затем мастерим новый срез книг из отобранных книг.
		//Неэффективно, но это же библиотекарь.

		var idxGoodBooksSlice []int              //срез индексов хороших книг
		badBooksMap := make(map[string][]uint32) //карта [имя автора][]id книги на удаление

		for i := range Books {
			if Books[i].Name != "" {
				//запишем индекс книги в список хороших
				idxGoodBooksSlice = append(idxGoodBooksSlice, i)
			} else {
				//запишем id книг на удаление и name авторов этих книг
				if slice, ok := badBooksMap[Books[i].Authors]; ok {
					badBooksMap[Books[i].Authors] = append(slice, Books[i].Id)
				} else {
					badBooksMap[Books[i].Authors] = []uint32{Books[i].Id}
				}

			}
		}
		//сначала вычистим каталог книг - составим новый каталог только из хороших карточек
		tempNewBooks := make([]Book, 0, cap(Books)) //для исключения аллокаций создаем срез идентичный по размеру существующему срезу книг
		for _, i := range idxGoodBooksSlice {
			tempNewBooks = append(tempNewBooks, Books[i])
		}
		Books = tempNewBooks //тут тоже надо делать безопасно!!

		//затем вычистим плохие карточки из каталогов книг авторов
		for auth, idbb := range badBooksMap {
			for i := range Authors { //ищем автора
				if Authors[i].Name == auth { //нашли
					nameSlice := strings.FieldsFunc(Authors[i].Books, func(c rune) bool { //из строки делаем срез наименований книг
						return c == ';'
					})
					for _, idbb := range idbb { //выбираем все плохие книги этого автора
						for ii, ib := range Authors[i].booksId { //для каждой книги в списке автора проверим вхождение ее в список на удаление
							if ib == idbb {
								//ii-индекс в двух слайсах на удаление
								Authors[i].booksId = append(Authors[i].booksId[:ii], Authors[i].booksId[ii+1:]...)
								Authors[i].Books = strings.Join(append(nameSlice[:ii], nameSlice[ii+1:]...), ";")
								Authors[i].BooksQuantity--
								break
							}
						}
					}
					break
				}
			}
		}

		//===============================================================================================
		//Должностная обязанность: "Актуализация списка авторов". Алгоритм:
		//для каждой книги берем каждого автора и ищем его в списке авторов:
		// - если нашли автора, то ищем в списке его книг нашу текущую книгу,
		//   -- если не нашли - добавляем автору новую книгу;
		// - если не нашли автора - добавляем в список авторов нового автора и сразу указываем ему его книгу

		for _, book := range Books { //для каждой книги из коллекции
			Auths := strings.Split(book.Authors, ";") //авторов не один?
			for _, currAut := range Auths {           //по каждому автору текущей книги отдельно
				if currAut != "" { //только не пустые авторы!
					var autFlag bool                 //флаг, что автор найден в общем списке авторов
					for i, author := range Authors { //просматриваем список всех авторов в коллекции
						if autFlag {
							break
						}
						if currAut == author.Name { //этот автор уже есть в списке
							autFlag = true
							var booFlag bool                       //флаг, что у автора есть такая книга в его списке книг
							for _, booId := range author.booksId { //проверяем список книг у найденного автора
								if booId == book.Id { //такая книга уже есть у этого автора
									booFlag = true
									break //дальше список книг у этого автора не смотрим
								}
							}
							if !booFlag { //по списку книг прошли, но ID этой книги у автора не нашли - добавим
								Authors[i].booksId = append(Authors[i].booksId, book.Id) //добавили известному автору новую книгу
								Authors[i].Books = Authors[i].Books + ";" + book.Name
								Authors[i].BooksQuantity++
								break //дальше коллекцию авторов не смотрим
							}
						}
					}
					if !autFlag { //по списку авторов прошли, но автора не обнаружили, значит это новичок - добавляем
						Authors = append(Authors, Author{
							Id:            currentIdAuthors,
							Name:          currAut,
							Books:         book.Name,
							booksId:       []uint32{book.Id},
							BooksQuantity: 1,
							modification:  "local;librarian;" + time.Now().Format("2006/01/02 15:05:04"),
						})
						currentIdAuthors++ //!!!!тут надо обеспечить безопасность
					}
				}
			}
		}

		//===============================================================================================
		//Должностная обязанность: "Спать 15 секунд и не мешать".
		time.Sleep(time.Second * 15)
	}
}
