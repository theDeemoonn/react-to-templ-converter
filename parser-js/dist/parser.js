"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.parseReactComponent = parseReactComponent;
const babel = __importStar(require("@babel/core"));
const babelPresetReact = __importStar(require("@babel/preset-react"));
const babelPresetTypeScript = __importStar(require("@babel/preset-typescript"));
const ast_converter_1 = require("./ast-converter");
/**
 * Парсит React компонент и возвращает его структуру
 */
function parseReactComponent(code) {
    try {
        // Шаг 1: Используем Babel для парсинга JSX/TSX в AST
        const babelResult = babel.transformSync(code, {
            presets: [
                babelPresetReact,
                [babelPresetTypeScript, { isTSX: true, allExtensions: true }]
            ],
            ast: true,
            code: false,
        });
        if (!babelResult || !babelResult.ast) {
            throw new Error('Не удалось распарсить код с помощью Babel');
        }
        // Шаг 2: Извлекаем информацию из AST
        const componentInfo = {
            name: '',
            props: [],
            state: [],
            effects: [],
            callbacks: [],
            refs: [],
            jsx: null,
            imports: [],
        };
        // Сохраняем исходный код для извлечения фрагментов кода
        const sourceCode = code;
        // Выполняем обход AST с помощью посетителей
        babel.traverse(babelResult.ast, {
            // Обработка импортов
            ImportDeclaration(path) {
                const importNode = path.node;
                const sourcePath = importNode.source.value;
                const importInfo = {
                    source: sourcePath,
                    named: []
                };
                // Обработка импортируемых элементов
                importNode.specifiers.forEach(specifier => {
                    if (babel.types.isImportDefaultSpecifier(specifier)) {
                        importInfo.default = specifier.local.name;
                    }
                    else if (babel.types.isImportSpecifier(specifier)) {
                        // Проверяем тип, прежде чем обращаться к свойству 'name'
                        if (babel.types.isIdentifier(specifier.imported)) {
                            importInfo.named.push(specifier.imported.name);
                        }
                        else if (babel.types.isStringLiteral(specifier.imported)) {
                            // Если это строковый литерал, используем его значение
                            importInfo.named.push(specifier.imported.value);
                        }
                    }
                });
                componentInfo.imports.push(importInfo);
            },
            // Поиск функциональных компонентов
            FunctionDeclaration(path) {
                if (isReactComponent(path.node)) {
                    // Найден функциональный компонент
                    componentInfo.name = path.node.id.name;
                    extractPropsFromFunction(path, componentInfo, sourceCode);
                    extractJSXFromFunction(path, componentInfo, sourceCode);
                }
            },
            // Поиск стрелочных функций компонентов
            VariableDeclarator(path) {
                if (path.node.init &&
                    (babel.types.isArrowFunctionExpression(path.node.init) ||
                        babel.types.isFunctionExpression(path.node.init)) &&
                    babel.types.isIdentifier(path.node.id)) {
                    componentInfo.name = path.node.id.name;
                    if (isReactComponent(path.node.init)) {
                        // Найден компонент как стрелочная функция
                        extractPropsFromFunction(path, componentInfo, sourceCode);
                        extractJSXFromFunction(path, componentInfo, sourceCode);
                    }
                }
            },
            // Поиск вызовов хуков (useState, useEffect, useRef, useCallback)
            CallExpression(path) {
                if (babel.types.isIdentifier(path.node.callee)) {
                    const calleeName = path.node.callee.name;
                    // Обработка useState
                    if (calleeName === 'useState') {
                        extractUseState(path, componentInfo, sourceCode);
                    }
                    // Обработка useEffect
                    else if (calleeName === 'useEffect') {
                        extractUseEffect(path, componentInfo, sourceCode);
                    }
                    // Обработка useCallback
                    else if (calleeName === 'useCallback') {
                        extractUseCallback(path, componentInfo, sourceCode);
                    }
                    // Обработка useRef
                    else if (calleeName === 'useRef') {
                        extractUseRef(path, componentInfo, sourceCode);
                    }
                }
            },
            // Обработка экспортов
            ExportNamedDeclaration(path) {
                if (!componentInfo.exports) {
                    componentInfo.exports = {};
                }
                // Обрабатываем именованные экспорты
                if (path.node.declaration) {
                    if (babel.types.isVariableDeclaration(path.node.declaration)) {
                        path.node.declaration.declarations.forEach(declaration => {
                            if (babel.types.isIdentifier(declaration.id)) {
                                componentInfo.exports[declaration.id.name] = true;
                            }
                        });
                    }
                    else if (babel.types.isFunctionDeclaration(path.node.declaration) &&
                        path.node.declaration.id) {
                        componentInfo.exports[path.node.declaration.id.name] = true;
                    }
                }
                // Обрабатываем экспорты спецификаторов
                path.node.specifiers.forEach(specifier => {
                    if (babel.types.isExportSpecifier(specifier)) {
                        // Проверяем тип, прежде чем обращаться к свойству 'name'
                        if (babel.types.isIdentifier(specifier.exported)) {
                            componentInfo.exports[specifier.exported.name] = true;
                        }
                        else if (babel.types.isStringLiteral(specifier.exported)) {
                            componentInfo.exports[specifier.exported.value] = true;
                        }
                    }
                });
            },
            // Обработка экспорта по умолчанию
            ExportDefaultDeclaration(path) {
                if (!componentInfo.exports) {
                    componentInfo.exports = {};
                }
                if (babel.types.isIdentifier(path.node.declaration)) {
                    componentInfo.exports.default = path.node.declaration.name;
                }
                else {
                    componentInfo.exports.default = true;
                }
            }
        });
        return componentInfo;
    }
    catch (error) {
        console.error('Ошибка парсинга React компонента:', error);
        throw error;
    }
}
/**
 * Проверяет, является ли узел функцией React компонента
 */
