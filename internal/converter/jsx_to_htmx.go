package converter

import (
	"fmt"
	"react-to-templ-converter/internal/config"
	"react-to-templ-converter/internal/models"
	"regexp"
	"strings"
)

// JSXToHTMXConverter преобразует JSX элементы в HTML с атрибутами HTMX
type JSXToHTMXConverter struct {
	options *config.ConversionOptions
	indent  int
	debug   bool
}

// NewJSXToHTMXConverter создает новый конвертер JSX в HTMX
func NewJSXToHTMXConverter(options *config.ConversionOptions) *JSXToHTMXConverter {
	return &JSXToHTMXConverter{
		options: options,
		indent:  0,
		debug:   options.Debug,
	}
}

// SetDebug устанавливает режим отладки
func (c *JSXToHTMXConverter) SetDebug(debug bool) {
	c.debug = debug
}

// ConvertJSXToTempl преобразует JSX дерево в код templ
func (c *JSXToHTMXConverter) ConvertJSXToTempl(jsx *models.JSXElement, indent int) string {
	if jsx == nil {
		return ""
	}

	c.indent = indent
	indentation := strings.Repeat("\t", indent)

	var sb strings.Builder

	// Fragment превращается в templ {<>...</>}
	if jsx.Type == "Fragment" {
		sb.WriteString(indentation + "{<>}\n")

		// Обрабатываем дочерние элементы
		for _, child := range jsx.Children {
			childHTML := c.ConvertJSXToTempl(child, indent+1)
			sb.WriteString(childHTML)
		}

		sb.WriteString(indentation + "{</>}\n")
		return sb.String()
	}

	// Проверяем, является ли тег HTML элементом (начинается с маленькой буквы)
	isHTMLElement := len(jsx.Type) > 0 && jsx.Type[0] >= 'a' && jsx.Type[0] <= 'z'

	// Если это пользовательский компонент (начинается с большой буквы), конвертируем в вызов templ
	if !isHTMLElement && jsx.Type != "text" && jsx.Type != "expression" {
		return c.convertComponent(jsx, indentation)
	}

	// Если это текстовый узел
	if jsx.Type == "text" {
		if content, ok := jsx.Props["content"].(string); ok && content != "" {
			sb.WriteString(indentation + content + "\n")
		}
		return sb.String()
	}

	// Если это выражение
	if jsx.Type == "expression" {
		if content, ok := jsx.Props["content"].(string); ok && content != "" {
			// Преобразуем React выражение в Go
			goExpr := c.convertReactExpressionToGo(content)
			sb.WriteString(indentation + "{ " + goExpr + " }\n")
		}
		return sb.String()
	}

	// Открывающий тег HTML элемента
	sb.WriteString(indentation + "<" + jsx.Type)

	// Атрибуты
	sb.WriteString(c.convertElementAttributes(jsx))

	// Если нет дочерних элементов, закрываем тег сразу
	if len(jsx.Children) == 0 {
		// Для самозакрывающихся тегов
		if isVoidElement(jsx.Type) {
			sb.WriteString(" />\n")
			return sb.String()
		}

		sb.WriteString(">\n")
		sb.WriteString(indentation + "</" + jsx.Type + ">\n")
		return sb.String()
	}

	sb.WriteString(">\n")

	// Обрабатываем дочерние элементы
	for _, child := range jsx.Children {
		childHTML := c.ConvertJSXToTempl(child, indent+1)
		sb.WriteString(childHTML)
	}

	// Закрывающий тег
	sb.WriteString(indentation + "</" + jsx.Type + ">\n")

	return sb.String()
}

// convertComponent преобразует пользовательский компонент в вызов templ
func (c *JSXToHTMXConverter) convertComponent(jsx *models.JSXElement, indentation string) string {
	var sb strings.Builder

	// Имя компонента с маленькой буквы для templ
	templComponentName := strings.ToLower(string(jsx.Type[0])) + jsx.Type[1:]

	sb.WriteString(indentation + "{\n")
	sb.WriteString(indentation + "\t// Вызов компонента " + jsx.Type + "\n")

	// Полный путь к компоненту (с учетом пакета)
	packagePrefix := ""
	if c.options.PackageName != "" && c.options.PackageName != "." {
		packagePrefix = c.options.PackageName + "."
	}

	sb.WriteString(indentation + "\t" + packagePrefix + templComponentName + "(")

	// Если есть пропсы, передаем их
	if len(jsx.Props) > 0 {
		sb.WriteString(packagePrefix + jsx.Type + "Props{\n")

		// Передаем пропсы
		for name, value := range jsx.Props {
			propName := strings.ToUpper(string(name[0])) + name[1:]

			if value == true {
				// Boolean prop
				sb.WriteString(indentation + "\t\t" + propName + ": true,\n")
			} else if valueStr, ok := value.(string); ok {
				// String prop
				sb.WriteString(indentation + "\t\t" + propName + ": \"" + valueStr + "\",\n")
			} else if valueExpr, ok := value.(map[string]interface{}); ok {
				// Expression prop
				if expr, ok := valueExpr["code"].(string); ok {
					// Преобразуем React выражение в Go
					goExpr := c.convertReactExpressionToGo(expr)
					sb.WriteString(indentation + "\t\t" + propName + ": " + goExpr + ",\n")
				}
			}
		}

		sb.WriteString(indentation + "\t}")
	}

	// Если используем HTMX, добавляем ID
	if c.options.UseHtmx {
		if len(jsx.Props) > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("id")
	}

	sb.WriteString(")\n")
	sb.WriteString(indentation + "}\n")

	return sb.String()
}

