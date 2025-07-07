db = db.getSiblingDB('demo');   // Получаем доступ к базе
const fs = require('fs');       // Импортируем модуль 'fs' для чтения файла
const content = fs.readFileSync('/docker-entrypoint-initdb.d/recipes.json', 'utf8');    // Читаем JSON-файл как строку
const data = JSON.parse(content);   // Парсим JSON
db.recipes.insertMany(data);        // Вставляем в коллекцию