function isReactComponent(node) {
    // Проверяем наличие JSX в возвращаемом значении функции
    let hasJSXReturn = false;
    if (babel.types.isFunctionDeclaration(node) ||
        babel.types.isArrowFunctionExpression(node) ||
        babel.types.isFunctionExpression(node)) {
        // Для функций с блоком кода
        if (babel.types.isBlockStatement(node.body)) {
            let found = false;
            babel.traverse(node.body, {
                ReturnStatement(path) {
                    if (found)
                        return;
                    if (path.node.argument && isJSX(path.node.argument)) {
                        hasJSXReturn = true;
                        found = true;
                    }
                }
            }, { scopeToSkip: null });
        }
        // Для стрелочных функций с неявным возвратом
        else if (isJSX(node.body)) {
            hasJSXReturn = true;
        }
    }
    return hasJSXReturn;
}
/**
 * Проверяет, является ли узел JSX элементом
 */
function isJSX(node) {
    return babel.types.isJSXElement(node) ||
        babel.types.isJSXFragment(node) ||
        babel.types.isJSXText(node);
}
/**
 * Извлекает информацию о пропсах из функции компонента
 */
function extractPropsFromFunction(path, componentInfo, sourceCode) {
    const node = path.node;
    // Получаем параметры функции
    let params;
    if (babel.types.isFunctionDeclaration(node)) {
        params = node.params;
    }
    else if (babel.types.isVariableDeclarator(node) &&
        (babel.types.isArrowFunctionExpression(node.init) ||
            babel.types.isFunctionExpression(node.init))) {
        params = node.init.params;
    }
    else {
        return;
    }
    if (!params || params.length === 0) {
        return;
    }
    const propsParam = params[0];
    // Обработка различных типов параметров пропсов
    if (babel.types.isObjectPattern(propsParam)) {
        // Деструктурированные пропсы { prop1, prop2 }
        propsParam.properties.forEach(prop => {
            if (babel.types.isObjectProperty(prop) &&
                babel.types.isIdentifier(prop.key)) {
                componentInfo.props.push({
                    name: prop.key.name,
                    required: true, // По умолчанию считаем обязательным
                    type: 'any', // Тип неизвестен из деструктуризации
                });
            }
            else if (babel.types.isRestElement(prop) &&
                babel.types.isIdentifier(prop.argument)) {
                // Обработка rest-параметра {...restProps}
                componentInfo.props.push({
                    name: prop.argument.name,
                    required: false,
                    type: 'object',
                });
            }
        });
    }
    else if (babel.types.isIdentifier(propsParam)) {
        // Поиск интерфейса или типа пропсов в коде
        extractPropsFromTypeAnnotation(propsParam, componentInfo, path);
    }
}
/**
 * Извлекает информацию о пропсах из аннотации типа
 */
