package generator

import (
	"fmt"
	"react-to-templ-converter/internal/config"
	"react-to-templ-converter/internal/models"
	"strings"
)

// GoGenerator генерирует Go контроллеры для работы с templ шаблонами
type GoGenerator struct {
	options      *config.ConversionOptions
	stateHandler StateHandler
	debug        bool
	indentSize   int
	indentStyle  string
}

// StateHandler определяет интерфейс для обработки состояний React
type StateHandler interface {
	GenerateStateStructs(component *models.ReactComponent) string
	GenerateStateHandlers(component *models.ReactComponent) string
	GenerateHtmxJSHelpers(component *models.ReactComponent) string
	SetDebug(debug bool)
	SetIndentation(style string, size int)
}

// NewGoGenerator создает новый генератор Go контроллеров
func NewGoGenerator(options *config.ConversionOptions) *GoGenerator {
	return &GoGenerator{
		options:     options,
		debug:       options.Debug,
		indentSize:  options.Indentation.Size,
		indentStyle: options.Indentation.Style,
	}
}

// SetStateHandler устанавливает обработчик состояний
func (g *GoGenerator) SetStateHandler(handler StateHandler) {
	g.stateHandler = handler
}

// SetDebug устанавливает режим отладки
func (g *GoGenerator) SetDebug(debug bool) {
	g.debug = debug
	if g.stateHandler != nil {
		g.stateHandler.SetDebug(debug)
	}
}

// SetIndentation устанавливает стиль отступов
func (g *GoGenerator) SetIndentation(style string, size int) {
	g.indentStyle = style
	g.indentSize = size
	if g.stateHandler != nil {
		g.stateHandler.SetIndentation(style, size)
	}
}

// GenerateGoController создает Go контроллер для React компонента
func (g *GoGenerator) GenerateGoController(component *models.ReactComponent) string {
	// Если не используем HTMX, контроллер не нужен
	if !g.options.UseHtmx {
		return ""
	}

	// Если нет состояний и эффектов, контроллер тоже не нужен
	if len(component.State) == 0 && len(component.Effects) == 0 && len(component.Callbacks) == 0 {
		return ""
	}

	var sb strings.Builder

	// 1. Генерация заголовка файла
	sb.WriteString(g.generateFileHeader(component))

	// 2. Генерация структур для состояний
	if g.stateHandler != nil {
		sb.WriteString(g.stateHandler.GenerateStateStructs(component))
	} else {
		sb.WriteString(g.generateBasicStateStructs(component))
	}

	// 3. Генерация обработчиков для состояний и эффектов
	if g.stateHandler != nil {
		sb.WriteString(g.stateHandler.GenerateStateHandlers(component))
	} else {
		sb.WriteString(g.generateBasicStateHandlers(component))
	}

	// 4. Генерация вспомогательных функций
	sb.WriteString(g.generateUtilityFunctions(component))

	return sb.String()
}

// GenerateJavaScript создает JavaScript код для поддержки HTMX
func (g *GoGenerator) GenerateJavaScript(component *models.ReactComponent) string {
	if !g.options.UseHtmx {
		return ""
	}

	if g.stateHandler != nil {
		return g.stateHandler.GenerateHtmxJSHelpers(component)
	}

	return g.generateBasicHtmxJS(component)
}

