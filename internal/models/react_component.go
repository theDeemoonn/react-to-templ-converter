package models

import "fmt"

// ReactComponent представляет структуру React компонента
type ReactComponent struct {
	Name      string                 `json:"name"`
	Props     []PropDefinition       `json:"props"`
	State     []StateDefinition      `json:"state"`
	Effects   []EffectDefinition     `json:"effects"`
	Callbacks []CallbackDefinition   `json:"callbacks"`
	Refs      []RefDefinition        `json:"refs"`
	JSX       *JSXElement            `json:"jsx"`
	Imports   []ImportDefinition     `json:"imports,omitempty"`
	Exports   map[string]interface{} `json:"exports,omitempty"`
}

// PropDefinition описывает пропс компонента
type PropDefinition struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"defaultValue,omitempty"`
}

// StateDefinition описывает состояние компонента (useState)
type StateDefinition struct {
	Name         string      `json:"name"`
	Setter       string      `json:"setter"`
	Type         string      `json:"type,omitempty"`
	InitialValue interface{} `json:"initialValue,omitempty"`
}

// EffectDefinition описывает эффект компонента (useEffect)
type EffectDefinition struct {
	Body         string   `json:"body"`
	Dependencies []string `json:"dependencies"`
}

// CallbackDefinition описывает колбэк компонента (useCallback)
type CallbackDefinition struct {
	Name         string   `json:"name"`
	Body         string   `json:"body"`
	Dependencies []string `json:"dependencies"`
}

// RefDefinition описывает ref компонента (useRef)
type RefDefinition struct {
	Name         string      `json:"name"`
	InitialValue interface{} `json:"initialValue,omitempty"`
}

// ImportDefinition описывает импорт в компоненте
type ImportDefinition struct {
	Source   string   `json:"source"`
	Defaults string   `json:"default,omitempty"`
	Named    []string `json:"named,omitempty"`
}

// JSXElement представляет JSX элемент в React компоненте
type JSXElement struct {
	Type     string                 `json:"type"`
	Props    map[string]interface{} `json:"props,omitempty"`
	Children []*JSXElement          `json:"children,omitempty"`
}

// Clone создает глубокую копию компонента
func (c *ReactComponent) Clone() *ReactComponent {
	if c == nil {
		return nil
	}

	clone := &ReactComponent{
		Name:      c.Name,
		Imports:   make([]ImportDefinition, len(c.Imports)),
		Props:     make([]PropDefinition, len(c.Props)),
		State:     make([]StateDefinition, len(c.State)),
		Effects:   make([]EffectDefinition, len(c.Effects)),
		Callbacks: make([]CallbackDefinition, len(c.Callbacks)),
		Refs:      make([]RefDefinition, len(c.Refs)),
	}

	// Копирование импортов
	for i, imp := range c.Imports {
		clone.Imports[i] = ImportDefinition{
			Source:   imp.Source,
			Defaults: imp.Defaults,
			Named:    make([]string, len(imp.Named)),
		}
		copy(clone.Imports[i].Named, imp.Named)
	}

	// Копирование пропсов
	for i, prop := range c.Props {
		clone.Props[i] = PropDefinition{
			Name:         prop.Name,
			Type:         prop.Type,
			Required:     prop.Required,
			DefaultValue: prop.DefaultValue,
		}
	}

	// Копирование состояний
	for i, state := range c.State {
		clone.State[i] = StateDefinition{
			Name:         state.Name,
			Setter:       state.Setter,
			Type:         state.Type,
			InitialValue: state.InitialValue,
		}
	}

	// Копирование эффектов
	for i, effect := range c.Effects {
		clone.Effects[i] = EffectDefinition{
			Body:         effect.Body,
			Dependencies: make([]string, len(effect.Dependencies)),
		}
		copy(clone.Effects[i].Dependencies, effect.Dependencies)
	}

	// Копирование колбэков
	for i, callback := range c.Callbacks {
		clone.Callbacks[i] = CallbackDefinition{
			Name:         callback.Name,
			Body:         callback.Body,
			Dependencies: make([]string, len(callback.Dependencies)),
		}
		copy(clone.Callbacks[i].Dependencies, callback.Dependencies)
	}

	// Копирование refs
	for i, ref := range c.Refs {
		clone.Refs[i] = RefDefinition{
			Name:         ref.Name,
			InitialValue: ref.InitialValue,
		}
	}

	// Копирование JSX (рекурсивно)
	if c.JSX != nil {
		clone.JSX = c.JSX.Clone()
	}

	// Копирование экспортов
	if c.Exports != nil {
		clone.Exports = make(map[string]interface{})
		for key, value := range c.Exports {
			clone.Exports[key] = value
		}
	}

	return clone
}

// Clone создает глубокую копию JSX элемента
func (j *JSXElement) Clone() *JSXElement {
	if j == nil {
		return nil
	}

	clone := &JSXElement{
		Type:  j.Type,
		Props: make(map[string]interface{}),
	}

	// Копирование свойств
	for key, value := range j.Props {
		clone.Props[key] = value
	}

	// Копирование дочерних элементов (рекурсивно)
	if len(j.Children) > 0 {
		clone.Children = make([]*JSXElement, len(j.Children))
		for i, child := range j.Children {
			clone.Children[i] = child.Clone()
		}
	}

	return clone
}

// Validate проверяет структуру компонента на корректность
func (c *ReactComponent) Validate() []string {
	var errors []string

	// Проверка имени компонента
	if c.Name == "" {
		errors = append(errors, "Имя компонента не может быть пустым")
	}

	// Проверка JSX
	if c.JSX == nil {
		errors = append(errors, "JSX не может быть пустым")
	}

	// Проверка пропсов
	for i, prop := range c.Props {
		if prop.Name == "" {
			errors = append(errors, fmt.Sprintf("Prop #%d: имя не может быть пустым", i+1))
		}
	}

	// Проверка состояний
	for i, state := range c.State {
		if state.Name == "" {
			errors = append(errors, fmt.Sprintf("State #%d: имя не может быть пустым", i+1))
		}
		if state.Setter == "" {
			errors = append(errors, fmt.Sprintf("State #%d: setter не может быть пустым", i+1))
		}
	}

	return errors
}
