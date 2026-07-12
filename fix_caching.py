# The user's second bug is: "Sometimes editing a link returns the older content rather than the new content and you have to manually refresh"
# The request cache is properly invalidated in UpdateBookmarks and CreateBookmarks.
# What else could cause this?
# Ah, maybe the URL parameter preservation is missing the cache-busting? No, there is no cache busting URL parameter.
# The user might be referring to `TaskDoneAutoRefreshPage`.
# Since we removed the `edit` query param, we also removed the redirect that clears the PRG pattern.
# Actually, the user says "returns the older content rather than the new content and you have to manually refresh"
# We removed `TaskDoneAutoRefreshPage` from the POST routes!
# `TaskDoneAutoRefreshPage` was responsible for telling the client to refresh the page after 1 second, or displaying "Done refreshing".
# Wait, no. `redirectToHandlerBranchToRef` does a `http.StatusSeeOther` (303) redirect to the new page.
# If they see older content, it could be the browser caching the 303 redirect target (the GET request).
