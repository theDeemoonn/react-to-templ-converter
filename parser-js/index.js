// index.js - Точка входа для Node.js парсера
const fs = require('fs');
const path = require('path');
const parser = require('./dist/parser'); // Используем скомпилированный JavaScript
const http = require('http');

// Порт для HTTP сервера парсера
const PORT = process.env.PARSER_PORT || 3001;

// Создаем HTTP сервер для обработки запросов от Go
const server = http.createServer((req, res) => {
    if (req.method === 'POST' && req.url === '/parse') {
        let body = '';

        req.on('data', chunk => {
            body += chunk.toString();
        });

        req.on('end', () => {
            try {
                const requestData = JSON.parse(body);
                const code = requestData.code;

                // Парсим React/TypeScript код
                const result = parser.parseReactComponent(code);

                // Отправляем результат обратно в Go
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify(result));
            } catch (error) {
                console.error('Ошибка парсинга:', error);
                res.writeHead(500, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ error: error.message }));
            }
        });
    } else {
        res.writeHead(404);
        res.end();
    }
});

server.listen(PORT, () => {
    console.log(`Парсер запущен на порту ${PORT}`);

    // Сигнализируем Go-процессу, что парсер готов (пишем в stdout)
    console.log('PARSER_READY');
});