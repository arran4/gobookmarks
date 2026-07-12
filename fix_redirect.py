with open('cmd/gobookmarks/serve.go', 'r') as f:
    content = f.read()

import re

# Currently `page := r.PostFormValue("page")` is missing if it wasn't a post form value.
# Also we need to check r.URL.Query().Get("page")

new_code = """		u.Path = gobookmarks.TabPath(tab)
		page := r.PostFormValue("page")
		if page == "" {
			page = r.URL.Query().Get("page")
		}
		if v, ok := r.Context().Value(gobookmarks.ContextValues("redirectPage")).(string); ok {
			page = v
		}"""

content = content.replace('''		u.Path = gobookmarks.TabPath(tab)
		page := r.PostFormValue("page")
		if v, ok := r.Context().Value(gobookmarks.ContextValues("redirectPage")).(string); ok {
			page = v
		}''', new_code)

with open('cmd/gobookmarks/serve.go', 'w') as f:
    f.write(content)
