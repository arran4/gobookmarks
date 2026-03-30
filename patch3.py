with open('templates/tail.gohtml', 'r') as f:
    content = f.read()

# We will replace the entire block we added with a cleaner set of functions.
old_block = """    if (!document.getElementById('edit-dialog')) {
        var dialog = document.createElement('dialog');
        dialog.id = 'edit-dialog';
        dialog.className = 'edit-dialog';
        dialog.innerHTML = `
            <div class="edit-dialog-header">
                <h2 id="edit-dialog-title" style="margin: 0;">Edit</h2>
                <button id="edit-dialog-close">&times;</button>
            </div>
            <div id="edit-dialog-content"></div>
        `;
        document.body.appendChild(dialog);

        document.getElementById('edit-dialog-close').addEventListener('click', function(e) {
            e.preventDefault();
            dialog.close();
        });
    }

    document.body.addEventListener('click', function(e) {
        var link = e.target.closest('a');
        if (!link) return;
        var href = link.getAttribute('href');
        if (!href) return;

        var u;
        try { u = new URL(href, window.location.origin); } catch(err) { return; }
        var path = u.pathname;

        if (path === '/editCategory' || path === '/editPage' || path === '/editTab' || path === '/edit' || path === '/addCategory' || path === '/addPage' || path === '/addTab') {
            if (e.ctrlKey || e.metaKey || e.shiftKey || link.target === '_blank') return;
            e.preventDefault();

            var dialog = document.getElementById('edit-dialog');
            var contentDiv = document.getElementById('edit-dialog-content');
            var titleEl = document.getElementById('edit-dialog-title');

            if (path === '/editCategory') titleEl.textContent = 'Edit Category';
            else if (path === '/editPage') titleEl.textContent = 'Edit Page';
            else if (path === '/editTab') titleEl.textContent = 'Edit Tab';
            else if (path === '/edit') titleEl.textContent = 'Edit All';
            else if (path === '/addCategory') titleEl.textContent = 'Add Category';
            else if (path === '/addPage') titleEl.textContent = 'Add Page';
            else if (path === '/addTab') titleEl.textContent = 'Add Tab';

            contentDiv.innerHTML = '<p>Loading...</p>';
            dialog.showModal();

            fetch(link.href)
                .then(function(res) { return res.text(); })
                .then(function(html) {
                    var parser = new DOMParser();
                    var doc = parser.parseFromString(html, 'text/html');
                    var form = doc.querySelector('form.edit-form');
                    var notes = doc.querySelector('.edit-notes');
                    var error = doc.querySelector('p[style="color: #FF0000"]');

                    contentDiv.innerHTML = '';
                    if (error) {
                        contentDiv.appendChild(error);
                    }
                    if (form) {
                        var actionUrl = form.getAttribute('action') || '';
                        form.action = new URL(actionUrl, link.href).toString();

                        var clickedButton = null;
                        form.addEventListener('click', function(e) {
                            if (e.target.tagName === 'INPUT' && e.target.type === 'submit') {
                                clickedButton = e.target;
                            }
                        });

                        form.addEventListener('submit', function(e) {
                            e.preventDefault();
                            var formData = new FormData(form);
                            if (clickedButton && clickedButton.name) {
                                formData.append(clickedButton.name, clickedButton.value);
                            }

                            var submitBtns = form.querySelectorAll('input[type="submit"]');
                            submitBtns.forEach(function(btn) { btn.disabled = true; });

                            fetch(form.action, {
                                method: form.method || 'POST',
                                body: formData,
                                redirect: 'follow'
                            }).then(function(res) {
                                if (res.ok && res.redirected) {
                                    if (clickedButton && clickedButton.value.indexOf('Stop') !== -1) {
                                        var url = new URL(window.location.href);
                                        url.searchParams.delete('edit');
                                        window.location.href = url.toString();
                                    } else {
                                        window.location.reload();
                                    }
                                } else if (res.ok) {
                                    res.text().then(function(html) {
                                        var parser = new DOMParser();
                                        var doc = parser.parseFromString(html, 'text/html');
                                        var error = doc.querySelector('p[style="color: #FF0000"]');
                                        if (error) {
                                            var existingError = contentDiv.querySelector('p[style="color: #FF0000"]');
                                            if (existingError) {
                                                existingError.textContent = error.textContent;
                                            } else {
                                                contentDiv.insertBefore(error, contentDiv.firstChild);
                                            }
                                            submitBtns.forEach(function(btn) { btn.disabled = false; });
                                        } else {
                                            if (clickedButton && clickedButton.value.indexOf('Done') !== -1) {
                                                 window.location.reload();
                                            } else {
                                                 window.location.reload();
                                            }
                                        }
                                    });
                                } else {
                                    alert('Error saving. HTTP ' + res.status);
                                    submitBtns.forEach(function(btn) { btn.disabled = false; });
                                }
                            }).catch(function(err) {
                                alert('Error saving: ' + err);
                                submitBtns.forEach(function(btn) { btn.disabled = false; });
                            });
                        });
                        contentDiv.appendChild(form);
                    } else {
                        contentDiv.innerHTML = '<p>Error loading form.</p>';
                    }

                    if (notes) {
                        contentDiv.appendChild(notes);
                    }
                })
                .catch(function(err) {
                    contentDiv.innerHTML = '<p>Error loading: ' + err + '</p>';
                });
        }
    });"""

