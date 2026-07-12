import re

# `{{if page}}` evaluates to false if `page` is an integer 0, or it might just not evaluate the way they want if the template doesn't provide it properly.
# The user's second bug is:
# "Sometimes editing a link returns the older content rather than the new content and you have to manually refresh"

# This sounds like a caching issue with the request-scoped cache.
# In `core.go` or `provider_access.go`, did we remove `invalidateRequestCache`?
