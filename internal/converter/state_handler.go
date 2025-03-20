package converter

import (
	"fmt"
	"react-to-templ-converter/internal/config"
	"react-to-templ-converter/internal/models"
	"strings"
)

// StateHandler обрабатывает React состояния и преобразует их в Go/HTMX
type StateHandler struct {
	options     *config.ConversionOptions
	debug       bool
	indentSize  int
	indentStyle string
}

// NewStateHandler создает новый обработчик состояний
func NewStateHandler(options *config.ConversionOptions) *StateHandler {
	indentStyle := "spaces"
	indentSize := 4

	if options.Indentation.Style != "" {
		indentStyle = options.Indentation.Style
	}

	if options.Indentation.Size > 0 {
		indentSize = options.Indentation.Size
	}

	return &StateHandler{
		options:     options,
		debug:       options.Debug,
		indentSize:  indentSize,
		indentStyle: indentStyle,
	}
}

// SetDebug устанавливает режим отладки
func (h *StateHandler) SetDebug(debug bool) {
	h.debug = debug
}

// SetIndentation устанавливает стиль отступов
func (h *StateHandler) SetIndentation(style string, size int) {
	h.indentStyle = style
	h.indentSize = size
}

// GenerateStateStructs генерирует структуры Go для хранения состояний компонента
func (h *StateHandler) GenerateStateStructs(component *models.ReactComponent) string {
	if len(component.State) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("// %sState определяет состояние компонента %s\n", component.Name, component.Name))
	sb.WriteString(fmt.Sprintf("type %sState struct {\n", component.Name))

	indent := h.getIndentation(1)

	// Поля для состояний
	for _, state := range component.State {
		goType := h.convertTypeToGo(state.Type, state.InitialValue)

		fieldName := strings.ToUpper(string(state.Name[0])) + state.Name[1:]
		sb.WriteString(fmt.Sprintf("%s%s %s\n", indent, fieldName, goType))
	}

	sb.WriteString("}\n\n")

	// Глобальное хранилище состояний
	sb.WriteString("var (\n")

	// В зависимости от способа хранения состояния
	switch h.options.StatePersistence {
	case "redis":
		// Для Redis используем клиент и ключи
		sb.WriteString(fmt.Sprintf("%sredisClient *redis.Client\n", indent))
		sb.WriteString(fmt.Sprintf("%s%sKeyPrefix = \"%s:\"\n", indent, strings.ToLower(component.Name), strings.ToLower(component.Name)))
	case "database":
		// Для БД используем типичный интерфейс репозитория
		sb.WriteString(fmt.Sprintf("%s%sRepository Repository\n", indent, strings.ToLower(component.Name)))
	default:
		// По умолчанию: хранение в памяти
		sb.WriteString(fmt.Sprintf("%s%sStates = make(map[string]*%sState)\n", indent, strings.ToLower(component.Name), component.Name))
		sb.WriteString(fmt.Sprintf("%s%sMutex sync.RWMutex\n", indent, strings.ToLower(component.Name)))
	}

	sb.WriteString(")\n\n")

	return sb.String()
}

