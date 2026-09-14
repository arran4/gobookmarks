1. **Fix `templates/tail.gohtml` Multiple Keystroke Bug**:
   - Use `document.createComment('search-placeholder')` instead of `<div>`.
   - Before extracting `li` to the global list, if `!li._searchPlaceholder`, insert the comment node before the `li` and set `li._searchPlaceholder = comment`.
   - In `resetResults()`:
     - Iterate through `globalList.children`.
     - For each `li` with `_searchPlaceholder`, call `li._searchPlaceholder.parentNode.insertBefore(li, li._searchPlaceholder)`.
     - Remove the `_searchPlaceholder` from the DOM and set `li._searchPlaceholder = null`.
   - This fixes the `DOMException` and ensures multiple keystrokes work.

2. **Update `search_js_test.go`**:
   - Create mock DOM with a more complex scenario involving adjacent matching nodes to properly test the DOM restoration.
   - Run multiple keystrokes:
     1. Search for "github"
     2. Update to "github issues" (changes match set)
     3. Clear search (verify original order restored)
     4. Search for "docs"
     5. Clear search
   - Assert at each step: no errors, correct list order, correct `searchResults` array.

3. Verify and test.