// convertElementAttributes преобразует атрибуты JSX в атрибуты HTML+HTMX
func (c *JSXToHTMXConverter) convertElementAttributes(jsx *models.JSXElement) string {
	var sb strings.Builder

	// Если используем HTMX и это корневой элемент (indent == 1), добавляем ID
	if c.options.UseHtmx && c.indent == 1 {
		componentName := c.options.ComponentName
		sb.WriteString(fmt.Sprintf(" id=\"%s-{id}\"", componentName))
	}

	// Обычные атрибуты
	for name, value := range jsx.Props {
		// Преобразуем camelCase в kebab-case для HTML атрибутов
		attrName := camelCaseToKebabCase(name)

		// Обработка специальных атрибутов
		if name == "className" {
			attrName = "class"
		} else if name == "htmlFor" {
			attrName = "for"
		}

		// Добавляем HTMX атрибуты, если нужно
		if c.options.UseHtmx {
			// React обработчики событий -> HTMX атрибуты
			htmxAttr := c.convertReactEventToHtmx(name, value)
			if htmxAttr != "" {
				sb.WriteString(htmxAttr)
				continue
			}
		}

		// Обычные атрибуты
		if value == true {
			// Boolean attribute
			sb.WriteString(" " + attrName)
		} else if valueStr, ok := value.(string); ok {
			// String attribute
			sb.WriteString(" " + attrName + "=\"" + valueStr + "\"")
		} else if valueExpr, ok := value.(map[string]interface{}); ok {
			// Expression attribute
			if expr, ok := valueExpr["code"].(string); ok {
				// Преобразуем React выражение в Go
				goExpr := c.convertReactExpressionToGo(expr)
				sb.WriteString(" " + attrName + "=\"{ " + goExpr + " }\"")
			}
		}
	}

	return sb.String()
}

// convertReactEventToHtmx преобразует React обработчик события в атрибуты HTMX
func (c *JSXToHTMXConverter) convertReactEventToHtmx(name string, value interface{}) string {
	// Обрабатываем только React обработчики событий
	if !strings.HasPrefix(name, "on") || len(name) < 3 || name[2] < 'A' || name[2] > 'Z' {
		return ""
	}

	var sb strings.Builder
	componentName := c.options.ComponentName

	switch name {
	case "onClick":
		// React onClick -> hx-post для изменения состояния
		if valueExpr, ok := value.(map[string]interface{}); ok {
			if expr, ok := valueExpr["code"].(string); ok {
				// Пытаемся извлечь имя функции или вызываемого метода
				setterMatch := regexp.MustCompile(`set(\w+)\(`).FindStringSubmatch(expr)
				if len(setterMatch) > 1 {
					stateName := setterMatch[1]
					statePath := strings.ToLower(stateName[:1]) + stateName[1:]

					sb.WriteString(fmt.Sprintf(" hx-post=\"/api/%s/%s?id={id}\"", strings.ToLower(componentName), statePath))
					sb.WriteString(fmt.Sprintf(" hx-target=\"#%s-{id}\"", componentName))
					sb.WriteString(" hx-swap=\"outerHTML\"")

					// Пытаемся извлечь значение, которое передается в setter
					valueMatch := regexp.MustCompile(`\((.*)\)`).FindStringSubmatch(expr)
					if len(valueMatch) > 1 {
						valueExpr := valueMatch[1]
						// Если это не просто переменная, а выражение
						if !regexp.MustCompile(`^\w+$`).MatchString(valueExpr) {
							sb.WriteString(fmt.Sprintf(" hx-vals='{\"value\": %s}'", valueExpr))
						}
					}
				} else {
					// Обычный onClick -> hx-post с именем функции
					funcMatch := regexp.MustCompile(`(\w+)\(`).FindStringSubmatch(expr)
					if len(funcMatch) > 1 {
						funcName := funcMatch[1]
						sb.WriteString(fmt.Sprintf(" hx-post=\"/api/%s/%s?id={id}\"", strings.ToLower(componentName), strings.ToLower(funcName)))
						sb.WriteString(fmt.Sprintf(" hx-target=\"#%s-{id}\"", componentName))
						sb.WriteString(" hx-swap=\"outerHTML\"")
					}
				}
			}
		}

	case "onChange":
		// React onChange -> hx-trigger с keyup событием
		sb.WriteString(" hx-trigger=\"keyup changed delay:500ms\"")

		if valueExpr, ok := value.(map[string]interface{}); ok {
			if expr, ok := valueExpr["code"].(string); ok {
				setterMatch := regexp.MustCompile(`set(\w+)\(`).FindStringSubmatch(expr)
				if len(setterMatch) > 1 {
					stateName := setterMatch[1]
					statePath := strings.ToLower(stateName[:1]) + stateName[1:]

					sb.WriteString(fmt.Sprintf(" hx-post=\"/api/%s/%s?id={id}\"", strings.ToLower(componentName), statePath))
					sb.WriteString(fmt.Sprintf(" hx-target=\"#%s-{id}\"", componentName))
					sb.WriteString(" hx-swap=\"outerHTML\"")
				}
			}
		}

	case "onSubmit":
		// React onSubmit -> hx-post для формы
		sb.WriteString(fmt.Sprintf(" hx-post=\"/api/%s/submit?id={id}\"", strings.ToLower(componentName)))
		sb.WriteString(fmt.Sprintf(" hx-target=\"#%s-{id}\"", componentName))
		sb.WriteString(" hx-swap=\"outerHTML\"")

	case "onFocus", "onBlur":
		// React onFocus/onBlur -> hx-trigger с focus/blur событием
		event := strings.ToLower(name[2:])
		sb.WriteString(fmt.Sprintf(" hx-trigger=\"%s\"", event))

		if valueExpr, ok := value.(map[string]interface{}); ok {
			if expr, ok := valueExpr["code"].(string); ok {
				funcMatch := regexp.MustCompile(`(\w+)\(`).FindStringSubmatch(expr)
				if len(funcMatch) > 1 {
					funcName := funcMatch[1]
					sb.WriteString(fmt.Sprintf(" hx-post=\"/api/%s/%s?id={id}\"", strings.ToLower(componentName), strings.ToLower(funcName)))
					sb.WriteString(fmt.Sprintf(" hx-target=\"#%s-{id}\"", componentName))
					sb.WriteString(" hx-swap=\"outerHTML\"")
				}
			}
		}
	}

	return sb.String()
}

