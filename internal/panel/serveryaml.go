package panel

import (
	"bytes"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// topMap returns the top-level mapping node of a decoded YAML document.
func topMap(root *yaml.Node) *yaml.Node {
	if root == nil {
		return nil
	}
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		return root.Content[0]
	}
	if root.Kind == yaml.MappingNode {
		return root
	}
	return nil
}

// mapGet returns the value node for key in a mapping node, or nil.
func mapGet(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// mapSet sets a double-quoted scalar value for key in a mapping, adding the
// key/value pair if missing. Returns the value node.
func mapSet(m *yaml.Node, key, value string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			val := m.Content[i+1]
			val.Kind = yaml.ScalarNode
			val.Tag = "" // let the encoder infer str implicitly (avoid leaking !<str>)
			val.Value = value
			val.Style = yaml.DoubleQuotedStyle
			return val
		}
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "", Value: value, Style: yaml.DoubleQuotedStyle},
	)
	return m.Content[len(m.Content)-1]
}

// mapDelete removes key (and its value) from a mapping node, if present.
func mapDelete(m *yaml.Node, key string) {
	if m == nil || m.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return
		}
	}
}

// mapSetPlain is like mapSet but emits a plain (unquoted) scalar, for numeric
// values such as port and timeout.
func mapSetPlain(m *yaml.Node, key, value string) *yaml.Node {
	n := mapSet(m, key, value)
	n.Style = 0
	return n
}

// ensureMap returns the top-level mapping for key, creating it if absent.
func ensureMap(root *yaml.Node, key string) *yaml.Node {
	tm := topMap(root)
	if n := mapGet(tm, key); n != nil && n.Kind == yaml.MappingNode {
		return n
	}
	tm.Content = append(tm.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.MappingNode},
	)
	return tm.Content[len(tm.Content)-1]
}

// setTopLevelSequence sets a top-level key to a sequence of double-quoted string
// items, replacing any existing value (used for allowed_sources).
func setTopLevelSequence(root *yaml.Node, key string, values []string) {
	tm := topMap(root)
	var seq *yaml.Node
	for i := 0; i+1 < len(tm.Content); i += 2 {
		if tm.Content[i].Value == key {
			seq = tm.Content[i+1]
			break
		}
	}
	if seq == nil {
		tm.Content = append(tm.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			&yaml.Node{Kind: yaml.SequenceNode},
		)
		seq = tm.Content[len(tm.Content)-1]
	}
	seq.Kind = yaml.SequenceNode
	seq.Tag = ""
	seq.Content = nil
	for _, v := range values {
		seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: v, Style: yaml.DoubleQuotedStyle})
	}
}

// reorderPanel reorders a panel mapping's key/value pairs into a canonical order
// (enabled, username, password, password_hash, secret). Keys not listed keep
// their original relative order, appended after. Comments attached to nodes
// travel with their nodes. This guarantees password_hash is followed by secret
// so a HeadComment on password_hash renders as a stable standalone line.
func reorderPanel(panel *yaml.Node) {
	if panel == nil || panel.Kind != yaml.MappingNode {
		return
	}
	order := []string{"enabled", "username", "password", "password_hash", "secret"}
	keys := map[string]*yaml.Node{} // key value -> key node
	vals := map[string]*yaml.Node{} // key value -> value node
	var seen []string
	for i := 0; i+1 < len(panel.Content); i += 2 {
		k := panel.Content[i].Value
		if _, dup := keys[k]; !dup {
			seen = append(seen, k)
		}
		keys[k] = panel.Content[i]
		vals[k] = panel.Content[i+1]
	}
	inOrder := map[string]bool{}
	for _, o := range order {
		inOrder[o] = true
	}
	var extras []string
	for _, k := range seen {
		if !inOrder[k] {
			extras = append(extras, k)
		}
	}

	var content []*yaml.Node
	for _, o := range order {
		if _, ok := keys[o]; ok {
			content = append(content, keys[o], vals[o])
		}
	}
	for _, e := range extras {
		content = append(content, keys[e], vals[e])
	}
	panel.Content = content
}

// loadServerRoot decodes server.yaml into a yaml.Node tree (preserving comments).
func loadServerRoot(cfgDir string) (*yaml.Node, error) {
	data, err := os.ReadFile(filepath.Join(cfgDir, "server.yaml"))
	if err != nil {
		return nil, err
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	return &root, nil
}

// writeServerRoot encodes a node tree back to server.yaml atomically, using a
// 2-space indent to match the project's existing YAML style.
func writeServerRoot(cfgDir string, root *yaml.Node) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	writeMu.Lock()
	defer writeMu.Unlock()
	return atomicWriteFile(filepath.Join(cfgDir, "server.yaml"), buf.Bytes(), 0o644)
}
