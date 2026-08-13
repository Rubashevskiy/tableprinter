package tableprinter

/*
Примеры тегов:
`table:"l;limit:10"` // Прижать влево ограничить общую длину в 10 символов
`table:"r;bool:✔/✘;limit:1"` // Прижать вправо, установить кастомный вид, обрезать строку
`table:"l;time:2006-01-02 15:04:05;nil:Не назначено;limit:19"` // Прижать влево кастомный формат времени, если nil установить 'Не назначено' обрезать до 19 сим. 
`table:"r;prec:2"`           // Отформатирует decimal с 2 знаками после запятой
`table:"r;prec:4;limit:10"`   // Отформатирует с 4 знаками и ограничит общую длину в 10 символов
`table:"l;prec:0"`           // Отформатирует как целое число (без знаков после точки)
*/

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Описание метаданных поля, полученных через теги
type fieldInfo struct {
	index      int    // индекс поля в структуре
	format     string // "l" или "r" к какому краю прижимать строку в столбце
	timeFormat string // кастомный формат времени
	nilValue   string // кастомная строка для nil-значений
	boolTrue   string // строка для значения true
	boolFalse  string // строка для значения false
	limit      int    // максимальная длина колонки в символах (0 — без ограничений)
	precision  int    // количество знаков после запятой для типов с фиксированной точкой (-1 — не задано)
}

// Table использует дженерик T, который должен быть структурой
type Table[T any] struct {
	name   string 
	space  int
	fields []fieldInfo
	data   []T
}

// NewTable автоматически строит формат таблицы на основе тегов структуры T
func NewTable[T any](name string, space int) (*Table[T], error) {
	var sample T
	valType := reflect.TypeOf(sample)
	if valType.Kind() == reflect.Ptr {
		valType = valType.Elem()
	}
	if valType.Kind() != reflect.Struct {
		return nil, errors.New("table generic type must be a struct")
	}

	var fields []fieldInfo
	for i := 0; i < valType.NumField(); i++ {
		field := valType.Field(i)
		if !field.IsExported() {
			continue
		}

		tagStr := field.Tag.Get("table")
		if tagStr == "" {
			continue
		}

		parts := strings.Split(tagStr, ";")
		align := strings.TrimSpace(parts[0])
		if align != "l" && align != "r" {
			return nil, fmt.Errorf("invalid table tag alignment '%s' for field %s", align, field.Name)
		}

		// Значения по умолчанию
		timeFmt := "2006-01-02 15:04:05"
		nilVal := "nil"
		bTrue := "True"
		bFalse := "False"
		limitVal := 0
		precisionVal := -1 // По умолчанию -1, чтобы отличить от явно заданного `prec:0`

		// Разбираем дополнительные параметры k:v
		for _, part := range parts[1:] {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			kv := strings.SplitN(part, ":", 2)
			if len(kv) != 2 {
				return nil, fmt.Errorf("invalid tag parameter format '%s' for field %s", part, field.Name)
			}

			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])

			switch key {
			case "time":
				timeFmt = val
			case "nil":
				nilVal = val
			case "bool":
				boolParts := strings.SplitN(val, "/", 2)
				if len(boolParts) == 2 {
					bTrue = strings.TrimSpace(boolParts[0])
					bFalse = strings.TrimSpace(boolParts[1])
				} else {
					bTrue = strings.TrimSpace(boolParts[0])
				}
			case "limit":
				var err error
				limitVal, err = strconv.Atoi(val)
				if err != nil || limitVal < 0 {
					return nil, fmt.Errorf("invalid limit value '%s' for field %s", val, field.Name)
				}
			case "prec":
				var err error
				precisionVal, err = strconv.Atoi(val)
				if err != nil || precisionVal < 0 {
					return nil, fmt.Errorf("invalid precision value '%s' for field %s", val, field.Name)
				}
			default:
				return nil, fmt.Errorf("unknown tag parameter '%s' for field %s", key, field.Name)
			}
		}

		fields = append(fields, fieldInfo{
			index:      i,
			format:     align,
			timeFormat: timeFmt,
			nilValue:   nilVal,
			boolTrue:   bTrue,
			boolFalse:  bFalse,
			limit:      limitVal,
			precision:  precisionVal,
		})
	}

	if len(fields) == 0 {
		return nil, errors.New("struct must have at least one exported field with 'table' tag")
	}

	return &Table[T]{
		name: name,
		space:  space,
		fields: fields,
		data:   make([]T, 0),
	}, nil
}

func (t *Table[T]) AddRow(row T) *Table[T] {
	t.data = append(t.data, row)
	return t
}

func (t *Table[T]) AddRows(rows []T) *Table[T] {
	t.data = append(t.data, rows...)
	return t
}

