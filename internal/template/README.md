# Template engine usage

This package provides a small template filling engine used by the project to render
Feishu card payloads from webhook data. The engine supports:

- simple placeholders: `{{key}}` and dotted paths `{{object.subkey}}`
- filters: `| length`, `| default('fallback')`
- simple conditionals: `{{#if expr}}...{{/if}}` (non-nested)

## Basic placeholders

Placeholders resolve into values from the `data` map passed to `FillTemplate`.
Examples:

- `{{sender.login}}` → resolves `data["sender"]["login"]`
- `{{issue.title}}` → resolves `data["issue"]["title"]`

## Filters

- `| length` — returns length for strings, arrays, maps; numeric/other values return a stringified length result.
- `| default('fallback')` — if value is missing or empty string, uses the provided fallback. The fallback may itself be an expression, e.g. `{{issue.link | default(issue.html_url | default(''))}}`.

Example:

```jsonc
{{issue.labels | length}} labels
{{sender.name | default('unknown')}}
```

## Conditionals: `{{#if ...}}{{/if}}`

A simple, non-nested `if` block is supported. Syntax:

```jsonc
{{#if some_expr}}
  ...content that may include placeholders...
{{/if}}
```

Behavior:

- The engine evaluates `some_expr` using the same expression language used by placeholders.
- If the expression is truthy, the inner content is kept and processed for placeholders.
- If falsy, the entire block is removed from the output.
- Blocks cannot be nested (the engine scans and replaces top-level `{{#if ...}} ... {{/if}}` blocks).

Truthiness rules (used by `{{#if}}`):

- `nil` / `null` → false
- `bool` → its boolean value
- `string` → false if empty, true otherwise
- `[]any` and `map[string]any` → true if length > 0
- numbers → false if "0", true otherwise
- other types → considered true

## Examples

1. Show labels only if present:

   ```jsonc
   {{#if issue_labels_joined}}**Labels:** {{issue_labels_joined}}{{/if}}
   ```

2. Use default fallback for missing user link:

   ```jsonc
   {{sender_link_md | default(sender.login)}}
   ```

3. Nested placeholders inside an `if` block will be evaluated when the block is kept:

   ```jsonc
   {{#if issue.body}}
   **Description:** {{issue.body | truncate(200)}}
   {{/if}}
   ```

## Limitations and future work

- `{{#if}}` blocks are not nested. If you need nested conditionals or `else`, we can extend the parser.
- No `for` loop construct yet; can be added if needed.
- The expression language is intentionally small and deterministic.

If you want me to extend the engine to support `{{else}}`, `{{#unless}}`, nested blocks, or loops, tell me which feature you'd like next and I will implement it.