// GenerateStateHandlers генерирует Go обработчики для взаимодействия с состоянием
func (h *StateHandler) GenerateStateHandlers(component *models.ReactComponent) string {
	if len(component.State) == 0 {
		return ""
	}

	var sb strings.Builder
	indent := h.getIndentation(1)

	// Функция для создания нового экземпляра компонента
	sb.WriteString(fmt.Sprintf("// New%s создает новый экземпляр компонента\n", component.Name))
	sb.WriteString(fmt.Sprintf("func New%s(w http.ResponseWriter, r *http.Request) {\n", component.Name))
	sb.WriteString(fmt.Sprintf("%s// Генерируем уникальный ID для компонента\n", indent))
	sb.WriteString(fmt.Sprintf("%sid := uuid.New().String()\n\n", indent))

	// Получение параметров из запроса
	if len(component.Props) > 0 {
		sb.WriteString(fmt.Sprintf("%s// Получаем пропсы из запроса\n", indent))
		if h.options.PackageName != "" && h.options.PackageName != "." {
			sb.WriteString(fmt.Sprintf("%svar props %s.%sProps\n", indent, h.options.PackageName, component.Name))
		} else {
			sb.WriteString(fmt.Sprintf("%svar props %sProps\n", indent, component.Name))
		}
		sb.WriteString(fmt.Sprintf("%sif err := json.NewDecoder(r.Body).Decode(&props); err != nil {\n", indent))
		sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"Ошибка декодирования пропсов\", http.StatusBadRequest)\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s}\n\n", indent))
	}

	// Создание начального состояния
	sb.WriteString(fmt.Sprintf("%s// Создаем начальное состояние\n", indent))
	sb.WriteString(fmt.Sprintf("%sstate := &%sState{\n", indent, component.Name))

	for _, state := range component.State {
		stateName := strings.ToUpper(string(state.Name[0])) + state.Name[1:]

		// Установка начального значения
		if state.InitialValue != nil {
			sb.WriteString(fmt.Sprintf("%s%s%s: %v,\n", indent, indent, stateName, h.formatGoValue(state.InitialValue)))
		} else {
			// Значение по умолчанию для типа
			goType := h.convertTypeToGo(state.Type, nil)
			sb.WriteString(fmt.Sprintf("%s%s%s: %s,\n", indent, indent, stateName, h.getDefaultValueForType(goType)))
		}
	}

	sb.WriteString(fmt.Sprintf("%s}\n\n", indent))

	// Сохранение состояния в зависимости от способа хранения
	sb.WriteString(fmt.Sprintf("%s// Сохраняем состояние\n", indent))
	switch h.options.StatePersistence {
	case "redis":
		// Для Redis: сериализуем состояние в JSON и сохраняем с TTL
		sb.WriteString(fmt.Sprintf("%sjsonState, _ := json.Marshal(state)\n", indent))
		sb.WriteString(fmt.Sprintf("%sredisClient.Set(context.Background(), %sKeyPrefix+id, jsonState, 24*time.Hour)\n\n", indent, strings.ToLower(component.Name)))
	case "database":
		// Для БД: используем метод репозитория
		sb.WriteString(fmt.Sprintf("%s%sRepository.SaveState(id, state)\n\n", indent, strings.ToLower(component.Name)))
	default:
		// По умолчанию: хранение в памяти
		sb.WriteString(fmt.Sprintf("%s%sMutex.Lock()\n", indent, strings.ToLower(component.Name)))
		sb.WriteString(fmt.Sprintf("%s%sStates[id] = state\n", indent, strings.ToLower(component.Name)))
		sb.WriteString(fmt.Sprintf("%s%sMutex.Unlock()\n\n", indent, strings.ToLower(component.Name)))
	}

	// Рендеринг компонента
	sb.WriteString(fmt.Sprintf("%s// Рендерим компонент\n", indent))

	if h.options.PackageName != "" && h.options.PackageName != "." {
		sb.WriteString(fmt.Sprintf("%stempl.Handler(%s.%s(", indent, h.options.PackageName,
			strings.ToLower(string(component.Name[0]))+component.Name[1:]))
	} else {
		sb.WriteString(fmt.Sprintf("%stempl.Handler(%s(", indent, strings.ToLower(string(component.Name[0]))+component.Name[1:]))
	}

	if len(component.Props) > 0 {
		sb.WriteString("props, ")
	}
	sb.WriteString("id)).ServeHTTP(w, r)\n")
	sb.WriteString("}\n\n")

	// Создаем обработчики для каждого состояния
	for _, state := range component.State {
		h.generateStateUpdater(&sb, component, state)
	}

	// Добавляем функции для callbacks, если они есть
	for _, callback := range component.Callbacks {
		h.generateCallbackHandler(&sb, component, callback)
	}

	// Добавляем обработчики для эффектов, если они есть и не используются только для загрузки данных
	for i, effect := range component.Effects {
		if !isEffectForDataFetching(effect) {
			h.generateEffectHandler(&sb, component, effect, i)
		}
	}

	return sb.String()
}

