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
exports.transformJSX = transformJSX;
exports.processArrayMapping = processArrayMapping;
const babel = __importStar(require("@babel/core"));
/**
 * Преобразует JSX AST в структурированный объект
 */
function transformJSX(node, sourceCode) {
    if (!node) {
        return null;
    }
    // JSX фрагмент: <>...</>
    if (babel.types.isJSXFragment(node)) {
        return {
            type: 'Fragment',
            props: {},
            children: extractJSXChildren(node.children, sourceCode),
        };
    }
    // JSX элемент: <div>...</div>
    if (babel.types.isJSXElement(node)) {
        const element = node.openingElement;
        // Получаем имя тега или компонента
        const tagName = getJSXElementName(element.name);
        // Извлекаем атрибуты (props)
        const props = extractJSXAttributes(element.attributes, sourceCode);
        // Извлекаем дочерние элементы
        const children = extractJSXChildren(node.children, sourceCode);
        return {
            type: tagName,
            props,
            children,
        };
    }
    // JSX выражение (например, {condition && <div>...</div>})
    if (babel.types.isJSXExpressionContainer(node)) {
        return {
            type: 'expression',
            props: {
                content: sourceCode.substring(node.expression.start, node.expression.end),
            },
            children: [],
        };
    }
    // JSX текст
    if (babel.types.isJSXText(node)) {
        const text = node.value.trim();
        if (text === '') {
            return null;
        }
        return {
            type: 'text',
            props: {
                content: text,
            },
            children: [],
        };
    }
    // Другие типы узлов, которые могут оказаться при неявном возврате JSX
    if (babel.types.isConditionalExpression(node)) {
        return {
            type: 'expression',
            props: {
                content: sourceCode.substring(node.start, node.end),
            },
            children: [],
        };
    }
    // Логическое выражение (например, condition && <div>...</div>)
    if (babel.types.isLogicalExpression(node)) {
        return {
            type: 'expression',
            props: {
                content: sourceCode.substring(node.start, node.end),
            },
            children: [],
        };
    }
    return null;
}
/**
 * Получает имя JSX элемента
 */
function getJSXElementName(nameNode) {
    if (babel.types.isJSXIdentifier(nameNode)) {
        return nameNode.name;
    }
    else if (babel.types.isJSXMemberExpression(nameNode)) {
        return `${getJSXElementName(nameNode.object)}.${nameNode.property.name}`;
    }
    else if (babel.types.isJSXNamespacedName(nameNode)) {
        return `${nameNode.namespace.name}:${nameNode.name.name}`;
    }
    return 'unknown';
}
/**
 * Извлекает атрибуты JSX элемента
 */
function extractJSXAttributes(attributes, sourceCode) {
    const props = {};
    attributes.forEach(attr => {
        if (babel.types.isJSXAttribute(attr)) {
            const name = attr.name.name;
            // Атрибут без значения (например, disabled)
            if (attr.value === null) {
                props[name] = true;
                return;
            }
            // Строковое значение
            if (babel.types.isStringLiteral(attr.value)) {
                props[name] = attr.value.value;
                return;
            }
            // Выражение в фигурных скобках
            if (babel.types.isJSXExpressionContainer(attr.value)) {
                if (babel.types.isJSXEmptyExpression(attr.value.expression)) {
                    props[name] = null;
                }
                else {
                    props[name] = {
                        type: 'expression',
                        code: sourceCode.substring(attr.value.expression.start, attr.value.expression.end),
                    };
                }
                return;
            }
            // Вложенный JSX (например, children={<div>...</div>})
            if (babel.types.isJSXElement(attr.value) || babel.types.isJSXFragment(attr.value)) {
                props[name] = transformJSX(attr.value, sourceCode);
                return;
            }
        }
        else if (babel.types.isJSXSpreadAttribute(attr)) {
            // Spread атрибуты ({...props})
            props[`__spread__${attr.argument.start}`] = {
                type: 'spread',
                code: sourceCode.substring(attr.argument.start, attr.argument.end),
            };
        }
    });
    return props;
}
/**
 * Извлекает дочерние элементы JSX
 */
