// Package config содержит доменное представление разобранного конфига.
package config

import (
	"sort"
	"strings"
)

// Config — универсальное дерево конфигурации (результат парсинга JSON/YAML).
// Формат-независимое представление позволяет добавлять новые форматы
// парсеров, не трогая правила.
type Config struct {
	root any
	raw  []byte
}

// New оборачивает уже распарсенное дерево
func New(root any, raw []byte) *Config {
	return &Config{root: root, raw: raw}
}

// Raw возвращает исходные байты конфига (нужно, например, для отладки).
func (c *Config) Raw() []byte {
	return c.raw
}

// Root возвращает корень дерева.
func (c *Config) Root() any {
	return c.root
}

// Node — один узел дерева при обходе. Path — полный путь от корня до узла
// включительно (для корня и элементов массива последний компонент пуст).
type Node struct {
	Path  []string
	Key   string
	Value any
}

// DotPath возвращает путь узла через точку, например "log.level".
func (n Node) DotPath() string {
	return strings.Join(n.Path, ".")
}

// Walk обходит всё дерево конфигурации в глубину и вызывает fn для каждого узла
func (c *Config) Walk(fn func(Node)) {
	walk(nil, c.root, fn)
}

func walk(path []string, value any, fn func(Node)) {
	key := ""
	if len(path) > 0 {
		key = path[len(path)-1]
	}
	fn(Node{Path: path, Key: key, Value: value})

	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			walk(append(append([]string(nil), path...), k), v[k], fn)
		}
	case []any:
		for _, item := range v {
			walk(path, item, fn)
		}
	}
}

// FindKey ищет первое (в детерминированном порядке обхода) значение, чей ключ
// совпадает с одним из candidates (без учёта регистра). Возвращает путь и
// значение. Используется правилами, которым важно только имя ключа, а не его
// точное расположение в дереве.
func (c *Config) FindKey(candidates ...string) (Node, bool) {
	set := make(map[string]struct{}, len(candidates))
	for _, k := range candidates {
		set[strings.ToLower(k)] = struct{}{}
	}

	var found Node
	ok := false
	c.Walk(func(n Node) {
		if ok || n.Key == "" {
			return
		}
		if _, match := set[strings.ToLower(n.Key)]; match {
			found = n
			ok = true
		}
	})
	return found, ok
}

// AsString приводит value к строке, если это возможно.
func AsString(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

// AsBool приводит value к bool, если это возможно.
func AsBool(v any) (bool, bool) {
	b, ok := v.(bool)
	return b, ok
}
