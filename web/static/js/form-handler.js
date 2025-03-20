// File upload and form handler for the React-to-Templ converter
document.addEventListener('DOMContentLoaded', function() {
    const fileInput = document.getElementById('file-input');
    const convertBtn = document.getElementById('convert-btn');
    const dragDropArea = document.getElementById('drag-drop-area');
    const sourceCode = document.getElementById('source-code');
    const form = document.getElementById('upload-form');

    // Enable convert button when a file is selected
    fileInput.addEventListener('change', function() {
        if (fileInput.files.length > 0) {
            // Enable the convert button
            convertBtn.disabled = false;

            // Display the file name in the drag-drop area
            dragDropArea.innerHTML = `<p>Выбран файл: ${fileInput.files[0].name}</p>`;

            // Read and display the file content in the source code area
            const reader = new FileReader();
            reader.onload = function(e) {
                sourceCode.textContent = e.target.result;
                // Highlight the code if hljs is available
                if (window.hljs) {
                    hljs.highlightElement(sourceCode);
                }
            };
            reader.readAsText(fileInput.files[0]);
        } else {
            convertBtn.disabled = true;
            dragDropArea.innerHTML = `
                <p>Перетащите файл сюда или кликните для выбора</p>
                <p class="text-muted small">Поддерживаемые форматы: .tsx, .jsx, .ts, .js</p>
            `;
            sourceCode.textContent = 'Здесь будет исходный код React компонента';
        }
    });

    // Drag and drop functionality
    dragDropArea.addEventListener('dragover', function(e) {
        e.preventDefault();
        dragDropArea.classList.add('dragover');
    });

    dragDropArea.addEventListener('dragleave', function() {
        dragDropArea.classList.remove('dragover');
    });

    dragDropArea.addEventListener('drop', function(e) {
        e.preventDefault();
        dragDropArea.classList.remove('dragover');

        if (e.dataTransfer.files.length > 0) {
            fileInput.files = e.dataTransfer.files;
            // Trigger the change event manually
            fileInput.dispatchEvent(new Event('change'));
        }
    });

    // Handle form submission results
    document.body.addEventListener('htmx:afterSwap', function(event) {
        if (event.detail.target.id === 'result-container') {
            // Initialize Bootstrap tabs
            if (window.bootstrap && document.getElementById('resultTabs')) {
                const tabElements = document.querySelectorAll('#resultTabs [data-bs-toggle="tab"]');
                tabElements.forEach(function(tab) {
                    new bootstrap.Tab(tab);
                });
            }

            // Highlight code blocks
            if (window.hljs) {
                document.querySelectorAll('pre code').forEach(function(block) {
                    hljs.highlightElement(block);
                });
            }

            // Setup copy buttons
            document.querySelectorAll('.copy-btn').forEach(function(btn) {
                btn.addEventListener('click', function() {
                    const targetId = btn.getAttribute('data-target');
                    const codeBlock = document.getElementById(targetId);
                    if (codeBlock) {
                        navigator.clipboard.writeText(codeBlock.textContent).then(function() {
                            const originalText = btn.textContent;
                            btn.textContent = 'Скопировано!';
                            setTimeout(function() {
                                btn.textContent = originalText;
                            }, 1500);
                        });
                    }
                });
            });
        }
    });

    // Function to load example components
    window.loadExample = function(name) {
        fetch(`/api/examples/${name}`)
            .then(response => {
                if (!response.ok) {
                    throw new Error('Пример не найден');
                }
                return response.text();
            })
            .then(content => {
                sourceCode.textContent = content;
                // Highlight the code
                if (window.hljs) {
                    hljs.highlightElement(sourceCode);
                }

                // Enable the convert button
                convertBtn.disabled = false;

                // Update drag-drop area
                dragDropArea.innerHTML = `<p>Загружен пример: ${name}</p>`;

                // Create a file object to attach to the form for submission
                const file = new File([content], `${name}.tsx`, { type: 'text/typescript' });
                const dataTransfer = new DataTransfer();
                dataTransfer.items.add(file);
                fileInput.files = dataTransfer.files;
            })
            .catch(error => {
                console.error('Error loading example:', error);
                alert('Ошибка загрузки примера: ' + error.message);
            });
    };
});