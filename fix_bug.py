# The `edit-link` for a page goes to `/editPage?ref=...&tab=...&page=X`.
# When the user clicks this, it loads the modal. The modal's form action gets rewritten by JS to:
# `formActionUrl.searchParams.delete('modal'); formActionUrl.searchParams.set('from_modal', '1'); form.action = formActionUrl.toString();`
# So the form's `action` *does* have `?page=X`.
# However, `r.URL.Query().Get("page")` was what we added.
# Before we added it, `page := r.PostFormValue("page")` was used.
# But `editPageForm.gohtml` does this: `{{if page}}<input type=hidden name="page" value="{{page}}" />{{end}}`
# If `page` was passed properly to the template, it should have been rendered as a hidden input, and thus submitted as `PostFormValue("page")`.
# Did `editPageForm` not get the `page` var?
