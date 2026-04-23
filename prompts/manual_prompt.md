Create a single image based on this prompt:

{{.Prompt}}

Target page shape: {{.PageFrame}}.
{{if .Style}}Visual style: {{.Style}}.
{{end}}{{if .Theme}}Theme context: {{.Theme}}.
{{end}}{{if .RenderingRequirement}}{{.RenderingRequirement}}
{{end}}No text, captions, logos, borders, panels, or UI elements.
The image must stay faithful to the prompt while incorporating the style and theme context above.{{if .RenderingRequirement}}{{.RenderingRequirementEnd}}{{end}}