// GenerateHtmxJSHelpers генерирует JavaScript код для поддержки HTMX
func (h *StateHandler) GenerateHtmxJSHelpers(component *models.ReactComponent) string {
	if !h.options.UseHtmx || (len(component.State) == 0 && len(component.Effects) == 0) {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("// JavaScript для работы с компонентом " + component.Name + "\n\n")

	sb.WriteString("document.addEventListener('DOMContentLoaded', function() {\n")

	indent := h.getIndentation(1)

	// Инициализация обработчиков после загрузки компонента
	sb.WriteString(fmt.Sprintf("%s// Обработчики для компонента будут добавлены после его загрузки через HTMX\n", indent))
	sb.WriteString(fmt.Sprintf("%sdocument.body.addEventListener('htmx:afterSwap', function(event) {\n", indent))
	sb.WriteString(fmt.Sprintf("%s%s// Проверяем, относится ли событие к нашему компоненту\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s%sconst target = event.detail.target;\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s%sif (target && target.id && target.id.startsWith('%s-')) {\n", indent, indent, component.Name))
	sb.WriteString(fmt.Sprintf("%s%s%s// Инициализация компонента после загрузки\n", indent, indent, indent))
	sb.WriteString(fmt.Sprintf("%s%s%sinitialize%s(target.id.split('-')[1]);\n", indent, indent, indent, component.Name))
	sb.WriteString(fmt.Sprintf("%s%s}\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s});\n", indent))

	// Функция инициализации компонента
	sb.WriteString(fmt.Sprintf("\n%s// Функция инициализации компонента\n", indent))
	sb.WriteString(fmt.Sprintf("%sfunction initialize%s(id) {\n", indent, component.Name))

	// Если есть эффекты, которые не используются только для загрузки данных
	hasNonDataFetchingEffects := false
	for _, effect := range component.Effects {
		if !isEffectForDataFetching(effect) {
			hasNonDataFetchingEffects = true
			break
		}
	}

	if hasNonDataFetchingEffects {
		sb.WriteString(fmt.Sprintf("%s%s// Установка эффектов, не связанных с загрузкой данных\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s%ssetupEffects(id);\n", indent, indent))
	}

	sb.WriteString(fmt.Sprintf("%s}\n", indent))

	// Функция для эффектов
	if hasNonDataFetchingEffects {
		sb.WriteString(fmt.Sprintf("\n%s// Функция для настройки эффектов\n", indent))
		sb.WriteString(fmt.Sprintf("%sfunction setupEffects(id) {\n", indent))

		for i, effect := range component.Effects {
			if !isEffectForDataFetching(effect) {
				sb.WriteString(fmt.Sprintf("%s%s// Эффект %d\n", indent, indent, i+1))

				// Создаем список зависимостей
				if len(effect.Dependencies) > 0 {
					sb.WriteString(fmt.Sprintf("%s%s// Зависимости: %s\n", indent, indent, strings.Join(effect.Dependencies, ", ")))

					// Если есть зависимости, добавляем обработчики для их отслеживания
					for _, dep := range effect.Dependencies {
						sb.WriteString(fmt.Sprintf("%s%s// Отслеживаем изменение %s\n", indent, indent, dep))
						sb.WriteString(fmt.Sprintf("%s%sdocument.getElementById('%s-' + id).addEventListener('htmx:afterSettle', function(event) {\n", indent, indent, component.Name))
						sb.WriteString(fmt.Sprintf("%s%s%s// Проверяем, изменилось ли состояние %s\n", indent, indent, indent, dep))
						sb.WriteString(fmt.Sprintf("%s%s%sfetch('/api/%s/effect/%d?id=' + id, { method: 'POST' });\n", indent, indent, indent, strings.ToLower(component.Name), i))
						sb.WriteString(fmt.Sprintf("%s%s});\n", indent, indent))
					}
				} else {
					// Если нет зависимостей, выполняем эффект при загрузке
					sb.WriteString(fmt.Sprintf("%s%s// Эффект без зависимостей, выполняется при загрузке\n", indent, indent))
					sb.WriteString(fmt.Sprintf("%s%sfetch('/api/%s/effect/%d?id=' + id, { method: 'POST' });\n", indent, indent, strings.ToLower(component.Name), i))
				}

				sb.WriteString("\n")
			}
		}

		sb.WriteString(fmt.Sprintf("%s}\n", indent))
	}

	// Обработка очистки при удалении компонента
	sb.WriteString(fmt.Sprintf("\n%s// Очистка ресурсов при удалении компонента\n", indent))
	sb.WriteString(fmt.Sprintf("%sdocument.body.addEventListener('htmx:beforeCleanupElement', function(event) {\n", indent))
	sb.WriteString(fmt.Sprintf("%s%sconst element = event.detail.element;\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s%sif (element && element.id && element.id.startsWith('%s-')) {\n", indent, indent, component.Name))
	sb.WriteString(fmt.Sprintf("%s%s%sconst id = element.id.split('-')[1];\n", indent, indent, indent))
	sb.WriteString(fmt.Sprintf("%s%s%s// Уведомляем сервер о необходимости очистки ресурсов\n", indent, indent, indent))
	sb.WriteString(fmt.Sprintf("%s%s%sfetch('/api/%s/cleanup?id=' + id, { method: 'POST' });\n", indent, indent, indent, strings.ToLower(component.Name)))
	sb.WriteString(fmt.Sprintf("%s%s}\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s});\n", indent))

	// Первоначальная инициализация компонента
	sb.WriteString(fmt.Sprintf("\n%s// Попытка инициализации компонента при загрузке страницы\n", indent))
	sb.WriteString(fmt.Sprintf("%sconst components = document.querySelectorAll('[id^=\"%s-\"]');\n", indent, component.Name))
	sb.WriteString(fmt.Sprintf("%scomponents.forEach(function(component) {\n", indent))
	sb.WriteString(fmt.Sprintf("%s%sconst id = component.id.split('-')[1];\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s%sinitialize%s(id);\n", indent, indent, component.Name))
	sb.WriteString(fmt.Sprintf("%s});\n", indent))

	sb.WriteString("});\n")

	return sb.String()
}

// Вспомогательные методы

// generateStateUpdater генерирует обработчик для обновления состояния
func (h *StateHandler) generateStateUpdater(sb *strings.Builder, component *models.ReactComponent, state models.StateDefinition) {
	stateName := strings.ToUpper(string(state.Name[0])) + state.Name[1:]
	setterName := state.Setter

	// Имя обработчика
	handlerName := setterName
	// Если setter не начинается с большой буквы, делаем первую букву заглавной
	if len(handlerName) > 0 && handlerName[0] >= 'a' && handlerName[0] <= 'z' {
		handlerName = strings.ToUpper(string(handlerName[0])) + handlerName[1:]
	}

	indent := h.getIndentation(1)

	sb.WriteString(fmt.Sprintf("// %s обрабатывает изменение состояния %s\n", handlerName, state.Name))
	sb.WriteString(fmt.Sprintf("func %s(w http.ResponseWriter, r *http.Request) {\n", handlerName))
	sb.WriteString(fmt.Sprintf("%s// Получаем ID компонента из запроса\n", indent))
	sb.WriteString(fmt.Sprintf("%sid := r.URL.Query().Get(\"id\")\n", indent))
	sb.WriteString(fmt.Sprintf("%sif id == \"\" {\n", indent))
	sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"ID компонента не указан\", http.StatusBadRequest)\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s}\n\n", indent))

	// Получаем и проверяем состояние компонента
	switch h.options.StatePersistence {
	case "redis":
		sb.WriteString(fmt.Sprintf("%s// Получаем состояние из Redis\n", indent))
		sb.WriteString(fmt.Sprintf("%sjsonState, err := redisClient.Get(context.Background(), %sKeyPrefix+id).Bytes()\n", indent, strings.ToLower(component.Name)))
		sb.WriteString(fmt.Sprintf("%sif err != nil {\n", indent))
		sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"Состояние компонента не найдено\", http.StatusNotFound)\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s}\n\n", indent))

		sb.WriteString(fmt.Sprintf("%svar state %sState\n", indent, component.Name))
		sb.WriteString(fmt.Sprintf("%sif err := json.Unmarshal(jsonState, &state); err != nil {\n", indent))
		sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"Ошибка десериализации состояния\", http.StatusInternalServerError)\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s}\n\n", indent))

	case "database":
		sb.WriteString(fmt.Sprintf("%s// Получаем состояние из БД\n", indent))
		sb.WriteString(fmt.Sprintf("%sstate, err := %sRepository.GetState(id)\n", indent, strings.ToLower(component.Name)))
		sb.WriteString(fmt.Sprintf("%sif err != nil {\n", indent))
		sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"Состояние компонента не найдено\", http.StatusNotFound)\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s}\n\n", indent))

	default:
		sb.WriteString(fmt.Sprintf("%s// Получаем состояние из памяти\n", indent))
		sb.WriteString(fmt.Sprintf("%s%sMutex.RLock()\n", indent, strings.ToLower(component.Name)))
		sb.WriteString(fmt.Sprintf("%sstate, ok := %sStates[id]\n", indent, strings.ToLower(component.Name)))
		sb.WriteString(fmt.Sprintf("%s%sMutex.RUnlock()\n", indent, strings.ToLower(component.Name)))
		sb.WriteString(fmt.Sprintf("%sif !ok {\n", indent))
		sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"Состояние компонента не найдено\", http.StatusNotFound)\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s}\n\n", indent))
	}

	// Получаем новое значение состояния
	sb.WriteString(fmt.Sprintf("%s// Получаем новое значение состояния из запроса\n", indent))

	// Тип значения зависит от типа состояния
	goType := h.convertTypeToGo(state.Type, state.InitialValue)

	if goType == "string" {
		sb.WriteString(fmt.Sprintf("%snewValue := r.FormValue(\"value\")\n\n", indent))
	} else if goType == "int" {
		sb.WriteString(fmt.Sprintf("%snewValueStr := r.FormValue(\"value\")\n", indent))
		sb.WriteString(fmt.Sprintf("%snewValue, err := strconv.Atoi(newValueStr)\n", indent))
		sb.WriteString(fmt.Sprintf("%sif err != nil {\n", indent))
		sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"Неверный формат значения\", http.StatusBadRequest)\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s}\n\n", indent))
	} else if goType == "bool" {
		sb.WriteString(fmt.Sprintf("%snewValueStr := r.FormValue(\"value\")\n", indent))
		sb.WriteString(fmt.Sprintf("%snewValue := newValueStr == \"true\"\n\n", indent))
	} else if goType == "float64" {
		sb.WriteString(fmt.Sprintf("%snewValueStr := r.FormValue(\"value\")\n", indent))
		sb.WriteString(fmt.Sprintf("%snewValue, err := strconv.ParseFloat(newValueStr, 64)\n", indent))
		sb.WriteString(fmt.Sprintf("%sif err != nil {\n", indent))
		sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"Неверный формат значения\", http.StatusBadRequest)\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s}\n\n", indent))
	} else {
		sb.WriteString(fmt.Sprintf("%svar newValue %s\n", indent, goType))
		sb.WriteString(fmt.Sprintf("%sif err := json.NewDecoder(r.Body).Decode(&newValue); err != nil {\n", indent))
		sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"Ошибка декодирования значения\", http.StatusBadRequest)\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s}\n\n", indent))
	}

	// Обновляем состояние
	sb.WriteString(fmt.Sprintf("%s// Обновляем состояние\n", indent))

	switch h.options.StatePersistence {
	case "redis":
		sb.WriteString(fmt.Sprintf("%sstate.%s = newValue\n", indent, stateName))
		sb.WriteString(fmt.Sprintf("%sjsonState, _ = json.Marshal(state)\n", indent))
		sb.WriteString(fmt.Sprintf("%sredisClient.Set(context.Background(), %sKeyPrefix+id, jsonState, 24*time.Hour)\n\n", indent, strings.ToLower(component.Name)))

	case "database":
		sb.WriteString(fmt.Sprintf("%sstate.%s = newValue\n", indent, stateName))
		sb.WriteString(fmt.Sprintf("%sif err := %sRepository.UpdateState(id, state); err != nil {\n", indent, strings.ToLower(component.Name)))
		sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"Ошибка обновления состояния\", http.StatusInternalServerError)\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
		sb.WriteString(fmt.Sprintf("%s}\n\n", indent))

	default:
		sb.WriteString(fmt.Sprintf("%s%sMutex.Lock()\n", indent, strings.ToLower(component.Name)))
		sb.WriteString(fmt.Sprintf("%sstate.%s = newValue\n", indent, stateName))
		sb.WriteString(fmt.Sprintf("%s%sMutex.Unlock()\n\n", indent, strings.ToLower(component.Name)))
	}

	// Рендерим компонент заново
	sb.WriteString(fmt.Sprintf("%s// Получаем пропсы (в реальном приложении нужно сохранять пропсы)\n", indent))
	if h.options.PackageName != "" && h.options.PackageName != "." {
		sb.WriteString(fmt.Sprintf("%svar props %s.%sProps\n\n", indent, h.options.PackageName, component.Name))
	} else {
		sb.WriteString(fmt.Sprintf("%svar props %sProps\n\n", indent, component.Name))
	}

	sb.WriteString(fmt.Sprintf("%s// Рендерим компонент с обновленным состоянием\n", indent))

	if h.options.PackageName != "" && h.options.PackageName != "." {
		sb.WriteString(fmt.Sprintf("%stempl.Handler(%s.%s(props, id)).ServeHTTP(w, r)\n", indent, h.options.PackageName, strings.ToLower(string(component.Name[0]))+component.Name[1:]))
	} else {
		sb.WriteString(fmt.Sprintf("%stempl.Handler(%s(props, id)).ServeHTTP(w, r)\n", indent, strings.ToLower(string(component.Name[0]))+component.Name[1:]))
	}

	sb.WriteString("}\n\n")
}