function extractPropsFromTypeAnnotation(propsParam, componentInfo, path) {
    // Получаем имя типа пропсов из аннотации
    let propsTypeName = null;
    if (propsParam.typeAnnotation &&
        babel.types.isTSTypeAnnotation(propsParam.typeAnnotation) &&
        babel.types.isTSTypeReference(propsParam.typeAnnotation.typeAnnotation) &&
        babel.types.isIdentifier(propsParam.typeAnnotation.typeAnnotation.typeName)) {
        propsTypeName = propsParam.typeAnnotation.typeAnnotation.typeName.name;
    }
    if (!propsTypeName) {
        // Если тип не указан явно, пробуем найти по соглашению об именовании
        propsTypeName = `${componentInfo.name}Props`;
    }
    // Ищем определение интерфейса пропсов в файле
    let foundInterface = false;
    const program = getProgram(path);
    if (program) {
        babel.traverse(program.node, {
            TSInterfaceDeclaration(interfacePath) {
                if (interfacePath.node.id &&
                    interfacePath.node.id.name === propsTypeName) {
                    foundInterface = true;
                    // Получаем свойства интерфейса
                    interfacePath.node.body.body.forEach(property => {
                        if (babel.types.isTSPropertySignature(property) &&
                            babel.types.isIdentifier(property.key)) {
                            const propName = property.key.name;
                            const isOptional = !!property.optional;
                            let propType = 'any';
                            // Получаем тип свойства
                            if (property.typeAnnotation &&
                                property.typeAnnotation.typeAnnotation) {
                                propType = getTypeFromTSAnnotation(property.typeAnnotation.typeAnnotation);
                            }
                            componentInfo.props.push({
                                name: propName,
                                required: !isOptional,
                                type: propType,
                                defaultValue: undefined, // Значение по умолчанию нельзя определить из интерфейса
                            });
                        }
                    });
                }
            },
            // Также проверяем тип (type Props = {...})
            TSTypeAliasDeclaration(typePath) {
                if (typePath.node.id &&
                    typePath.node.id.name === propsTypeName) {
                    foundInterface = true;
                    // Обрабатываем тип в зависимости от его структуры
                    if (babel.types.isTSTypeLiteral(typePath.node.typeAnnotation)) {
                        typePath.node.typeAnnotation.members.forEach(member => {
                            if (babel.types.isTSPropertySignature(member) &&
                                babel.types.isIdentifier(member.key)) {
                                const propName = member.key.name;
                                const isOptional = !!member.optional;
                                let propType = 'any';
                                // Получаем тип свойства
                                if (member.typeAnnotation &&
                                    member.typeAnnotation.typeAnnotation) {
                                    propType = getTypeFromTSAnnotation(member.typeAnnotation.typeAnnotation);
                                }
                                componentInfo.props.push({
                                    name: propName,
                                    required: !isOptional,
                                    type: propType,
                                    defaultValue: undefined,
                                });
                            }
                        });
                    }
                }
            }
        }, { scopeToSkip: null });
    }
    // Если интерфейс не найден, добавляем props как единый объект
    if (!foundInterface && propsParam.name) {
        componentInfo.props.push({
            name: propsParam.name,
            required: true,
            type: 'object',
        });
    }
}
/**
 * Извлекает структуру JSX из функции компонента
 */
function extractJSXFromFunction(path, componentInfo, sourceCode) {
    let node;
    if (babel.types.isFunctionDeclaration(path.node)) {
        node = path.node;
    }
    else if (babel.types.isVariableDeclarator(path.node) &&
        (babel.types.isArrowFunctionExpression(path.node.init) ||
            babel.types.isFunctionExpression(path.node.init))) {
        node = path.node.init;
    }
    else {
        return;
    }
    // Для стрелочных функций с неявным возвратом
    if (babel.types.isArrowFunctionExpression(node) && isJSX(node.body)) {
        componentInfo.jsx = (0, ast_converter_1.transformJSX)(node.body, sourceCode);
        return;
    }
    // Для функций с блоком кода
    if (babel.types.isBlockStatement(node.body)) {
        let found = false;
        babel.traverse(node.body, {
            ReturnStatement(returnPath) {
                if (found)
                    return;
                if (returnPath.node.argument && isJSX(returnPath.node.argument)) {
                    componentInfo.jsx = (0, ast_converter_1.transformJSX)(returnPath.node.argument, sourceCode);
                    found = true;
                }
            }
        }, { scopeToSkip: null });
    }
}
/**
 * Извлекает вызов useState
 */