// Render собирает таблицу вместе с верхней шапкой имени (линия подстраивается под ширину таблицы)
func (t *Table[T]) Render(useFigureSpace bool) string {
	if len(t.data) == 0 {
		// Если данных нет, выводим только имя (линия по длине имени)
		if t.name == "" {
			return ""
		}
		var buf strings.Builder
		nameLen := utf8.RuneCountInString(t.name)
		border := strings.Repeat("-", nameLen)
		buf.WriteString(border)
		buf.WriteString("\n")
		buf.WriteString(t.name)
		buf.WriteString("\n")
		buf.WriteString(border)
		buf.WriteString("\n")
		return buf.String()
	}

	spaceChar := " "
	if useFigureSpace {
		spaceChar = "\u2007"
	}

	colCount := len(t.fields)
	maxColSize := make([]int, colCount)
	stringMatrix := make([][]string, len(t.data))

	// 1. Сначала заполняем матрицу строк и вычисляем максимальную ширину колонок
	for rowIndex, rowData := range t.data {
		stringMatrix[rowIndex] = make([]string, colCount)
		v := reflect.ValueOf(rowData)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		for colIndex, field := range t.fields {
			fieldVal := v.Field(field.index)
			var strVal string

			if isNilable(fieldVal.Kind()) && fieldVal.IsNil() {
				strVal = field.nilValue
			} else {
				actualInterface := fieldVal.Interface()
				if fieldVal.Kind() == reflect.Ptr {
					actualInterface = fieldVal.Elem().Interface()
				}

				matched := false
				method := fieldVal.MethodByName("StringFixed")
				if !method.IsValid() && fieldVal.CanAddr() {
					method = fieldVal.Addr().MethodByName("StringFixed")
				}

				if method.IsValid() {
					prec := int32(2)
					if field.precision >= 0 {
						prec = int32(field.precision)
					}
					args := []reflect.Value{reflect.ValueOf(prec)}
					results := method.Call(args)
					if len(results) > 0 && results[0].Kind() == reflect.String {
						strVal = results[0].String()
						matched = true
					}
				}

				if !matched {
					switch val := actualInterface.(type) {
					case time.Time:
						if val.IsZero() {
							strVal = field.nilValue
						} else {
							strVal = val.Format(field.timeFormat)
						}
					case bool:
						if val {
							strVal = field.boolTrue
						} else {
							strVal = field.boolFalse
						}
					default:
						strVal = fmt.Sprint(actualInterface)
					}
				}
			}

			if field.limit > 0 {
				runes := []rune(strVal)
				if len(runes) > field.limit {
					strVal = string(runes[:field.limit])
				}
			}

			stringMatrix[rowIndex][colIndex] = strVal
			runes := utf8.RuneCountInString(strVal)
			if maxColSize[colIndex] < runes {
				maxColSize[colIndex] = runes
			}
		}
	}

	// 2. Вычисляем общую максимальную ширину всей таблицы для линии
	totalTableWidth := 0
	for _, size := range maxColSize {
		totalTableWidth += size
	}
	if colCount > 1 {
		totalTableWidth += (colCount - 1) * t.space
	}

	// Если имя таблицы длиннее, чем сама таблица, расширяем линию до длине имени
	nameLen := utf8.RuneCountInString(t.name)
	if nameLen > totalTableWidth {
		totalTableWidth = nameLen
	}

	// 3. Формируем итоговый буфер (начиная со сложной шапки)
	var buf strings.Builder

	if t.name != "" {
		border := strings.Repeat("-", totalTableWidth)
		buf.WriteString(border)
		buf.WriteString("\n")
		buf.WriteString(t.name)
		buf.WriteString("\n")
		buf.WriteString(border)
		buf.WriteString("\n")
	}

	// 4. Отрисовываем строки данных
	rowSpacing := strings.Repeat(spaceChar, t.space)
	for rowIndex := range stringMatrix {
		for colIndex := 0; colIndex < colCount; colIndex++ {
			cell := t.genCell(
				maxColSize[colIndex],
				t.fields[colIndex].format,
				stringMatrix[rowIndex][colIndex],
				spaceChar,
			)
			buf.WriteString(cell)
			if colIndex < colCount-1 {
				buf.WriteString(rowSpacing)
			}
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

func (t *Table[T]) genCell(targetLen int, format string, data string, spaceChar string) string {
	currentLen := utf8.RuneCountInString(data)
	if currentLen >= targetLen {
		return data
	}

	padding := strings.Repeat(spaceChar, targetLen-currentLen)
	if format == "r" {
		return padding + data
	}
	return data + padding
}

func isNilable(kind reflect.Kind) bool {
	switch kind {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	default:
		return false
	}
}