// generateCallbackHandler генерирует обработчик для callback функции
func (h *StateHandler) generateCallbackHandler(sb *strings.Builder, component *models.ReactComponent, callback models.CallbackDefinition) {
	callbackName := callback.Name

	// Имя обработчика
	handlerName := callbackName
	// Если callbackName не начинается с большой буквы, делаем первую букву заглавной
	if len(handlerName) > 0 && handlerName[0] >= 'a' && handlerName[0] <= 'z' {
		handlerName = strings.ToUpper(string(handlerName[0])) + handlerName[1:]
	}

	indent := h.getIndentation(1)

	sb.WriteString(fmt.Sprintf("// %s обрабатывает вызов callback-функции %s\n", handlerName, callbackName))
	sb.WriteString(fmt.Sprintf("func %s(w http.ResponseWriter, r *http.Request) {\n", handlerName))
	sb.WriteString(fmt.Sprintf("%s// Получаем ID компонента из запроса\n", indent))
	sb.WriteString(fmt.Sprintf("%sid := r.URL.Query().Get(\"id\")\n", indent))
	sb.WriteString(fmt.Sprintf("%sif id == \"\" {\n", indent))
	sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"ID компонента не указан\", http.StatusBadRequest)\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s}\n\n", indent))

	// Получаем и проверяем состояние компонента (аналогично обработчику состояния)
	// ...код аналогичен generateStateUpdater...

	// Логика callback (зависит от конкретного callback)
	sb.WriteString(fmt.Sprintf("%s// Реализация callback-функции %s\n", indent, callbackName))
	sb.WriteString(fmt.Sprintf("%s// TODO: Реализуйте логику callback-функции\n\n", indent))

	// Рендерим компонент заново
	sb.WriteString(fmt.Sprintf("%s// Рендерим компонент с обновленным состоянием\n", indent))
	if h.options.PackageName != "" && h.options.PackageName != "." {
		sb.WriteString(fmt.Sprintf("%stempl.Handler(%s.%s(props, id)).ServeHTTP(w, r)\n", indent, h.options.PackageName, strings.ToLower(string(component.Name[0]))+component.Name[1:]))
	} else {
		sb.WriteString(fmt.Sprintf("%stempl.Handler(%s(props, id)).ServeHTTP(w, r)\n", indent, strings.ToLower(string(component.Name[0]))+component.Name[1:]))
	}

	sb.WriteString("}\n\n")
}