function extractUseState(path, componentInfo, sourceCode) {
    const variableDeclarator = path.findParent(p => babel.types.isVariableDeclarator(p.node));
    if (!variableDeclarator || !babel.types.isVariableDeclarator(variableDeclarator.node)) {
        return;
    }
    const declaration = variableDeclarator.node;
    // Проверяем, что переменная деструктурируется как массив [state, setState]
    if (babel.types.isArrayPattern(declaration.id) &&
        declaration.id.elements.length === 2 &&
        declaration.id.elements[0] && declaration.id.elements[1] &&
        babel.types.isIdentifier(declaration.id.elements[0]) &&
        babel.types.isIdentifier(declaration.id.elements[1])) {
        const stateVar = declaration.id.elements[0];
        const setter = declaration.id.elements[1];
        // Получаем начальное значение
        let initialValue = undefined;
        if (path.node.arguments.length > 0) {
            initialValue = getInitialStateValue(path.node.arguments[0], sourceCode);
        }
        // Получаем тип состояния
        let stateType = 'any';
        // Пытаемся определить тип из аргумента useState<T>
        if (path.node.typeParameters &&
            babel.types.isTSTypeParameterInstantiation(path.node.typeParameters) &&
            path.node.typeParameters.params.length > 0) {
            stateType = getTypeFromTSAnnotation(path.node.typeParameters.params[0]);
        }
        // Если тип не указан явно, пытаемся определить из начального значения
        else if (initialValue !== undefined) {
            stateType = getTypeFromValue(initialValue);
        }
        componentInfo.state.push({
            name: stateVar.name,
            setter: setter.name,
            type: stateType,
            initialValue,
        });
    }
}
/**
 * Извлекает вызов useEffect
 */
function extractUseEffect(path, componentInfo, sourceCode) {
    if (path.node.arguments.length === 0) {
        return;
    }
    // Получаем функцию эффекта (первый аргумент)
    const effectFunc = path.node.arguments[0];
    if (!babel.types.isArrowFunctionExpression(effectFunc) &&
        !babel.types.isFunctionExpression(effectFunc)) {
        return;
    }
    // Получаем тело эффекта
    let effectBody = '';
    if (babel.types.isBlockStatement(effectFunc.body)) {
        // Для функций с блоком кода
        effectBody = sourceCode.substring(effectFunc.body.start, effectFunc.body.end);
    }
    else {
        // Для стрелочных функций с неявным возвратом
        effectBody = sourceCode.substring(effectFunc.start + effectFunc.body.start, effectFunc.body.end);
    }
    // Получаем массив зависимостей (второй аргумент)
    const dependencies = [];
    if (path.node.arguments.length > 1 &&
        babel.types.isArrayExpression(path.node.arguments[1])) {
        path.node.arguments[1].elements.forEach(element => {
            if (babel.types.isIdentifier(element)) {
                dependencies.push(element.name);
            }
            else if (babel.types.isMemberExpression(element) &&
                babel.types.isIdentifier(element.object) &&
                babel.types.isIdentifier(element.property)) {
                dependencies.push(`${element.object.name}.${element.property.name}`);
            }
        });
    }
    componentInfo.effects.push({
        body: effectBody,
        dependencies,
    });
}
/**
 * Извлекает вызов useCallback
 */
function extractUseCallback(path, componentInfo, sourceCode) {
    if (path.node.arguments.length === 0) {
        return;
    }
    // Ищем, куда присваивается результат useCallback
    const variableDeclarator = path.findParent(p => babel.types.isVariableDeclarator(p.node));
    if (!variableDeclarator || !babel.types.isVariableDeclarator(variableDeclarator.node)) {
        return;
    }
    const declaration = variableDeclarator.node;
    if (!babel.types.isIdentifier(declaration.id)) {
        return;
    }
    const callbackName = declaration.id.name;
    // Получаем функцию колбэка (первый аргумент)
    const callbackFunc = path.node.arguments[0];
    if (!babel.types.isArrowFunctionExpression(callbackFunc) &&
        !babel.types.isFunctionExpression(callbackFunc)) {
        return;
    }
    // Получаем тело колбэка
    let callbackBody = '';
    if (babel.types.isBlockStatement(callbackFunc.body)) {
        // Для функций с блоком кода
        callbackBody = sourceCode.substring(callbackFunc.body.start, callbackFunc.body.end);
    }
    else {
        // Для стрелочных функций с неявным возвратом
        callbackBody = sourceCode.substring(callbackFunc.body.start, callbackFunc.body.end);
    }
    // Получаем массив зависимостей (второй аргумент)
    const dependencies = [];
    if (path.node.arguments.length > 1 &&
        babel.types.isArrayExpression(path.node.arguments[1])) {
        path.node.arguments[1].elements.forEach(element => {
            if (babel.types.isIdentifier(element)) {
                dependencies.push(element.name);
            }
            else if (babel.types.isMemberExpression(element) &&
                babel.types.isIdentifier(element.object) &&
                babel.types.isIdentifier(element.property)) {
                dependencies.push(`${element.object.name}.${element.property.name}`);
            }
        });
    }
    componentInfo.callbacks.push({
        name: callbackName,
        body: callbackBody,
        dependencies,
    });
}
/**
 * Извлекает вызов useRef
 */
