package generator

import (
	"fmt"
	"react-to-templ-converter/internal/config"
	"react-to-templ-converter/internal/models"
	"strings"
)

// TemplGenerator генерирует templ шаблоны из React компонентов
type TemplGenerator struct {
	options     *config.ConversionOptions
	debug       bool
	indentSize  int
	indentStyle string
	jsxToHtml   JSXToHTMLConverter
}

// JSXToHTMLConverter определяет интерфейс для конвертации JSX в HTML
type JSXToHTMLConverter interface {
	ConvertJSXToTempl(jsx *models.JSXElement, indent int) string
	SetDebug(debug bool)
}

// NewTemplGenerator создает новый генератор templ шаблонов
func NewTemplGenerator(options *config.ConversionOptions) *TemplGenerator {
	return &TemplGenerator{
		options:     options,
		debug:       options.Debug,
		indentSize:  options.Indentation.Size,
		indentStyle: options.Indentation.Style,
	}
}

// SetJSXConverter устанавливает конвертер JSX в HTML
func (g *TemplGenerator) SetJSXConverter(converter JSXToHTMLConverter) {
	g.jsxToHtml = converter
}

// SetDebug устанавливает режим отладки
func (g *TemplGenerator) SetDebug(debug bool) {
	g.debug = debug
	if g.jsxToHtml != nil {
		g.jsxToHtml.SetDebug(debug)
	}
}

// SetIndentation устанавливает стиль отступов
func (g *TemplGenerator) SetIndentation(style string, size int) {
	g.indentStyle = style
	g.indentSize = size
}

// GenerateTemplFile создает полный templ файл для React компонента
func (g *TemplGenerator) GenerateTemplFile(component *models.ReactComponent) string {
	var sb strings.Builder

	// 1. Генерация заголовка файла (пакет, импорты)
	sb.WriteString(g.generateFileHeader(component))

	// 2. Генерация структуры пропсов
	if len(component.Props) > 0 {
		sb.WriteString(g.generatePropsStruct(component))
	}

	// 3. Генерация вспомогательных структур для состояний
	if len(component.State) > 0 {
		sb.WriteString(g.generateStateStructs(component))
	}

	// 4. Генерация templ компонента
	sb.WriteString(g.generateTemplComponent(component))

	// 5. Генерация вспомогательных функций (если нужны)
	if g.needsHelperFunctions(component) {
		sb.WriteString(g.generateHelperFunctions(component))
	}

	return sb.String()
}

// generateFileHeader генерирует заголовок файла с пакетом и импортами
func (g *TemplGenerator) generateFileHeader(component *models.ReactComponent) string {
	var sb strings.Builder

	// Пакет
	packageName := "templates"
	if g.options.PackageName != "" && g.options.PackageName != "." {
		packageName = g.options.PackageName
	}
	sb.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// Импорты
	imports := g.detectRequiredImports(component)
	if len(imports) > 0 {
		sb.WriteString("import (\n")
		for _, imp := range imports {
			sb.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
		}
		sb.WriteString(")\n\n")
	}

	return sb.String()
}

// generatePropsStruct генерирует структуру Go для пропсов
func (g *TemplGenerator) generatePropsStruct(component *models.ReactComponent) string {
	var sb strings.Builder

	// Комментарий
	sb.WriteString(fmt.Sprintf("// %sProps определяет пропсы для компонента\n", component.Name))
	sb.WriteString(fmt.Sprintf("type %sProps struct {\n", component.Name))

	indent := g.getIndentation(1)

	// Поля структуры
	for _, prop := range component.Props {
		goType := g.convertTypeToGo(prop.Type)
		fieldName := strings.Title(prop.Name) // Title вместо ToUpper для совместимости с Go conventions

		// Комментарий о необходимости заполнения
		if prop.Required {
			sb.WriteString(fmt.Sprintf("%s// %s обязательное поле\n", indent, fieldName))
		}

		sb.WriteString(fmt.Sprintf("%s%s %s\n", indent, fieldName, goType))
	}

	sb.WriteString("}\n\n")

	return sb.String()
}

// generateStateStructs генерирует структуры для хранения состояний
func (g *TemplGenerator) generateStateStructs(component *models.ReactComponent) string {
	// В templ файле не нужно включать структуры состояний - они будут в Go контроллерах
	// Эта функция оставлена на случай, если есть локальные состояния, которые нужны только в templ

	return ""
}