// generateEffectHandler генерирует обработчик для useEffect
func (h *StateHandler) generateEffectHandler(sb *strings.Builder, component *models.ReactComponent, effect models.EffectDefinition, index int) {
	handlerName := fmt.Sprintf("Effect%d", index+1)

	indent := h.getIndentation(1)

	sb.WriteString(fmt.Sprintf("// %s обрабатывает эффект компонента\n", handlerName))
	sb.WriteString(fmt.Sprintf("func %s(w http.ResponseWriter, r *http.Request) {\n", handlerName))
	sb.WriteString(fmt.Sprintf("%s// Получаем ID компонента из запроса\n", indent))
	sb.WriteString(fmt.Sprintf("%sid := r.URL.Query().Get(\"id\")\n", indent))
	sb.WriteString(fmt.Sprintf("%sif id == \"\" {\n", indent))
	sb.WriteString(fmt.Sprintf("%s%shttp.Error(w, \"ID компонента не указан\", http.StatusBadRequest)\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s%sreturn\n", indent, indent))
	sb.WriteString(fmt.Sprintf("%s}\n\n", indent))

	// Получаем и проверяем состояние компонента (аналогично обработчику состояния)
	// ...код аналогичен generateStateUpdater...

	// Логика эффекта (зависит от конкретного эффекта)
	sb.WriteString(fmt.Sprintf("%s// Реализация эффекта\n", indent))
	sb.WriteString(fmt.Sprintf("%s// TODO: Реализуйте логику эффекта на основе его тела и зависимостей\n", indent))
	if len(effect.Dependencies) > 0 {
		sb.WriteString(fmt.Sprintf("%s// Зависимости: %s\n", indent, strings.Join(effect.Dependencies, ", ")))
	}
	sb.WriteString("\n")

	// Возвращаем успешный статус
	sb.WriteString(fmt.Sprintf("%s// Возвращаем успешный статус\n", indent))
	sb.WriteString(fmt.Sprintf("%sw.WriteHeader(http.StatusOK)\n", indent))

	sb.WriteString("}\n\n")
}