function extractUseRef(path, componentInfo, sourceCode) {
    // Ищем, куда присваивается результат useRef
    const variableDeclarator = path.findParent(p => babel.types.isVariableDeclarator(p.node));
    if (!variableDeclarator || !babel.types.isVariableDeclarator(variableDeclarator.node)) {
        return;
    }
    const declaration = variableDeclarator.node;
    if (!babel.types.isIdentifier(declaration.id)) {
        return;
    }
    const refName = declaration.id.name;
    // Получаем начальное значение
    let initialValue = undefined;
    if (path.node.arguments.length > 0) {
        initialValue = getInitialStateValue(path.node.arguments[0], sourceCode);
    }
    componentInfo.refs.push({
        name: refName,
        initialValue,
    });
}
/**
 * Получает родительский узел Program
 */
function getProgram(path) {
    let current = path;
    while (current && !babel.types.isProgram(current.node)) {
        if (!current.parentPath) {
            return null;
        }
        current = current.parentPath;
    }
    return current;
}
/**
 * Получает строковое представление типа из аннотации TypeScript
 */
function getTypeFromTSAnnotation(typeAnnotation) {
    if (babel.types.isTSStringKeyword(typeAnnotation)) {
        return 'string';
    }
    else if (babel.types.isTSNumberKeyword(typeAnnotation)) {
        return 'number';
    }
    else if (babel.types.isTSBooleanKeyword(typeAnnotation)) {
        return 'boolean';
    }
    else if (babel.types.isTSArrayType(typeAnnotation)) {
        const elementType = getTypeFromTSAnnotation(typeAnnotation.elementType);
        return `Array<${elementType}>`;
    }
    else if (babel.types.isTSObjectKeyword(typeAnnotation)) {
        return 'object';
    }
    else if (babel.types.isTSFunctionType(typeAnnotation)) {
        return 'function';
    }
    else if (babel.types.isTSUnionType(typeAnnotation)) {
        return 'union';
    }
    else if (babel.types.isTSTypeReference(typeAnnotation) && babel.types.isIdentifier(typeAnnotation.typeName)) {
        return typeAnnotation.typeName.name;
    }
    return 'any';
}
/**
 * Получает начальное значение состояния из аргумента useState
 */
function getInitialStateValue(node, sourceCode) {
    if (babel.types.isStringLiteral(node)) {
        return node.value;
    }
    else if (babel.types.isNumericLiteral(node)) {
        return node.value;
    }
    else if (babel.types.isBooleanLiteral(node)) {
        return node.value;
    }
    else if (babel.types.isNullLiteral(node)) {
        return null;
    }
    else if (babel.types.isArrayExpression(node)) {
        if (node.elements.length === 0) {
            return [];
        }
        return sourceCode.substring(node.start, node.end);
    }
    else if (babel.types.isObjectExpression(node)) {
        if (node.properties.length === 0) {
            return {};
        }
        return sourceCode.substring(node.start, node.end);
    }
    // Для сложных выражений возвращаем исходный код
    return node.start !== undefined && node.end !== undefined
        ? sourceCode.substring(node.start, node.end)
        : undefined;
}
/**
 * Определяет тип на основе значения
 */
function getTypeFromValue(value) {
    if (value === null || value === undefined) {
        return 'any';
    }
    const type = typeof value;
    if (type === 'string') {
        return 'string';
    }
    else if (type === 'number') {
        return 'number';
    }
    else if (type === 'boolean') {
        return 'boolean';
    }
    else if (Array.isArray(value)) {
        return 'array';
    }
    else if (type === 'object') {
        return 'object';
    }
    else if (type === 'function') {
        return 'function';
    }
    return 'any';
}
// Экспортируем функцию для использования в index.js
exports.default = parseReactComponent;
//# sourceMappingURL=parser.js.map