with open('cmd/gobookmarks/serve.go', 'r') as f:
    content = f.read()

import re

# "When editing on any page but the first it doesn't redirect after updating the contents to the page you were on"
# In redirectToHandlerBranchToRef, tab is taken from gobookmarks.TabFromRequest(r), but `page` is taken from `r.PostFormValue("page")`.
# However, if we look at the forms, editPageForm doesn't send "page" explicitly unless it is in the URL or modal.
# It seems `page` might be empty.
# Let's check editPageForm.gohtml