// Utility functions

// convertTypeToGo преобразует тип TypeScript/JavaScript в тип Go
func (h *StateHandler) convertTypeToGo(tsType string, value interface{}) string {
	if tsType == "" && value != nil {
		// Определяем тип на основе значения
		return h.inferGoTypeFromValue(value)
	}

	switch tsType {
	case "string":
		return "string"
	case "number":
		return "float64"
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
			goElementType := h.convertTypeToGo(elementType, nil)
			return "[]" + goElementType
		}
		return "interface{}"
	}
}

// inferGoTypeFromValue определяет тип Go на основе значения
func (h *StateHandler) inferGoTypeFromValue(value interface{}) string {
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
func (h *StateHandler) formatGoValue(value interface{}) string {
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
func (h *StateHandler) getDefaultValueForType(goType string) string {
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
func (h *StateHandler) getIndentation(level int) string {
	if h.indentStyle == "tabs" {
		return strings.Repeat("\t", level)
	}
	return strings.Repeat(" ", h.indentSize*level)
}

// isEffectForDataFetching проверяет, используется ли эффект только для загрузки данных
func isEffectForDataFetching(effect models.EffectDefinition) bool {
	// Упрощенная эвристика: если в теле эффекта есть fetch или axios,
	// и нет других явных побочных эффектов, то считаем, что эффект
	// используется только для загрузки данных
	body := effect.Body
	hasFetch := strings.Contains(body, "fetch(") || strings.Contains(body, "axios.")
	hasOtherEffects := strings.Contains(body, "document.") ||
		strings.Contains(body, "window.") ||
		strings.Contains(body, "localStorage") ||
		strings.Contains(body, "sessionStorage")

	return hasFetch && !hasOtherEffects
}