// generateTemplComponent генерирует основной templ компонент
func (g *TemplGenerator) generateTemplComponent(component *models.ReactComponent) string {
	var sb strings.Builder

	// Проверяем, что имя компонента не пустое
	if component.Name == "" {
		// Используем имя компонента из опций или значение по умолчанию
		if g.options.ComponentName != "" {
			component.Name = g.options.ComponentName
		} else {
			component.Name = "Counter" // Настраиваем по умолчанию для счетчика
		}
	}

	// Имя функции templ с маленькой буквы
	funcName := strings.ToLower(string(component.Name[0])) + component.Name[1:]

	// Параметры функции
	params := ""
	if len(component.Props) > 0 {
		params = fmt.Sprintf("props %sProps", component.Name)
	}

	// Если используем HTMX, добавим параметр id
	if g.options.UseHtmx {
		if params != "" {
			params += ", "
		}
		params += "id string"
	}

	// Параметр для хранения состояния
	if len(component.State) > 0 {
		if params != "" {
			params += ", "
		}
		for _, state := range component.State {
			if state.Type == "number" {
				params += fmt.Sprintf("%s int", state.Name)
				break
			} else if state.Type == "string" {
				params += fmt.Sprintf("%s string", state.Name)
				break
			} else {
				params += fmt.Sprintf("%s interface{}", state.Name)
				break
			}
		}
	}

	// Определение templ компонента
	sb.WriteString(fmt.Sprintf("templ %s(%s) {\n", funcName, params))

	// Если JSX есть, используем его
	if component.JSX != nil {
		var jsxTemplate string
		if g.jsxToHtml != nil {
			jsxTemplate = g.jsxToHtml.ConvertJSXToTempl(component.JSX, 1)
		} else {
			jsxTemplate = g.simpleJSXToTempl(component, component.JSX, 1)
		}
		sb.WriteString(jsxTemplate)
	} else if len(component.State) > 0 {
		// Если JSX нет, но есть состояния, генерируем базовый компонент
		indent := g.getIndentation(1)

		// Внешний div с ID для HTMX (если используется)
		if g.options.UseHtmx {
			sb.WriteString(fmt.Sprintf("%s<div id=\"%s-{id}\">\n", indent, component.Name))
		} else {
			sb.WriteString(fmt.Sprintf("%s<div>\n", indent))
		}

		// Генерируем отображение для каждого состояния
		for _, state := range component.State {
			if state.Name == "count" {
				// Заголовок счетчика
				sb.WriteString(fmt.Sprintf("%s%s<h2>Счетчик: ", indent, indent))

				if state.Type == "number" {
					sb.WriteString("{ strconv.Itoa(count) }</h2>\n")
				} else if state.Type == "string" {
					sb.WriteString("{ count }</h2>\n")
				} else {
					sb.WriteString("{ fmt.Sprint(count) }</h2>\n")
				}

				// Кнопки
				sb.WriteString(fmt.Sprintf("%s%s<div>\n", indent, indent))

				// Кнопка уменьшения (-)
				if g.options.UseHtmx {
					sb.WriteString(fmt.Sprintf("%s%s%s<button hx-post=\"/api/%s/setCount?id={id}&value=-1\" "+
						"hx-target=\"#%s-{id}\" hx-swap=\"outerHTML\">-</button>\n",
						indent, indent, indent, strings.ToLower(component.Name), component.Name))
				} else {
					sb.WriteString(fmt.Sprintf("%s%s%s<button>-</button>\n", indent, indent, indent))
				}

				// Кнопка увеличения (+)
				if g.options.UseHtmx {
					sb.WriteString(fmt.Sprintf("%s%s%s<button hx-post=\"/api/%s/setCount?id={id}&value=1\" "+
						"hx-target=\"#%s-{id}\" hx-swap=\"outerHTML\">+</button>\n",
						indent, indent, indent, strings.ToLower(component.Name), component.Name))
				} else {
					sb.WriteString(fmt.Sprintf("%s%s%s<button>+</button>\n", indent, indent, indent))
				}

				sb.WriteString(fmt.Sprintf("%s%s</div>\n", indent, indent))
			} else {
				// Для других состояний - общий шаблон
				sb.WriteString(fmt.Sprintf("%s%s<h3>%s: ", indent, indent, state.Name))

				// Отображаем значение в зависимости от типа
				if state.Type == "number" {
					sb.WriteString(fmt.Sprintf("{ strconv.Itoa(%s) }</h3>\n", state.Name))
				} else if state.Type == "string" {
					sb.WriteString(fmt.Sprintf("{ %s }</h3>\n", state.Name))
				} else {
					sb.WriteString(fmt.Sprintf("{ fmt.Sprint(%s) }</h3>\n", state.Name))
				}
			}
		}

		// Закрываем внешний div
		sb.WriteString(fmt.Sprintf("%s</div>\n", indent))
	} else {
		// Если ни JSX, ни состояний нет, просто добавляем заглушку
		sb.WriteString(fmt.Sprintf("%s<div>Компонент без JSX</div>\n", g.getIndentation(1)))
	}

	sb.WriteString("}\n\n")

	return sb.String()
}

