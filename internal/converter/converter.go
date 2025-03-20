package converter

import (
	"fmt"
	"react-to-templ-converter/internal/config"
	"react-to-templ-converter/internal/models"
	"react-to-templ-converter/internal/parser"
	"strings"
)

// Converter определяет интерфейс для конвертации React компонентов
type Converter interface {
	// Convert преобразует React код в templ шаблоны и Go код
	Convert(reactCode string, options *config.ConversionOptions) (*models.ConversionResult, error)
}

// ReactToTemplConverter реализует интерфейс Converter для преобразования React компонентов в templ
type ReactToTemplConverter struct {
	parser         parser.ReactParser
	debug          bool
	indentStyle    string
	indentSize     int
	jsxConverter   *JSXToHTMXConverter
	stateHandler   *StateHandler
	templGenerator TemplGenerator
	goGenerator    GoGenerator
}

// TemplGenerator определяет интерфейс для генерации templ шаблонов
type TemplGenerator interface {
	GenerateTemplFile(component *models.ReactComponent) string
	SetDebug(debug bool)
	SetIndentation(style string, size int)
}

// GoGenerator определяет интерфейс для генерации Go контроллеров
type GoGenerator interface {
	GenerateGoController(component *models.ReactComponent) string
	GenerateJavaScript(component *models.ReactComponent) string
	SetDebug(debug bool)
	SetIndentation(style string, size int)
}

// Convert преобразует React код в templ шаблоны и Go код
func (c *ReactToTemplConverter) Convert(reactCode string, options *config.ConversionOptions) (*models.ConversionResult, error) {
	// Парсинг React компонента
	component, err := c.parser.ParseComponent(reactCode)
	if err != nil {
		return nil, err
	}

	// Если имя компонента не указано, используем имя из опций
	if component.Name == "" {
		if options.ComponentName != "" {
			component.Name = options.ComponentName
		} else {
			// Пытаемся определить имя компонента из exports
			if component.Exports != nil {
				if defaultExport, ok := component.Exports["default"].(string); ok && defaultExport != "" {
					component.Name = defaultExport
				} else if len(component.Exports) > 0 {
					// Берем первый экспорт
					for name := range component.Exports {
						if name != "default" {
							component.Name = name
							break
						}
					}
				}
			}

			// Если имя все еще не определено, используем "Component"
			if component.Name == "" {
				component.Name = "Counter" // По умолчанию для компонента счетчика
			}
		}
	}

	// Создаем конвертеры и генераторы с указанными опциями
	c.jsxConverter = NewJSXToHTMXConverter(options)
	c.stateHandler = NewStateHandler(options)

	// Применяем настройки отступов и режима отладки
	if c.debug {
		c.jsxConverter.SetDebug(c.debug)
		c.stateHandler.SetDebug(c.debug)

		if c.templGenerator != nil {
			c.templGenerator.SetDebug(c.debug)
		}

		if c.goGenerator != nil {
			c.goGenerator.SetDebug(c.debug)
		}
	}

	if c.indentStyle != "" {
		c.stateHandler.SetIndentation(c.indentStyle, c.indentSize)

		if c.templGenerator != nil {
			c.templGenerator.SetIndentation(c.indentStyle, c.indentSize)
		}

		if c.goGenerator != nil {
			c.goGenerator.SetIndentation(c.indentStyle, c.indentSize)
		}
	}

	// Генерация templ шаблона
	var templCode string
	if c.templGenerator != nil {
		templCode = c.templGenerator.GenerateTemplFile(component)
	} else {
		// Простая генерация templ, если генератор не установлен
		templCode = c.generateBasicTempl(component, options)
	}

	// Генерация Go контроллера
	var goController string
	if c.goGenerator != nil {
		goController = c.goGenerator.GenerateGoController(component)
	} else if options.UseHtmx && (len(component.State) > 0 || len(component.Effects) > 0) {
		// Генерация контроллеров для состояний, если они есть и используется HTMX
		goController = c.stateHandler.GenerateStateHandlers(component)
	}

	// Генерация JavaScript для HTMX
	var htmxJS string
	if c.goGenerator != nil {
		htmxJS = c.goGenerator.GenerateJavaScript(component)
	} else if options.UseHtmx {
		// Генерация JavaScript для HTMX, если используется HTMX
		htmxJS = c.stateHandler.GenerateHtmxJSHelpers(component)
	}

	// Формирование результата
	result := models.NewConversionResult(options.ComponentName, "")
	result.TemplFile = templCode
	result.GoController = goController
	result.HtmxJS = htmxJS

	// Сохраняем настройки конвертации в результате
	result.Settings = map[string]interface{}{
		"useHtmx":          options.UseHtmx,
		"packageName":      options.PackageName,
		"statePersistence": options.StatePersistence,
	}

	return result, nil
}