// generateFileHeader генерирует заголовок файла с пакетом и импортами
func (g *GoGenerator) generateFileHeader(component *models.ReactComponent) string {
	var sb strings.Builder

	// Пакет
	sb.WriteString("package controllers\n\n")

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

// generateBasicStateStructs генерирует базовую структуру состояний компонента
func (g *GoGenerator) generateBasicStateStructs(component *models.ReactComponent) string {
	if len(component.State) == 0 {
		return ""
	}

	var sb strings.Builder

	// Определение структуры состояния
	sb.WriteString(fmt.Sprintf("// %sState определяет состояние компонента %s\n", component.Name, component.Name))
	sb.WriteString(fmt.Sprintf("type %sState struct {\n", component.Name))

	indent := g.getIndentation(1)

	// Добавляем поля для состояний
	for _, state := range component.State {
		stateName := strings.Title(state.Name)
		goType := g.convertTypeToGo(state.Type, state.InitialValue)

		sb.WriteString(fmt.Sprintf("%s%s %s\n", indent, stateName, goType))
	}

	sb.WriteString("}\n\n")

	// Глобальные переменные для хранения состояний
	sb.WriteString("var (\n")
	sb.WriteString(fmt.Sprintf("%s%sStates = make(map[string]*%sState)\n", indent, strings.ToLower(component.Name), component.Name))
	sb.WriteString(fmt.Sprintf("%s%sMutex sync.RWMutex\n", indent, strings.ToLower(component.Name)))
	sb.WriteString(")\n\n")

	return sb.String()
}

// generateBasicStateHandlers генерирует базовые обработчики для управления состоянием
func (g *GoGenerator) generateBasicStateHandlers(component *models.ReactComponent) string {
	if len(component.State) == 0 {
		return ""
	}

	var sb strings.Builder
	indent := g.getIndentation(1)

	// Функция для создания нового компонента
	sb.WriteString(fmt.Sprintf("// New%s создает новый экземпляр компонента\n", component.Name))
	sb.WriteString(fmt.Sprintf("func New%s(w http.ResponseWriter, r *http.Request) {\n", component.Name))
	sb.WriteString(fmt.Sprintf("%sid := uuid.New().String()\n\n", indent))

	// Создание начального состояния
	sb.WriteString(fmt.Sprintf("%sstate := &%sState{\n", indent, component.Name))

	for _, state := range component.State {
		stateName := strings.Title(state.Name)

		// Начальное значение
		if state.InitialValue != nil {
			sb.WriteString(fmt.Sprintf("%s%s%s: %v,\n", indent, indent, stateName, g.formatGoValue(state.InitialValue)))
		} else {
			defaultValue := g.getDefaultValueForType(g.convertTypeToGo(state.Type, nil))
			sb.WriteString(fmt.Sprintf("%s%s%s: %s,\n", indent, indent, stateName, defaultValue))
		}
	}

	sb.WriteString(fmt.Sprintf("%s}\n\n", indent))

	// Сохраняем состояние
	sb.WriteString(fmt.Sprintf("%s%sMutex.Lock()\n", indent, strings.ToLower(component.Name)))
	sb.WriteString(fmt.Sprintf("%s%sStates[id] = state\n", indent, strings.ToLower(component.Name)))
	sb.WriteString(fmt.Sprintf("%s%sMutex.Unlock()\n\n", indent, strings.ToLower(component.Name)))

	// Рендерим компонент
	packagePrefix := ""
	if g.options.PackageName != "" && g.options.PackageName != "." {
		packagePrefix = g.options.PackageName + "."
	}

	sb.WriteString(fmt.Sprintf("%stempl.Handler(%s%s(", indent, packagePrefix,
		strings.ToLower(string(component.Name[0]))+component.Name[1:]))

	// Передаем пропсы (упрощенно)
	if len(component.Props) > 0 {
		sb.WriteString(fmt.Sprintf("%sProps{}, ", component.Name))
	}

	sb.WriteString("id)).ServeHTTP(w, r)\n")
	sb.WriteString("}\n\n")

	// Создание базовых обработчиков для каждого состояния
	for _, state := range component.State {
		g.generateBasicStateUpdater(&sb, component, state)
	}

	return sb.String()
}

// generateBasicStateUpdater генерирует базовый обработчик для обновления состояния
func (g *GoGenerator) generateBasicStateUpdater(sb *strings.Builder, component *models.ReactComponent, state models.StateDefinition) {
	stateName := strings.Title(state.Name)
	setterName := state.Setter

	// Если setter не начинается с большой буквы, делаем первую букву заглавной
	if len(setterName) > 0 && setterName[0] >= 'a' && setterName[0] <= 'z' {
		setterName = strings.Title(setterName)
	}

	indent := g.getIndentation(1)

	sb.WriteString(fmt.Sprintf("// %s обрабатывает изменение состояния %s\n", setterName, state.Name))
	sb.WriteString(fmt.Sprintf("func %s(w http.ResponseWriter, r *http.Request) {\n", setterName))

	// Получаем ID компонента
	sb.WriteString(fmt.Sprintf("%sid := r.URL.Query().Get(\"id\")\n", indent))
	sb.WriteString(fmt.Sprintf("%sif id == \"\" {\n", indent))
	sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"ID компонента не указан\", http.StatusBadRequest)\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s}\n\n", indent))

	// Получаем состояние
	sb.WriteString(fmt.Sprintf("%s%sMutex.RLock()\n", indent, strings.ToLower(component.Name)))
	sb.WriteString(fmt.Sprintf("%sstate, ok := %sStates[id]\n", indent, strings.ToLower(component.Name)))
	sb.WriteString(fmt.Sprintf("%s%sMutex.RUnlock()\n", indent, strings.ToLower(component.Name)))
	sb.WriteString(fmt.Sprintf("%sif !ok {\n", indent))
	sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"Состояние не найдено\", http.StatusNotFound)\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s}\n\n", indent))

	// Получаем новое значение из запроса
	goType := g.convertTypeToGo(state.Type, state.InitialValue)

	sb.WriteString(fmt.Sprintf("%s// Получаем новое значение\n", indent))

	if goType == "string" {
		sb.WriteString(fmt.Sprintf("%snewValue := r.FormValue(\"value\")\n", indent))
	} else if goType == "int" {
		sb.WriteString(fmt.Sprintf("%snewValueStr := r.FormValue(\"value\")\n", indent))
		sb.WriteString(fmt.Sprintf("%snewValue, err := strconv.Atoi(newValueStr)\n", indent))
		sb.WriteString(fmt.Sprintf("%sif err != nil {\n", indent))
		sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"Неверный формат значения\", http.StatusBadRequest)\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s}\n", indent))
	} else if goType == "bool" {
		sb.WriteString(fmt.Sprintf("%snewValue := r.FormValue(\"value\") == \"true\"\n", indent))
	} else {
		sb.WriteString(fmt.Sprintf("%s// Для сложных типов используем JSON\n", indent))
		sb.WriteString(fmt.Sprintf("%svar newValue %s\n", indent, goType))
		sb.WriteString(fmt.Sprintf("%sif err := json.NewDecoder(r.Body).Decode(&newValue); err != nil {\n", indent))
		sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"Ошибка декодирования: \" + err.Error(), http.StatusBadRequest)\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s}\n", indent))
	}

	// Обновляем состояние
	sb.WriteString(fmt.Sprintf("\n%s// Обновляем состояние\n", indent))
	sb.WriteString(fmt.Sprintf("%s%sMutex.Lock()\n", indent, strings.ToLower(component.Name)))
	sb.WriteString(fmt.Sprintf("%sstate.%s = newValue\n", indent, stateName))
	sb.WriteString(fmt.Sprintf("%s%sMutex.Unlock()\n\n", indent, strings.ToLower(component.Name)))

	// Рендерим компонент с обновленным состоянием
	packagePrefix := ""
	if g.options.PackageName != "" && g.options.PackageName != "." {
		packagePrefix = g.options.PackageName + "."
	}

	sb.WriteString(fmt.Sprintf("%s// Рендерим компонент с обновленным состоянием\n", indent))
	sb.WriteString(fmt.Sprintf("%stempl.Handler(%s%s(", indent, packagePrefix,
		strings.ToLower(string(component.Name[0]))+component.Name[1:]))

	// Передаем пропсы (упрощенно)
	if len(component.Props) > 0 {
		sb.WriteString(fmt.Sprintf("%sProps{}, ", component.Name))
	}

	sb.WriteString("id)).ServeHTTP(w, r)\n")
	sb.WriteString("}\n\n")
}