// simpleJSXToTempl - простая реализация конвертера JSX в templ
// Первый параметр - компонент, второй - JSX элемент, третий - уровень отступа
func (g *TemplGenerator) simpleJSXToTempl(component *models.ReactComponent, jsx *models.JSXElement, indent int) string {
	if jsx == nil {
		return ""
	}

	indentation := g.getIndentation(indent)
	var sb strings.Builder

	// Если это Fragment, преобразуем в <>...</>
	if jsx.Type == "Fragment" {
		sb.WriteString(indentation + "{<>}\n")
		for _, child := range jsx.Children {
			sb.WriteString(g.simpleJSXToTempl(component, child, indent+1))
		}
		sb.WriteString(indentation + "{</>}\n")
		return sb.String()
	}

	// Проверяем, является ли это HTML элементом или пользовательским компонентом
	isHTMLElement := len(jsx.Type) > 0 && jsx.Type[0] >= 'a' && jsx.Type[0] <= 'z'

	// Пользовательский компонент
	if !isHTMLElement && jsx.Type != "text" && jsx.Type != "expression" {
		sb.WriteString(indentation + "{\n")
		sb.WriteString(indentation + "\t// Вызов компонента " + jsx.Type + "\n")

		// Имя компонента в templ формате
		templComponentName := strings.ToLower(string(jsx.Type[0])) + jsx.Type[1:]

		// Вызов компонента
		sb.WriteString(indentation + "\t" + templComponentName + "(")

		// Параметры
		if len(jsx.Props) > 0 {
			sb.WriteString(jsx.Type + "Props{\n")
			for name, value := range jsx.Props {
				propName := strings.Title(name)
				sb.WriteString(indentation + "\t\t" + propName + ": ")

				// Форматируем значение в зависимости от его типа
				if str, ok := value.(string); ok {
					sb.WriteString(fmt.Sprintf("\"%s\"", str))
				} else if b, ok := value.(bool); ok {
					sb.WriteString(fmt.Sprintf("%v", b))
				} else {
					sb.WriteString("/* TODO: complex value */")
				}

				sb.WriteString(",\n")
			}
			sb.WriteString(indentation + "\t}")
		}

		// Если используем HTMX, добавляем id
		if g.options.UseHtmx {
			if len(jsx.Props) > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("id")
		}

		sb.WriteString(")\n")
		sb.WriteString(indentation + "}\n")
		return sb.String()
	}

	// Текстовый узел
	if jsx.Type == "text" {
		if content, ok := jsx.Props["content"].(string); ok {
			sb.WriteString(indentation + content + "\n")
		}
		return sb.String()
	}

	// Выражение
	if jsx.Type == "expression" {
		if content, ok := jsx.Props["content"].(string); ok {
			sb.WriteString(indentation + "{ " + content + " }\n")
		}
		return sb.String()
	}

	// HTML элемент
	sb.WriteString(indentation + "<" + jsx.Type)

	// Атрибуты
	for name, value := range jsx.Props {
		// Преобразуем camelCase в kebab-case для HTML атрибутов
		attrName := g.camelCaseToKebabCase(name)

		// Специальные атрибуты
		if name == "className" {
			attrName = "class"
		} else if name == "htmlFor" {
			attrName = "for"
		}

		// Запись атрибута
		if value == true {
			sb.WriteString(" " + attrName)
		} else if valueStr, ok := value.(string); ok {
			sb.WriteString(" " + attrName + "=\"" + valueStr + "\"")
		} else {
			// Сложные значения
			sb.WriteString(" " + attrName + "=\"/* TODO: complex value */\"")
		}
	}

	// HTMX атрибуты для корневого элемента
	if g.options.UseHtmx && indent == 1 {
		sb.WriteString(fmt.Sprintf(" id=\"%s-{id}\"", component.Name))
	}

	// Закрытие тега и дочерние элементы
	if len(jsx.Children) == 0 {
		sb.WriteString(">\n")
		sb.WriteString(indentation + "</" + jsx.Type + ">\n")
	} else {
		sb.WriteString(">\n")

		for _, child := range jsx.Children {
			sb.WriteString(g.simpleJSXToTempl(component, child, indent+1))
		}

		sb.WriteString(indentation + "</" + jsx.Type + ">\n")
	}

	return sb.String()
}

// generateHelperFunctions генерирует вспомогательные функции для templ
func (g *TemplGenerator) generateHelperFunctions(component *models.ReactComponent) string {
	var sb strings.Builder

	// Здесь можно добавить другие вспомогательные функции, если они нужны

	return sb.String()
}

