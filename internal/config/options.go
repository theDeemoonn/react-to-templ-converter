package config

// ConversionOptions определяет опции для конвертации React в Templ/HTMX
type ConversionOptions struct {
	// UseHtmx включает использование HTMX для интерактивности
	UseHtmx bool

	// ComponentName задает имя компонента (если не указано, используется имя из парсера)
	ComponentName string

	// PackageName задает имя пакета для Go/templ файлов
	PackageName string

	// IncludeComments добавляет комментарии к сгенерированному коду
	IncludeComments bool

	// CustomImports добавляет пользовательские импорты к Go файлам
	CustomImports []string

	// StatePersistence определяет способ хранения состояния
	// Возможные значения: "memory", "redis", "database"
	StatePersistence string

	// Debug включает режим отладки
	Debug bool

	// Indentation настройки отступов
	Indentation struct {
		Style string // "spaces" или "tabs"
		Size  int    // количество пробелов или табуляций
	}
}

// NewDefaultOptions создает новые опции конвертации со значениями по умолчанию
func NewDefaultOptions() *ConversionOptions {
	options := &ConversionOptions{
		UseHtmx:          true,
		PackageName:      "templates",
		IncludeComments:  true,
		StatePersistence: "memory",
		Debug:            false,
	}

	options.Indentation.Style = "spaces"
	options.Indentation.Size = 4

	return options
}

// Clone создает копию опций
func (o *ConversionOptions) Clone() *ConversionOptions {
	clone := *o

	if o.CustomImports != nil {
		clone.CustomImports = make([]string, len(o.CustomImports))
		copy(clone.CustomImports, o.CustomImports)
	}

	return &clone
}
