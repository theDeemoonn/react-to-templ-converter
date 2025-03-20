// JavaScript для форматирования результатов конвертации React → Templ
document.addEventListener('DOMContentLoaded', function() {
    // Элементы формы
    const fileInput = document.getElementById('file-input');
    const convertBtn = document.getElementById('convert-btn');
    const form = document.getElementById('upload-form');
    const dragDropArea = document.getElementById('drag-drop-area');
    const sourceCode = document.getElementById('source-code');

    // Разблокировка кнопки конвертации при выборе файла
    fileInput.addEventListener('change', function() {
        if (fileInput.files.length > 0) {
            convertBtn.disabled = false;

            // Отображаем имя выбранного файла
            const fileName = fileInput.files[0].name;
            dragDropArea.innerHTML = `<p>Выбран файл: ${fileName}</p>`;

            // Читаем и отображаем содержимое файла
            const reader = new FileReader();
            reader.onload = function(e) {
                sourceCode.textContent = e.target.result;
                if (window.hljs) {
                    hljs.highlightElement(sourceCode);
                }
            };
            reader.readAsText(fileInput.files[0]);
        } else {
            convertBtn.disabled = true;
        }
    });

    // Обработка формы с HTMX
    form.addEventListener('htmx:beforeRequest', function(e) {
        // Убедимся, что флажок useHtmx отмечен
        const useHtmxCheckbox = document.getElementById('useHtmx');
        if (useHtmxCheckbox && !useHtmxCheckbox.checked) {
            useHtmxCheckbox.checked = true;
        }
    });

    // Обработка результата конвертации
    document.body.addEventListener('htmx:afterSwap', function(event) {
        if (event.detail.target.id === 'result-container') {
            try {
                // Проверяем, является ли содержимое JSON строкой
                const content = event.detail.target.textContent.trim();
                if (content.startsWith('{') && content.endsWith('}')) {
                    // Пытаемся распарсить JSON
                    const jsonData = JSON.parse(content);

                    // Создаем красивое представление результата
                    event.detail.target.innerHTML = createFormattedResult(jsonData);

                    // Инициализируем Bootstrap табы
                    initBootstrapTabs();

                    // Подсветка синтаксиса и кнопки копирования
                    initCodeHighlighting();
                }
            } catch (error) {
                console.error('Ошибка обработки результата:', error);
            }
        }
    });

    // Функция для создания форматированного HTML результата
    function createFormattedResult(data) {
        // Проверяем, что у нас есть результаты конвертации
        if (!data.templFile && !data.goController && !data.htmxJS) {
            return '<div class="alert alert-warning">Не удалось получить результаты конвертации</div>';
        }

        let html = `
            <div class="alert alert-success">
                <h4 class="alert-heading">Конвертация успешно завершена!</h4>
                <p>Ваш React компонент был успешно преобразован в templ и Go код.</p>
            </div>
            
            <div class="card mt-3">
                <div class="card-body">
                    <h5 class="card-title">Результаты конвертации</h5>
                    
                    <ul class="nav nav-tabs" id="resultTabs" role="tablist">
                        <li class="nav-item" role="presentation">
                            <button class="nav-link active" id="templ-tab" data-bs-toggle="tab" data-bs-target="#templ" type="button" role="tab">
                                Templ шаблон
                            </button>
                        </li>
                        <li class="nav-item" role="presentation">
                            <button class="nav-link" id="controller-tab" data-bs-toggle="tab" data-bs-target="#controller" type="button" role="tab">
                                Go контроллер
                            </button>
                        </li>
                        <li class="nav-item" role="presentation">
                            <button class="nav-link" id="js-tab" data-bs-toggle="tab" data-bs-target="#js" type="button" role="tab">
                                JavaScript (HTMX)
                            </button>
                        </li>
                    </ul>
                    
                    <div class="tab-content mt-3" id="resultTabsContent">
                        <div class="tab-pane fade show active" id="templ" role="tabpanel">
                            <div class="code-section">
                                <div class="code-header">
                                    <span>Templ шаблон</span>
                                    <button class="btn btn-sm btn-outline-secondary copy-btn" data-target="templ-code">Копировать</button>
                                </div>
                                <div class="code-container code-content">
                                    <pre><code id="templ-code" class="language-go">${data.templFile || '// Шаблон не был сгенерирован'}</code></pre>
                                </div>
                            </div>
                        </div>
                        
                        <div class="tab-pane fade" id="controller" role="tabpanel">
                            <div class="code-section">
                                <div class="code-header">
                                    <span>Go контроллер</span>
                                    <button class="btn btn-sm btn-outline-secondary copy-btn" data-target="controller-code">Копировать</button>
                                </div>
                                <div class="code-container code-content">
                                    <pre><code id="controller-code" class="language-go">${data.goController || '// Go контроллер не был сгенерирован'}</code></pre>
                                </div>
                            </div>
                        </div>
                        
                        <div class="tab-pane fade" id="js" role="tabpanel">
                            <div class="code-section">
                                <div class="code-header">
                                    <span>JavaScript (HTMX)</span>
                                    <button class="btn btn-sm btn-outline-secondary copy-btn" data-target="js-code">Копировать</button>
                                </div>
                                <div class="code-container code-content">
                                    <pre><code id="js-code" class="language-javascript">${data.htmxJS || '// JavaScript не был сгенерирован'}</code></pre>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            
            <div class="card mt-3">
                <div class="card-body">
                    <h5 class="card-title">Использование сконвертированного кода</h5>
                    <div class="accordion" id="usageAccordion">
                        <div class="accordion-item">
                            <h2 class="accordion-header">
                                <button class="accordion-button" type="button" data-bs-toggle="collapse" data-bs-target="#collapseOne">
                                    Установка зависимостей
                                </button>
                            </h2>
                            <div id="collapseOne" class="accordion-collapse collapse show" data-bs-parent="#usageAccordion">
                                <div class="accordion-body">
                                    <p>1. Установите библиотеку templ:</p>
                                    <div class="code-container mb-3">
                                        <pre><code class="language-bash">go install github.com/a-h/templ/cmd/templ@latest</code></pre>
                                    </div>
                                    
                                    <p>2. Если вы используете HTMX, добавьте его в ваш проект:</p>
                                    <div class="code-container">
                                        <pre><code class="language-html">&lt;script src="https://unpkg.com/htmx.org@1.9.5"&gt;&lt;/script&gt;</code></pre>
                                    </div>
                                </div>
                            </div>
                        </div>
                        
                        <div class="accordion-item">
                            <h2 class="accordion-header">
                                <button class="accordion-button collapsed" type="button" data-bs-toggle="collapse" data-bs-target="#collapseTwo">
                                    Структура проекта
                                </button>
                            </h2>
                            <div id="collapseTwo" class="accordion-collapse collapse" data-bs-parent="#usageAccordion">
                                <div class="accordion-body">
                                    <p>Рекомендуемая структура проекта:</p>
                                    <div class="code-container">
                                        <pre><code class="language-bash">your-project/
├── cmd/
│   └── server/
│       └── main.go           # Основная точка входа сервера
├── internal/
│   └── controllers/
│       └── component.go      # Go контроллеры
├── templates/
│   └── component.templ       # Templ шаблоны
└── static/
    └── js/
        └── component.js      # JavaScript для HTMX</code></pre>
                                    </div>
                                </div>
                            </div>
                        </div>
                        
                        <div class="accordion-item">
                            <h2 class="accordion-header">
                                <button class="accordion-button collapsed" type="button" data-bs-toggle="collapse" data-bs-target="#collapseThree">
                                    Компиляция и запуск
                                </button>
                            </h2>
                            <div id="collapseThree" class="accordion-collapse collapse" data-bs-parent="#usageAccordion">
                                <div class="accordion-body">
                                    <p>1. Скомпилируйте templ шаблоны:</p>
                                    <div class="code-container mb-3">
                                        <pre><code class="language-bash">templ generate</code></pre>
                                    </div>
                                    
                                    <p>2. Запустите сервер:</p>
                                    <div class="code-container">
                                        <pre><code class="language-bash">go run cmd/server/main.go</code></pre>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;

        return html;
    }

    // Инициализация Bootstrap табов
    function initBootstrapTabs() {
        if (window.bootstrap) {
            const triggerTabList = document.querySelectorAll('#resultTabs button');
            triggerTabList.forEach(function(triggerEl) {
                new bootstrap.Tab(triggerEl);
            });
        } else {
            // Fallback для случая, если Bootstrap JS не загружен
            document.querySelectorAll('#resultTabs button').forEach(function(button) {
                button.addEventListener('click', function(e) {
                    e.preventDefault();

                    // Убираем active со всех табов
                    document.querySelectorAll('#resultTabs button').forEach(function(btn) {
                        btn.classList.remove('active');
                    });

                    // Скрываем все панели
                    document.querySelectorAll('.tab-pane').forEach(function(pane) {
                        pane.classList.remove('show', 'active');
                    });

                    // Активируем текущий таб
                    button.classList.add('active');

                    // Показываем соответствующую панель
                    const targetId = button.getAttribute('data-bs-target');
                    const targetPane = document.querySelector(targetId);
                    if (targetPane) {
                        targetPane.classList.add('show', 'active');
                    }
                });
            });
        }
    }

    // Инициализация подсветки синтаксиса и кнопок копирования
    function initCodeHighlighting() {
        // Подсветка кода
        if (window.hljs) {
            document.querySelectorAll('pre code').forEach(function(block) {
                hljs.highlightElement(block);
            });
        }

        // Обработчики для кнопок копирования
        document.querySelectorAll('.copy-btn').forEach(function(button) {
            button.addEventListener('click', function() {
                const targetId = button.getAttribute('data-target');
                const codeBlock = document.getElementById(targetId);

                if (codeBlock) {
                    // Копируем текст
                    navigator.clipboard.writeText(codeBlock.textContent)
                        .then(function() {
                            // Показываем подтверждение
                            const originalText = button.textContent;
                            button.textContent = 'Скопировано!';

                            // Возвращаем исходный текст через 1.5 секунды
                            setTimeout(function() {
                                button.textContent = originalText;
                            }, 1500);
                        })
                        .catch(function(err) {
                            console.error('Ошибка копирования: ', err);
                        });
                }
            });
        });
    }

    // Инициализация Drag & Drop
    if (dragDropArea) {
        // Предотвращаем стандартное поведение перетаскивания файлов
        ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
            dragDropArea.addEventListener(eventName, function(e) {
                e.preventDefault();
                e.stopPropagation();
            }, false);
        });

        // Подсветка при перетаскивании файла над зоной
        ['dragenter', 'dragover'].forEach(eventName => {
            dragDropArea.addEventListener(eventName, function() {
                dragDropArea.classList.add('dragover');
            }, false);
        });

        // Удаление подсветки
        ['dragleave', 'drop'].forEach(eventName => {
            dragDropArea.addEventListener(eventName, function() {
                dragDropArea.classList.remove('dragover');
            }, false);
        });

        // Обработка сброшенного файла
        dragDropArea.addEventListener('drop', function(e) {
            if (e.dataTransfer.files.length) {
                fileInput.files = e.dataTransfer.files;
                fileInput.dispatchEvent(new Event('change'));
            }
        }, false);
    }

    // Загрузка примеров
    window.loadExample = function(name) {
        fetch(`/api/examples/${name}`)
            .then(response => {
                if (!response.ok) {
                    throw new Error('Пример не найден');
                }
                return response.text();
            })
            .then(code => {
                // Отображаем код примера
                sourceCode.textContent = code;
                if (window.hljs) {
                    hljs.highlightElement(sourceCode);
                }

                // Разблокируем кнопку конвертации
                convertBtn.disabled = false;

                // Обновляем информацию о файле
                dragDropArea.innerHTML = `<p>Загружен пример: ${name}</p>`;

                // Создаем виртуальный файл для отправки формой
                const file = new File([code], `${name}.tsx`, { type: 'application/typescript' });
                const dataTransfer = new DataTransfer();
                dataTransfer.items.add(file);
                fileInput.files = dataTransfer.files;
            })
            .catch(error => {
                console.error('Ошибка загрузки примера:', error);
                alert('Не удалось загрузить пример: ' + error.message);
            });
    };
});