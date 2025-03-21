# React → Templ/Go/HTMX Конвертер

Конвертер для преобразования React компонентов (TypeScript/JavaScript) в Go шаблоны с использованием библиотеки [templ](https://github.com/a-h/templ) и [HTMX](https://htmx.org/).

## Возможности

- Конвертация React компонентов (функциональных) в шаблоны templ
- Преобразование React хуков (useState, useEffect) в HTMX-атрибуты и серверные обработчики Go
- Генерация Go-контроллеров для управления состоянием на сервере
- Удобный веб-интерфейс для загрузки и конвертации компонентов
- Примеры типичных компонентов для обучения

## Требования

- [Go](https://golang.org/dl/) (версия 1.19+)
- [Node.js](https://nodejs.org/) (версия 16+)
- [Templ](https://github.com/a-h/templ) (`go install github.com/a-h/templ/cmd/templ@latest`)

## Установка

### Через Go

```bash
# Клонирование репозитория
git clone https://github.com/your-username/react-to-templ-converter.git
cd react-to-templ-converter

# Установка зависимостей и сборка
make setup
```

### Через Docker

```bash
# Сборка и запуск Docker-контейнера
docker-compose up -d
```

## Запуск

После установки запустите приложение:

```bash
make run
```

Откройте браузер и перейдите по адресу [http://localhost:8080](http://localhost:8080)

## Использование

1. **Загрузка React компонента**:
    - Перетащите `.tsx`, `.jsx`, `.ts` или `.js` файл в область загрузки
    - Или нажмите на область для выбора файла через диалог

2. **Настройка параметров конвертации**:
    - **Использовать HTMX**: включите для добавления HTMX-атрибутов и создания интерактивного компонента
    - При необходимости настройте дополнительные параметры

3. **Просмотр и копирование результата**:
    - После конвертации вы увидите три вкладки:
        - **Templ шаблон**: основной код шаблона для Go
        - **Go контроллер**: серверный код для обработки HTMX-запросов
        - **JavaScript**: дополнительный JS-код для клиентской части (если требуется)

4. **Использование примеров**:
    - Нажмите на одну из кнопок примеров для загрузки готового React компонента
    - Изучите результат конвертации для понимания принципов работы

## Архитектура

Система состоит из следующих основных компонентов:

1. **Парсер React/TypeScript**:
    - Использует Node.js и Babel для анализа кода
    - Извлекает структуру компонента, пропсы, состояния, эффекты и JSX

2. **Конвертер**:
    - Преобразует React структуры в templ/Go/HTMX
    - Генерирует шаблоны templ, Go контроллеры и JS вспомогательные функции

3. **Веб-интерфейс**:
    - Позволяет загружать и конвертировать компоненты
    - Показывает результаты конвертации и предоставляет примеры

## Поддерживаемые функции React

| Функциональность | Статус | Примечания |
|------------------|--------|------------|
| Функциональные компоненты | ✅ | Полная поддержка |
| Props | ✅ | TypeScript интерфейсы преобразуются в Go структуры |
| useState | ✅ | Преобразуется в серверное состояние + HTMX |
| useEffect | ✅ | Базовая поддержка через HTMX-триггеры |
| useRef | ⚠️ | Ограниченная поддержка |
| useCallback | ⚠️ | Базовая поддержка |
| Условный рендеринг | ✅ | Преобразуется в условные блоки templ |
| Рендеринг списков | ✅ | Преобразуется в циклы for в templ |
| Компонентная композиция | ✅ | Поддерживается через компоненты templ |

## Примеры

### Счетчик

**React**:
```tsx
const Counter: React.FC<{initialCount: number}> = ({ initialCount }) => {
  const [count, setCount] = useState(initialCount);
  
  return (
    <div>
      <h2>Счетчик: {count}</h2>
      <button onClick={() => setCount(count - 1)}>-</button>
      <button onClick={() => setCount(count + 1)}>+</button>
    </div>
  );
};
```

**Templ + HTMX**:
```html
templ counter(props CounterProps, id string) {
  <div id={"counter-" + id}>
    <h2>Счетчик: { strconv.Itoa(props.Count) }</h2>
    <button 
      hx-post={"/api/counter/decrement?id=" + id} 
      hx-target={"#counter-" + id} 
      hx-swap="outerHTML">
      -
    </button>
    <button 
      hx-post={"/api/counter/increment?id=" + id} 
      hx-target={"#counter-" + id} 
      hx-swap="outerHTML">
      +
    </button>
  </div>
}
```

## Разработка

### Структура проекта

```
react-to-templ-converter/
├── cmd/                      # Точки входа
├── internal/                 # Внутренние пакеты
│   ├── parser/               # Парсинг React компонентов
│   ├── converter/            # Конвертация в templ/Go/HTMX
│   ├── generator/            # Генерация результирующих файлов
│   └── models/               # Модели данных
├── web/                      # Веб-интерфейс
│   ├── static/               # Статические файлы
│   └── templates/            # Шаблоны веб-интерфейса
├── parser-js/                # JavaScript парсер React
├── examples/                 # Примеры компонентов
├── go.mod
└── README.md
```

### Сборка и тестирование

```bash
# Запуск тестов
make test

# Запуск линтера
make lint

# Сборка для продакшн
make build
```

## Лицензия

MIT