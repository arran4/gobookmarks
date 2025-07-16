        document.addEventListener('DOMContentLoaded', function () {
            if (!document.body.classList.contains('edit-mode')) return;

            var startZone = null;

            function findColumnZone(block) {
                if (!block) return null;
                if (block.parentNode.tagName === 'TD') {
                    return block.parentNode.querySelector('.columnEndDropZone');
                }
                var col = block.closest('.bookmarkColumn');
                if (col) {
                    return col.querySelector('.columnEndDropZone');
                }
                return null;
            }

            function removeEmptyColumn(zone) {
                if (!zone) return;
                var page = zone.closest('.bookmarkPage');
                if (zone.parentNode.tagName === 'TD') {
                    var td = zone.parentNode;
                    if (!td.querySelector('.categoryBlock')) {
                        td.remove();
                        updateColumnIndices(page);
                    }
                } else {
                    var col = zone.closest('.bookmarkColumn');
                    if (col && !col.querySelector('.categoryBlock')) {
                        var nextZone = col.nextElementSibling;
                        if (nextZone && nextZone.classList.contains('newColumnDropZone')) {
                            nextZone.remove();
                        }
                        col.remove();
                        updateColumnIndices(page);
                    }
                }
            }

            function updateColumnIndices(page) {
                if (!page) return;
                var zones = page.querySelectorAll('.columnEndDropZone');
                zones.forEach(function (z, i) {
                    z.dataset.col = i;
                });
                var newZones = page.querySelectorAll('.newColumnDropZone');
                newZones.forEach(function (z, i) {
                    z.dataset.col = i;
                });
            }

            document.querySelectorAll('.categoryBlock').forEach(function (block) {
                block.addEventListener('dragover', dragOver);
                block.addEventListener('dragleave', dragLeave);
                block.addEventListener('drop', drop);
            });

            document.querySelectorAll('.categoryBlock h2').forEach(function (title) {
                title.setAttribute('draggable', 'true');
                title.addEventListener('dragstart', dragStart);
            });

            document.querySelectorAll('.newColumnDropZone').forEach(function (zone) {
                zone.addEventListener('dragover', dragOver);
                zone.addEventListener('dragleave', dragLeave);
                zone.addEventListener('drop', dropNewColumn);
            });

            document.querySelectorAll('.columnEndDropZone').forEach(function (zone) {
                zone.addEventListener('dragover', dragOver);
                zone.addEventListener('dragleave', dragLeave);
                zone.addEventListener('drop', dropEndColumn);
            });

            document.querySelectorAll('#page-list li[data-page-sha]').forEach(function (li) {
                li.addEventListener('dragover', dragOver);
                li.addEventListener('dragleave', dragLeave);
                li.addEventListener('drop', dropOnPageIndex);
            });

            document.querySelectorAll('#tab-list li[data-page-sha]').forEach(function (li) {
                li.addEventListener('dragover', dragOver);
                li.addEventListener('dragleave', dragLeave);
                li.addEventListener('drop', dropOnTabIndex);
            });

            function dragStart(e) {
                var block = e.currentTarget.closest('.categoryBlock');
                startZone = findColumnZone(block);
                e.dataTransfer.setData('text/plain', block.id);
                var page = block.closest('.bookmarkPage');
                if (page) {
                    e.dataTransfer.setData('pageSha', page.dataset.sha);
                }
                e.dataTransfer.effectAllowed = 'move';
            }

            function sendMoveBefore(from, to, pageSha, destSha, destCol) {
                var params = new URLSearchParams(window.location.search);
                var ref = params.get('ref') || 'refs/heads/main';
                var branch = '';
                if (ref.startsWith('refs/heads/')) {
                    branch = ref.slice(11);
                } else if (ref.startsWith('refs/tags/')) {
                    branch = 'New' + ref.slice(10);
                } else if (ref) {
                    branch = 'FromCommit' + ref;
                } else {
                    branch = 'main';
                }

                var fd = new FormData();
                fd.append('from', from);
                fd.append('to', to);
                if (pageSha) fd.append('pageSha', pageSha);
                fd.append('branch', branch);
                fd.append('ref', ref);
                if (destSha) fd.append('destPageSha', destSha);
                if (destCol !== null) fd.append('destCol', destCol);
                fetch('/moveCategory', {method: 'POST', body: fd, credentials: 'same-origin'})
                    .then(() => location.reload());
            }

            function sendMoveEnd(from, pageSha, destSha, destCol) {
                var params = new URLSearchParams(window.location.search);
                var ref = params.get('ref') || 'refs/heads/main';
                var branch = '';
                if (ref.startsWith('refs/heads/')) {
                    branch = ref.slice(11);
                } else if (ref.startsWith('refs/tags/')) {
                    branch = 'New' + ref.slice(10);
                } else if (ref) {
                    branch = 'FromCommit' + ref;
                } else {
                    branch = 'main';
                }

                var fd = new FormData();
                fd.append('from', from);
                if (pageSha) fd.append('pageSha', pageSha);
                fd.append('branch', branch);
                fd.append('ref', ref);
                if (destSha) fd.append('destPageSha', destSha);
                if (destCol !== null) fd.append('destCol', destCol);
                fetch('/moveCategoryEnd', {method: 'POST', body: fd, credentials: 'same-origin'})
                    .then(() => location.reload());
            }

            function sendMoveNewColumn(from, pageSha, destSha, destCol) {
                var params = new URLSearchParams(window.location.search);
                var ref = params.get('ref') || 'refs/heads/main';
                var branch = '';
                if (ref.startsWith('refs/heads/')) {
                    branch = ref.slice(11);
                } else if (ref.startsWith('refs/tags/')) {
                    branch = 'New' + ref.slice(10);
                } else if (ref) {
                    branch = 'FromCommit' + ref;
                } else {
                    branch = 'main';
                }

                var fd = new FormData();
                fd.append('from', from);
                if (pageSha) fd.append('pageSha', pageSha);
                fd.append('branch', branch);
                fd.append('ref', ref);
                if (destSha) fd.append('destPageSha', destSha);
                if (destCol !== undefined && destCol !== null) fd.append('destCol', destCol);
                fetch('/moveCategoryNewColumn', {method: 'POST', body: fd, credentials: 'same-origin'})
                    .then(() => location.reload());
            }

            function dragOver(e) {
                e.preventDefault();
                e.currentTarget.classList.add('drag-over');
            }

            function dragLeave(e) {
                e.currentTarget.classList.remove('drag-over');
            }

            function drop(e) {
                e.preventDefault();
                e.currentTarget.classList.remove('drag-over');
                var id = e.dataTransfer.getData('text/plain');
                var el = document.getElementById(id);
                if (el && el !== e.currentTarget) {
                    e.currentTarget.parentNode.insertBefore(el, e.currentTarget);
                    var from = parseInt(id.substring(3));
                    var to = parseInt(e.currentTarget.id.substring(3));
                    var pageSha = e.dataTransfer.getData('pageSha');
                    var destPage = e.currentTarget.closest('.bookmarkPage');
                    var destSha = destPage.dataset.sha;
                    var destCol = e.currentTarget.dataset.col ? parseInt(e.currentTarget.dataset.col) : null;
                    sendMoveBefore(from, to, pageSha, destSha, destCol);
                    updateColumnIndices(destPage);
                    removeEmptyColumn(startZone);
                    startZone = null;
                }
            }

            function dropNewColumn(e) {
                e.preventDefault();
                e.currentTarget.classList.remove('drag-over');
                var id = e.dataTransfer.getData('text/plain');
                var el = document.getElementById(id);
                if (el) {
                    var from = parseInt(id.substring(3));
                    var pageSha = e.dataTransfer.getData('pageSha');
                    var destPage = e.currentTarget.closest('.bookmarkPage');
                    var destSha = destPage.dataset.sha;
                    var destCol = e.currentTarget.dataset.col ? parseInt(e.currentTarget.dataset.col) : -1;
                    var zone;
                    if (e.currentTarget.tagName === 'TD') {
                        var td = document.createElement('td');
                        td.appendChild(el);
                        zone = document.createElement('div');
                        zone.className = 'columnEndDropZone';
                        td.appendChild(zone);
                        e.currentTarget.parentNode.insertBefore(td, e.currentTarget);
                    } else {
                        var col = document.createElement('div');
                        col.className = 'bookmarkColumn';
                        col.appendChild(el);
                        zone = document.createElement('div');
                        zone.className = 'columnEndDropZone';
                        col.appendChild(zone);
                        e.currentTarget.parentNode.insertBefore(col, e.currentTarget);
                    }
                    sendMoveNewColumn(from, pageSha, destSha, destCol);
                    updateColumnIndices(destPage);
                    removeEmptyColumn(startZone);
                    startZone = null;
                }
            }

            function dropEndColumn(e) {
                e.preventDefault();
                e.currentTarget.classList.remove('drag-over');
                var id = e.dataTransfer.getData('text/plain');
                var el = document.getElementById(id);
                if (el) {
                    var parent = e.currentTarget.parentNode;
                    parent.insertBefore(el, e.currentTarget);
                    var from = parseInt(id.substring(3));
                    var pageSha = e.dataTransfer.getData('pageSha');
                    var destPage = e.currentTarget.closest('.bookmarkPage');
                    var destSha = destPage.dataset.sha;
                    var destCol = e.currentTarget.dataset.col ? parseInt(e.currentTarget.dataset.col) : null;
                    sendMoveEnd(from, pageSha, destSha, destCol);
                    updateColumnIndices(destPage);
                    removeEmptyColumn(startZone);
                    startZone = null;
                }
            }

            function dropOnPageIndex(e) {
                e.preventDefault();
                e.currentTarget.classList.remove('drag-over');
                var id = e.dataTransfer.getData('text/plain');
                if (id) {
                    var from = parseInt(id.substring(3));
                    var pageSha = e.dataTransfer.getData('pageSha');
                    var destSha = e.currentTarget.dataset.pageSha;
                    sendMoveEnd(from, pageSha, destSha, -1);
                    startZone = null;
                }
            }

            function dropOnTabIndex(e) {
                e.preventDefault();
                e.currentTarget.classList.remove('drag-over');
                var id = e.dataTransfer.getData('text/plain');
                if (id) {
                    var from = parseInt(id.substring(3));
                    var pageSha = e.dataTransfer.getData('pageSha');
                    var destSha = e.currentTarget.dataset.pageSha;
                    sendMoveEnd(from, pageSha, destSha, -1);
                    startZone = null;
                }
            }
        });
