import re

with open('cmd/gobookmarks/serve.go', 'r') as f:
    content = f.read()

# remove unused redirectToHandlerTabPage
content = re.sub(r'func redirectToHandlerTabPage\(toURL string\) func\(http\.ResponseWriter, \*http\.Request\) \{[\s\S]*?\n\}\n', '', content)

with open('cmd/gobookmarks/serve.go', 'w') as f:
    f.write(content)