// detectRequiredImports определяет необходимые импорты на основе компонента
func (g *TemplGenerator) detectRequiredImports(component *models.ReactComponent) []string {
	imports := make(map[string]bool)

	// Всегда импортируем fmt для интерполяции строк и форматирования
	imports["fmt"] = true

	// Если используем HTMX или есть числовые состояния, импортируем strconv
	if g.options.UseHtmx || hasNumericState(component) {
		imports["strconv"] = true
	}

	// Проверка JSX на необходимость дополнительных импортов
	g.detectImportsFromJSX(component.JSX, imports)

	// Проверка эффектов, состояний и т.д.
	g.detectImportsFromState(component, imports)

	// Преобразуем map в slice
	result := make([]string, 0, len(imports))
	for imp := range imports {
		result = append(result, imp)
	}

	return result
}

// hasNumericState проверяет, есть ли в компоненте числовые состояния
func hasNumericState(component *models.ReactComponent) bool {
	for _, state := range component.State {
		if state.Type == "number" {
			return true
		}
		if state.InitialValue != nil {
			_, isNumeric := state.InitialValue.(float64)
			if isNumeric {
				return true
			}
		}
	}
	return false
}

// detectImportsFromJSX проверяет JSX на необходимость дополнительных импортов
func (g *TemplGenerator) detectImportsFromJSX(jsx *models.JSXElement, imports map[string]bool) {
	if jsx == nil {
		return
	}

	// Проверяем текущий элемент
	for _, value := range jsx.Props {
		if expr, ok := value.(map[string]interface{}); ok {
			if code, ok := expr["code"].(string); ok {
				// Проверяем выражения в атрибутах
				g.detectImportsFromExpression(code, imports)
			}
		}
	}

	// Рекурсивно проверяем дочерние элементы
	for _, child := range jsx.Children {
		g.detectImportsFromJSX(child, imports)
	}
}

// detectImportsFromState проверяет состояния на необходимость дополнительных импортов
func (g *TemplGenerator) detectImportsFromState(component *models.ReactComponent, imports map[string]bool) {
	// Для работы с числами
	for _, state := range component.State {
		if state.Type == "number" || (state.InitialValue != nil && isNumeric(state.InitialValue)) {
			imports["strconv"] = true
			break
		}
	}

	// Другие проверки для эффектов, колбэков и т.д.
}

// detectImportsFromExpression проверяет выражения на необходимость дополнительных импортов
func (g *TemplGenerator) detectImportsFromExpression(expr string, imports map[string]bool) {
	// Шаблонные строки требуют fmt
	if strings.Contains(expr, "`") {
		imports["fmt"] = true
	}

	// Числовые операции могут требовать strconv
	if strings.Contains(expr, ".toString()") ||
		strings.Contains(expr, ".toFixed(") ||
		strings.Contains(expr, "parseInt(") ||
		strings.Contains(expr, "parseFloat(") {
		imports["strconv"] = true
	}

	// Работа с датами может требовать time
	if strings.Contains(expr, "new Date(") ||
		strings.Contains(expr, ".getTime(") ||
		strings.Contains(expr, ".getDate(") {
		imports["time"] = true
	}

	// Регулярные выражения
	if strings.Contains(expr, "RegExp(") ||
		strings.Contains(expr, "match(") ||
		strings.Contains(expr, "test(") ||
		strings.Contains(expr, "replace(") {
		imports["regexp"] = true
	}
}

// needsHelperFunctions проверяет, нужны ли вспомогательные функции для компонента
func (g *TemplGenerator) needsHelperFunctions(component *models.ReactComponent) bool {
	// В будущем здесь могут быть дополнительные проверки
	return false
}

// Вспомогательные функции

// convertTypeToGo преобразует тип TypeScript в тип Go
func (g *TemplGenerator) convertTypeToGo(tsType string) string {
	switch tsType {
	case "string":
		return "string"
	case "number":
		return "int"
	case "boolean":
		return "bool"
	case "any":
		return "interface{}"
	case "array":
		return "[]interface{}"
	case "object":
		return "map[string]interface{}"
	default:
		if strings.HasPrefix(tsType, "Array<") {
			elementType := tsType[6 : len(tsType)-1]
			goElementType := g.convertTypeToGo(elementType)
			return "[]" + goElementType
		}
		return "interface{}"
	}
}

// getIndentation возвращает строку с отступом заданного уровня
func (g *TemplGenerator) getIndentation(level int) string {
	if g.indentStyle == "tabs" {
		return strings.Repeat("\t", level)
	}
	return strings.Repeat(" ", g.indentSize*level)
}

// camelCaseToKebabCase преобразует camelCase в kebab-case
func (g *TemplGenerator) camelCaseToKebabCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('-')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// isNumeric проверяет, является ли значение числовым
func isNumeric(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	default:
		return false
	}
}