// generateBasicHtmxJS генерирует базовый JavaScript для HTMX
func (g *GoGenerator) generateBasicHtmxJS(component *models.ReactComponent) string {
	if len(component.State) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("// JavaScript для поддержки HTMX функциональности компонента " + component.Name + "\n\n")

	sb.WriteString("document.addEventListener('DOMContentLoaded', function() {\n")
	sb.WriteString("    // Инициализация HTMX обработчиков\n")
	sb.WriteString("    document.body.addEventListener('htmx:afterSwap', function(event) {\n")
	sb.WriteString("        const target = event.detail.target;\n")
	sb.WriteString(fmt.Sprintf("        if (target && target.id && target.id.startsWith('%s-')) {\n", component.Name))
	sb.WriteString("            // Компонент обновлен - можно выполнить дополнительные действия\n")
	sb.WriteString("        }\n")
	sb.WriteString("    });\n")
	sb.WriteString("});\n")

	return sb.String()
}

// generateUtilityFunctions генерирует вспомогательные функции для контроллера
func (g *GoGenerator) generateUtilityFunctions(component *models.ReactComponent) string {
	var sb strings.Builder

	// Функция очистки ресурсов
	sb.WriteString("// CleanupResources освобождает ресурсы компонента\n")
	sb.WriteString(fmt.Sprintf("func Cleanup%s(w http.ResponseWriter, r *http.Request) {\n", component.Name))

	indent := g.getIndentation(1)

	// Получаем ID компонента
	sb.WriteString(fmt.Sprintf("%sid := r.URL.Query().Get(\"id\")\n", indent))
	sb.WriteString(fmt.Sprintf("%sif id == \"\" {\n", indent))
	sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"ID компонента не указан\", http.StatusBadRequest)\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s}\n\n", indent))

	// Удаляем состояние
	sb.WriteString(fmt.Sprintf("%s// Удаляем состояние компонента\n", indent))
	sb.WriteString(fmt.Sprintf("%s%sMutex.Lock()\n", indent, strings.ToLower(component.Name)))
	sb.WriteString(fmt.Sprintf("%sdelete(%sStates, id)\n", indent, strings.ToLower(component.Name)))
	sb.WriteString(fmt.Sprintf("%s%sMutex.Unlock()\n\n", indent, strings.ToLower(component.Name)))

	// Возвращаем успешный статус
	sb.WriteString(fmt.Sprintf("%s// Возвращаем успешный статус\n", indent))
	sb.WriteString(fmt.Sprintf("%sw.WriteHeader(http.StatusOK)\n", indent))
	sb.WriteString("}\n")

	return sb.String()
}