function extractJSXChildren(children, sourceCode) {
    const result = [];
    children.forEach(child => {
        if (babel.types.isJSXText(child)) {
            // Пропускаем пустые текстовые узлы (только пробелы и переносы строк)
            const text = child.value.trim();
            if (text === '') {
                return;
            }
            result.push({
                type: 'text',
                props: {
                    content: text,
                },
                children: [],
            });
        }
        else if (babel.types.isJSXElement(child) || babel.types.isJSXFragment(child)) {
            const transformed = transformJSX(child, sourceCode);
            if (transformed) {
                result.push(transformed);
            }
        }
        else if (babel.types.isJSXExpressionContainer(child)) {
            if (!babel.types.isJSXEmptyExpression(child.expression)) {
                result.push({
                    type: 'expression',
                    props: {
                        content: sourceCode.substring(child.expression.start, child.expression.end),
                    },
                    children: [],
                });
            }
        }
        else if (babel.types.isJSXSpreadChild && babel.types.isJSXSpreadChild(child)) {
            result.push({
                type: 'spread',
                props: {
                    content: sourceCode.substring(child.expression.start, child.expression.end),
                },
                children: [],
            });
        }
    });
    return result;
}
/**
 * Обрабатывает выражение для отображения списка элементов (map)
 */
function processArrayMapping(node, sourceCode) {
    // Проверяем, что это вызов метода map
    if (!babel.types.isMemberExpression(node.callee) ||
        !babel.types.isIdentifier(node.callee.property) ||
        node.callee.property.name !== 'map') {
        return null;
    }
    // Получаем массив, к которому применяется map
    const arrayCode = sourceCode.substring(node.callee.object.start, node.callee.object.end);
    // Получаем callback функцию
    if (node.arguments.length === 0 ||
        (!babel.types.isArrowFunctionExpression(node.arguments[0]) &&
            !babel.types.isFunctionExpression(node.arguments[0]))) {
        return null;
    }
    const callback = node.arguments[0];
    // Получаем параметры callback функции
    let itemParam = '';
    let indexParam = '';
    if (callback.params.length > 0 && babel.types.isIdentifier(callback.params[0])) {
        itemParam = callback.params[0].name;
    }
    if (callback.params.length > 1 && babel.types.isIdentifier(callback.params[1])) {
        indexParam = callback.params[1].name;
    }
    // Получаем возвращаемый JSX
    let returnJSX = null;
    if (babel.types.isBlockStatement(callback.body)) {
        // Для функций с блоком кода ищем return statement
        let foundReturn = false;
        babel.traverse(callback.body, {
            ReturnStatement(path) {
                if (foundReturn)
                    return;
                if (path.node.argument &&
                    (babel.types.isJSXElement(path.node.argument) ||
                        babel.types.isJSXFragment(path.node.argument) ||
                        babel.types.isJSXExpressionContainer(path.node.argument))) {
                    returnJSX = transformJSX(path.node.argument, sourceCode);
                    foundReturn = true;
                }
            }
        }, { scopeToSkip: null });
    }
    else if (babel.types.isJSXElement(callback.body) ||
        babel.types.isJSXFragment(callback.body) ||
        babel.types.isJSXExpressionContainer(callback.body)) {
        // Для стрелочных функций с неявным возвратом
        returnJSX = transformJSX(callback.body, sourceCode);
    }
    if (!returnJSX) {
        return null;
    }
    // Создаем результат
    return {
        type: 'mapping',
        props: {
            array: arrayCode,
            item: itemParam,
            index: indexParam,
            template: returnJSX,
        },
        children: [],
    };
}
// Экспортируем функции для использования в index.js
exports.default = {
    transformJSX,
    processArrayMapping,
};
//# sourceMappingURL=ast-converter.js.map