// convertReactExpressionToGo преобразует React выражение в Go
func (c *JSXToHTMXConverter) convertReactExpressionToGo(expr string) string {
	// Это очень упрощенная версия, в реальности потребуется более сложный парсинг
	expr = strings.TrimSpace(expr)

	// Тернарный оператор: condition ? trueExpr : falseExpr
	ternaryRegex := regexp.MustCompile(`(.*)\s*\?\s*(.*)\s*:\s*(.*)`)
	if ternaryRegex.MatchString(expr) {
		matches := ternaryRegex.FindStringSubmatch(expr)
		if len(matches) == 4 {
			condition := c.convertReactExprToGoExpr(matches[1])
			trueValue := c.convertReactExprToGoExpr(matches[2])
			falseValue := c.convertReactExprToGoExpr(matches[3])

			return fmt.Sprintf("templ.KV((%s), %s, %s)", condition, trueValue, falseValue)
		}
	}

	// Преобразуем выражение
	return c.convertReactExprToGoExpr(expr)
}

// convertReactExprToGoExpr преобразует подвыражение React в Go
func (c *JSXToHTMXConverter) convertReactExprToGoExpr(expr string) string {
	expr = strings.TrimSpace(expr)

	// Заменяем некоторые общие выражения

	// Доступ к пропсам
	expr = regexp.MustCompile(`props\.(\w+)`).ReplaceAllString(expr, "props.$1")

	// Логические операторы
	expr = strings.ReplaceAll(expr, "&&", "&&")
	expr = strings.ReplaceAll(expr, "||", "||")
	expr = strings.ReplaceAll(expr, "!", "!")

	// Сравнения
	expr = strings.ReplaceAll(expr, "===", "==")
	expr = strings.ReplaceAll(expr, "!==", "!=")

	// Шаблонные строки
	templateStringRegex := regexp.MustCompile("`(.*?)`")
	if templateStringRegex.MatchString(expr) {
		// Заменяем шаблонные строки на fmt.Sprintf
		expr = templateStringRegex.ReplaceAllStringFunc(expr, func(s string) string {
			// Убираем обратные кавычки
			s = s[1 : len(s)-1]

			// Заменяем ${expr} на %v
			s = regexp.MustCompile(`\$\{(.*?)\}`).ReplaceAllString(s, "%v")

			// Создаем вызов fmt.Sprintf
			return fmt.Sprintf("fmt.Sprintf(\"%s\")", s)
		})
	}

	return expr
}

// Вспомогательные функции

// camelCaseToKebabCase преобразует camelCase в kebab-case
func camelCaseToKebabCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('-')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// isVoidElement проверяет, является ли тег самозакрывающимся
func isVoidElement(tag string) bool {
	voidElements := map[string]bool{
		"area":   true,
		"base":   true,
		"br":     true,
		"col":    true,
		"embed":  true,
		"hr":     true,
		"img":    true,
		"input":  true,
		"link":   true,
		"meta":   true,
		"param":  true,
		"source": true,
		"track":  true,
		"wbr":    true,
	}
	return voidElements[tag]
}