// detectRequiredImports определяет необходимые импорты для контроллера
func (g *GoGenerator) detectRequiredImports(component *models.ReactComponent) []string {
	imports := make(map[string]bool)

	// Стандартные импорты
	imports["net/http"] = true
	imports["sync"] = true

	// Если используется templ
	templatePackage := "templates"
	if g.options.PackageName != "" && g.options.PackageName != "." {
		templatePackage = g.options.PackageName
	}

	imports["github.com/a-h/templ"] = true
	imports[fmt.Sprintf("react-to-templ/%s", templatePackage)] = true

	// Для работы с JSON
	if len(component.Props) > 0 || g.options.StatePersistence == "redis" {
		imports["encoding/json"] = true
	}

	// Для генерации ID
	imports["github.com/google/uuid"] = true

	// Для преобразования типов
	for _, state := range component.State {
		if state.Type == "number" || (state.InitialValue != nil && isNumeric(state.InitialValue)) {
			imports["strconv"] = true
			break
		}
	}

	// Преобразуем map в slice
	result := make([]string, 0, len(imports))
	for imp := range imports {
		result = append(result, imp)
	}

	return result
}

// Вспомогательные функции

// convertTypeToGo преобразует тип TypeScript/JavaScript в тип Go
func (g *GoGenerator) convertTypeToGo(tsType string, value interface{}) string {
	if tsType == "" && value != nil {
		// Определяем тип на основе значения
		return g.inferGoTypeFromValue(value)
	}

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
			goElementType := g.convertTypeToGo(elementType, nil)
			return "[]" + goElementType
		}
		return "interface{}"
	}
}

// inferGoTypeFromValue определяет тип Go на основе значения
func (g *GoGenerator) inferGoTypeFromValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return "string"
	case float64:
		return "float64"
	case float32:
		return "float32"
	case int, int32, int64:
		return "int"
	case bool:
		return "bool"
	case []interface{}:
		return "[]interface{}"
	case map[string]interface{}:
		return "map[string]interface{}"
	default:
		if v == nil {
			return "interface{}"
		}

		// Проверяем строковое представление для других типов
		valueStr := fmt.Sprintf("%v", v)
		if valueStr == "[]" {
			return "[]interface{}"
		} else if valueStr == "{}" {
			return "map[string]interface{}"
		}

		return "interface{}"
	}
}

// formatGoValue форматирует значение для использования в Go коде
func (g *GoGenerator) formatGoValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case bool, int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		if v == nil {
			return "nil"
		}

		// Проверяем строковое представление для других типов
		valueStr := fmt.Sprintf("%v", v)
		if valueStr == "[]" {
			return "[]interface{}{}"
		} else if valueStr == "{}" {
			return "map[string]interface{}{}"
		}

		return "nil"
	}
}

// getDefaultValueForType возвращает значение по умолчанию для типа Go
func (g *GoGenerator) getDefaultValueForType(goType string) string {
	switch goType {
	case "string":
		return "\"\""
	case "int", "int32", "int64", "float32", "float64":
		return "0"
	case "bool":
		return "false"
	case "[]interface{}":
		return "[]interface{}{}"
	case "map[string]interface{}":
		return "map[string]interface{}{}"
	default:
		return "nil"
	}
}

// getIndentation возвращает строку с отступом заданного уровня
func (g *GoGenerator) getIndentation(level int) string {
	if g.indentStyle == "tabs" {
		return strings.Repeat("\t", level)
	}
	return strings.Repeat(" ", g.indentSize*level)
}

// isNumeric проверяет, является ли значение числовым
//func isNumeric(value interface{}) bool {
//	switch value.(type) {
//	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
//		return true
//	default:
//		return false
//	}
//}