new_block = """    function setupEditDialog() {
        if (document.getElementById('edit-dialog')) return;
        var dialog = document.createElement('dialog');
        dialog.id = 'edit-dialog';
        dialog.className = 'edit-dialog';
        dialog.innerHTML = `
            <div class="edit-dialog-header">
                <h2 id="edit-dialog-title" style="margin: 0;">Edit</h2>
                <button id="edit-dialog-close">&times;</button>
            </div>
            <div id="edit-dialog-content"></div>
        `;
        document.body.appendChild(dialog);

        document.getElementById('edit-dialog-close').addEventListener('click', function(e) {
            e.preventDefault();
            dialog.close();
        });
    }

    function getDialogTitle(path) {
        var titles = {
            '/editCategory': 'Edit Category',
            '/editPage': 'Edit Page',
            '/editTab': 'Edit Tab',
            '/edit': 'Edit All',
            '/addCategory': 'Add Category',
            '/addPage': 'Add Page',
            '/addTab': 'Add Tab'
        };
        return titles[path] || 'Edit';
    }

    function handleDialogSubmit(e, form, link, contentDiv) {
        e.preventDefault();
        var clickedButton = form.dataset.clickedButton ? JSON.parse(form.dataset.clickedButton) : null;
        var formData = new FormData(form);
        if (clickedButton && clickedButton.name) {
            formData.append(clickedButton.name, clickedButton.value);
        }

        var submitBtns = form.querySelectorAll('input[type="submit"]');
        submitBtns.forEach(function(btn) { btn.disabled = true; });

        fetch(form.action, {
            method: form.method || 'POST',
            body: formData,
            redirect: 'follow'
        }).then(function(res) {
            if (res.ok && res.redirected) {
                handleSuccessfulSave(clickedButton);
            } else if (res.ok) {
                handleHtmlResponse(res, contentDiv, submitBtns, clickedButton);
            } else {
                alert('Error saving. HTTP ' + res.status);
                submitBtns.forEach(function(btn) { btn.disabled = false; });
            }
        }).catch(function(err) {
            alert('Error saving: ' + err);
            submitBtns.forEach(function(btn) { btn.disabled = false; });
        });
    }

    function handleSuccessfulSave(clickedButton) {
        if (clickedButton && clickedButton.value.indexOf('Stop') !== -1) {
            var url = new URL(window.location.href);
            url.searchParams.delete('edit');
            window.location.href = url.toString();
        } else {
            window.location.reload();
        }
    }

    function handleHtmlResponse(res, contentDiv, submitBtns, clickedButton) {
        res.text().then(function(html) {
            var parser = new DOMParser();
            var doc = parser.parseFromString(html, 'text/html');
            var error = doc.querySelector('p[style="color: #FF0000"]');
            if (error) {
                displayFormError(error, contentDiv);
                submitBtns.forEach(function(btn) { btn.disabled = false; });
            } else {
                handleSuccessfulSave(clickedButton);
            }
        });
    }

    function displayFormError(error, contentDiv) {
        var existingError = contentDiv.querySelector('p[style="color: #FF0000"]');
        if (existingError) {
            existingError.textContent = error.textContent;
        } else {
            contentDiv.insertBefore(error, contentDiv.firstChild);
        }
    }

    function populateDialogContent(html, linkHref, contentDiv) {
        var parser = new DOMParser();
        var doc = parser.parseFromString(html, 'text/html');
        var form = doc.querySelector('form.edit-form');
        var notes = doc.querySelector('.edit-notes');
        var error = doc.querySelector('p[style="color: #FF0000"]');

        contentDiv.innerHTML = '';
        if (error) {
            contentDiv.appendChild(error);
        }
        if (form) {
            var actionUrl = form.getAttribute('action') || '';
            form.action = new URL(actionUrl, linkHref).toString();

            form.addEventListener('click', function(e) {
                if (e.target.tagName === 'INPUT' && e.target.type === 'submit') {
                    form.dataset.clickedButton = JSON.stringify({name: e.target.name, value: e.target.value});
                }
            });

            form.addEventListener('submit', function(e) {
                handleDialogSubmit(e, form, linkHref, contentDiv);
            });
            contentDiv.appendChild(form);
        } else {
            contentDiv.innerHTML = '<p>Error loading form.</p>';
        }

        if (notes) {
            contentDiv.appendChild(notes);
        }
    }

    function openEditDialog(link) {
        setupEditDialog();
        var u;
        try { u = new URL(link.href, window.location.origin); } catch(err) { return; }

        var dialog = document.getElementById('edit-dialog');
        var contentDiv = document.getElementById('edit-dialog-content');
        var titleEl = document.getElementById('edit-dialog-title');

        titleEl.textContent = getDialogTitle(u.pathname);
        contentDiv.innerHTML = '<p>Loading...</p>';
        dialog.showModal();

        fetch(link.href)
            .then(function(res) { return res.text(); })
            .then(function(html) { populateDialogContent(html, link.href, contentDiv); })
            .catch(function(err) { contentDiv.innerHTML = '<p>Error loading: ' + err + '</p>'; });
    }

    document.body.addEventListener('click', function(e) {
        var link = e.target.closest('a');
        if (!link) return;
        var href = link.getAttribute('href');
        if (!href) return;

        var u;
        try { u = new URL(href, window.location.origin); } catch(err) { return; }
        var validPaths = ['/editCategory', '/editPage', '/editTab', '/edit', '/addCategory', '/addPage', '/addTab'];

        if (validPaths.indexOf(u.pathname) !== -1) {
            if (e.ctrlKey || e.metaKey || e.shiftKey || link.target === '_blank') return;
            e.preventDefault();
            openEditDialog(link);
        }
    });"""

if old_block in content:
    print("Found exact old block")
    content = content.replace(old_block, new_block)
else:
    print("Old block not found!")
    # Trying regex or finding parts

with open('templates/tail.gohtml', 'w') as f:
    f.write(content)