// generateBasicTempl создает простой templ шаблон, если генератор не установлен
func (c *ReactToTemplConverter) generateBasicTempl(component *models.ReactComponent, options *config.ConversionOptions) string {
	// Базовая реализация для генерации templ шаблона без использования внешнего генератора
	// В реальном приложении здесь будет более сложная логика

	var sb strings.Builder

	// Пакет
	packageName := "templates"
	if options.PackageName != "" && options.PackageName != "." {
		packageName = options.PackageName
	}
	sb.WriteString(fmt.Sprintf("package %s\n\n", packageName))

	// Импорты
	sb.WriteString("import (\n")
	sb.WriteString("\t\"fmt\"\n")
	if options.UseHtmx {
		sb.WriteString("\t\"strconv\"\n")
	}
	sb.WriteString(")\n\n")

	// Структура пропсов если есть
	if len(component.Props) > 0 {
		sb.WriteString(fmt.Sprintf("// %sProps определяет пропсы для компонента\n", component.Name))
		sb.WriteString(fmt.Sprintf("type %sProps struct {\n", component.Name))
		for _, prop := range component.Props {
			sb.WriteString(fmt.Sprintf("\t%s %s\n", strings.Title(prop.Name), c.convertTypeToGo(prop.Type)))
		}
		sb.WriteString("}\n\n")
	}

	// Определение templ компонента
	funcName := strings.ToLower(string(component.Name[0])) + component.Name[1:]

	params := ""
	if len(component.Props) > 0 {
		params = fmt.Sprintf("props %sProps", component.Name)
	}

	if options.UseHtmx {
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

	sb.WriteString(fmt.Sprintf("templ %s(%s) {\n", funcName, params))

	// Если есть JSX, конвертируем его
	if component.JSX != nil {
		jsxTemplate := c.jsxConverter.ConvertJSXToTempl(component.JSX, 1)
		sb.WriteString(jsxTemplate)
	} else if len(component.State) > 0 {
		// Базовый шаблон для состояний (для счетчика)
		indent := c.getIndentation(1)

		// Внешний div с ID для HTMX (если используется)
		if options.UseHtmx {
			sb.WriteString(fmt.Sprintf("%s<div id=\"%s-{id}\">\n", indent, component.Name))
		} else {
			sb.WriteString(fmt.Sprintf("%s<div>\n", indent))
		}

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
				if options.UseHtmx {
					sb.WriteString(fmt.Sprintf("%s%s%s<button hx-post=\"/api/%s/setCount?id={id}&value=-1\" "+
						"hx-target=\"#%s-{id}\" hx-swap=\"outerHTML\">-</button>\n",
						indent, indent, indent, strings.ToLower(component.Name), component.Name))
				} else {
					sb.WriteString(fmt.Sprintf("%s%s%s<button>-</button>\n", indent, indent, indent))
				}

				// Кнопка увеличения (+)
				if options.UseHtmx {
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
		// Заглушка, если JSX нет
		sb.WriteString("\t<div>Компонент без JSX</div>\n")
	}

	sb.WriteString("}\n")

	return sb.String()
}

// convertTypeToGo преобразует тип TypeScript в Go
func (c *ReactToTemplConverter) convertTypeToGo(tsType string) string {
	switch tsType {
	case "string":
		return "string"
	case "number":
		return "int"
	case "boolean":
		return "bool"
	default:
		return "interface{}"
	}
}

// SetParser устанавливает парсер для конвертера
func (c *ReactToTemplConverter) SetParser(parser parser.ReactParser) {
	c.parser = parser
}

// SetTemplGenerator устанавливает генератор templ шаблонов
func (c *ReactToTemplConverter) SetTemplGenerator(generator TemplGenerator) {
	c.templGenerator = generator
}

// SetGoGenerator устанавливает генератор Go контроллеров
func (c *ReactToTemplConverter) SetGoGenerator(generator GoGenerator) {
	c.goGenerator = generator
}

// SetDebug устанавливает режим отладки
func (c *ReactToTemplConverter) SetDebug(debug bool) {
	c.debug = debug
}

// SetIndentation устанавливает стиль отступов для генерируемого кода
func (c *ReactToTemplConverter) SetIndentation(style string, size int) {
	c.indentStyle = style
	c.indentSize = size
}

// getIndentation возвращает строку с отступом заданного уровня
func (c *ReactToTemplConverter) getIndentation(level int) string {
	if c.indentStyle == "tabs" {
		return strings.Repeat("\t", level)
	}
	return strings.Repeat(" ", c.indentSize*level)
}

// ConverterOption определяет опцию конфигурации для конвертера
type ConverterOption func(converter Converter)

// WithTemplGenerator устанавливает генератор templ шаблонов
func WithTemplGenerator(generator TemplGenerator) ConverterOption {
	return func(converter Converter) {
		if generatorAware, ok := converter.(interface{ SetTemplGenerator(TemplGenerator) }); ok {
			generatorAware.SetTemplGenerator(generator)
		}
	}
}

// WithGoGenerator устанавливает генератор Go контроллеров
func WithGoGenerator(generator GoGenerator) ConverterOption {
	return func(converter Converter) {
		if generatorAware, ok := converter.(interface{ SetGoGenerator(GoGenerator) }); ok {
			generatorAware.SetGoGenerator(generator)
		}
	}
}

// WithDebugMode включает режим отладки для конвертера
func WithDebugMode(debug bool) ConverterOption {
	return func(converter Converter) {
		if debugConverter, ok := converter.(interface{ SetDebug(bool) }); ok {
			debugConverter.SetDebug(debug)
		}
	}
}

// WithIndentation устанавливает стиль отступов для генерируемого кода
func WithIndentation(style string, size int) ConverterOption {
	return func(converter Converter) {
		if indentConverter, ok := converter.(interface{ SetIndentation(string, int) }); ok {
			indentConverter.SetIndentation(style, size)
		}
	}
}

// NewConverter создает новый экземпляр конвертера с заданными опциями
func NewConverter(parser parser.ReactParser, options ...ConverterOption) Converter {
	converter := &ReactToTemplConverter{
		parser:      parser,
		indentStyle: "spaces",
		indentSize:  4,
	}

	for _, option := range options {
		option(converter)
	}

	return converter
}
