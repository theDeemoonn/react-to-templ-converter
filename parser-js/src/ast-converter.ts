import * as babel from '@babel/core';

// Интерфейс для JSX элемента
interface JSXElementInfo {
    type: string;
    props: Record<string, any>;
    children: JSXElementInfo[];
}

/**
 * Преобразует JSX AST в структурированный объект
 */
export function transformJSX(node: babel.types.Node, sourceCode: string): JSXElementInfo | null {
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
                content: sourceCode.substring(node.expression.start as number, node.expression.end as number),
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
                content: sourceCode.substring(node.start as number, node.end as number),
            },
            children: [],
        };
    }

    // Логическое выражение (например, condition && <div>...</div>)
    if (babel.types.isLogicalExpression(node)) {
        return {
            type: 'expression',
            props: {
                content: sourceCode.substring(node.start as number, node.end as number),
            },
            children: [],
        };
    }

    return null;
}

/**
 * Получает имя JSX элемента
 */
function getJSXElementName(
    nameNode: babel.types.JSXIdentifier | babel.types.JSXMemberExpression | babel.types.JSXNamespacedName
): string {
    if (babel.types.isJSXIdentifier(nameNode)) {
        return nameNode.name;
    } else if (babel.types.isJSXMemberExpression(nameNode)) {
        return `${getJSXElementName(nameNode.object)}.${nameNode.property.name}`;
    } else if (babel.types.isJSXNamespacedName(nameNode)) {
        return `${nameNode.namespace.name}:${nameNode.name.name}`;
    }

    return 'unknown';
}

/**
 * Извлекает атрибуты JSX элемента
 */
function extractJSXAttributes(
    attributes: Array<babel.types.JSXAttribute | babel.types.JSXSpreadAttribute>,
    sourceCode: string
): Record<string, any> {
    const props: Record<string, any> = {};

    attributes.forEach(attr => {
        if (babel.types.isJSXAttribute(attr)) {
            const name = (attr.name.name as string);

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
                } else {
                    props[name] = {
                        type: 'expression',
                        code: sourceCode.substring(
                            attr.value.expression.start as number,
                            attr.value.expression.end as number
                        ),
                    };
                }
                return;
            }

            // Вложенный JSX (например, children={<div>...</div>})
            if (babel.types.isJSXElement(attr.value) || babel.types.isJSXFragment(attr.value)) {
                props[name] = transformJSX(attr.value, sourceCode);
                return;
            }
        } else if (babel.types.isJSXSpreadAttribute(attr)) {
            // Spread атрибуты ({...props})
            props[`__spread__${attr.argument.start}`] = {
                type: 'spread',
                code: sourceCode.substring(attr.argument.start as number, attr.argument.end as number),
            };
        }
    });

    return props;
}

/**
 * Извлекает дочерние элементы JSX
 */
function extractJSXChildren(
    children: Array<babel.types.Node>,
    sourceCode: string
): JSXElementInfo[] {
    const result: JSXElementInfo[] = [];

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
        } else if (babel.types.isJSXElement(child) || babel.types.isJSXFragment(child)) {
            const transformed = transformJSX(child, sourceCode);
            if (transformed) {
                result.push(transformed);
            }
        } else if (babel.types.isJSXExpressionContainer(child)) {
            if (!babel.types.isJSXEmptyExpression(child.expression)) {
                result.push({
                    type: 'expression',
                    props: {
                        content: sourceCode.substring(
                            child.expression.start as number,
                            child.expression.end as number
                        ),
                    },
                    children: [],
                });
            }
        } else if (babel.types.isJSXSpreadChild && babel.types.isJSXSpreadChild(child)) {
            result.push({
                type: 'spread',
                props: {
                    content: sourceCode.substring(
                        child.expression.start as number,
                        child.expression.end as number
                    ),
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
export function processArrayMapping(node: babel.types.CallExpression, sourceCode: string): JSXElementInfo | null {
    // Проверяем, что это вызов метода map
    if (!babel.types.isMemberExpression(node.callee) ||
        !babel.types.isIdentifier(node.callee.property) ||
        node.callee.property.name !== 'map') {
        return null;
    }

    // Получаем массив, к которому применяется map
    const arrayCode = sourceCode.substring(
        node.callee.object.start as number,
        node.callee.object.end as number
    );

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
                if (foundReturn) return;
                if (path.node.argument &&
                    (babel.types.isJSXElement(path.node.argument) ||
                        babel.types.isJSXFragment(path.node.argument) ||
                        babel.types.isJSXExpressionContainer(path.node.argument))) {

                    returnJSX = transformJSX(path.node.argument, sourceCode);
                    foundReturn = true;
                }
            }
        }, { scopeToSkip: null } as any);
    } else if (babel.types.isJSXElement(callback.body) ||
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
export default {
    transformJSX,
    processArrayMapping,
};