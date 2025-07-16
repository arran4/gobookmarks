                document.addEventListener('DOMContentLoaded', function () {
                    var toggleEdit = document.getElementById('toggle-edit');

                    function currentPage() {
                        var pages = document.querySelectorAll('.bookmarkPage[id^="page"]');
                        var closest = -1;
                        var closestDist = Infinity;
                        pages.forEach(function (p, idx) {
                            var rect = p.getBoundingClientRect();
                            var dist = Math.abs(rect.top);
                            if (dist < closestDist) {
                                closestDist = dist;
                                closest = idx;
                            }
                        });
                        return closest;
                    }

                    function attach(link) {
                        if (!link) return;
                        link.addEventListener('click', function (e) {
                            var page = currentPage();
                            if (page >= 0) {
                                var url = new URL(link.getAttribute('href'), window.location);
                                url.searchParams.set('page', page);
                                url.hash = 'page' + page;
                                e.preventDefault();
                                window.location.href = url.toString();
                            }
                        });
                    }

                    attach(toggleEdit);

                    var searchBox = document.getElementById('search-box');
                    var searchResults = [];
                    var selectedIndex = 0;
                    var initialHash = '';
                    var initialPage = -1;

                    function restoreInitial() {
                        if (initialHash !== '') {
                            var pages = document.querySelectorAll('.bookmarkPage[id^="page"]');
                            if (initialPage >= 0 && pages.length > initialPage) {
                                var p = pages[initialPage];
                                location.hash = '#' + p.id;
                                p.scrollIntoView({block: 'center'});
                            } else if (initialHash) {
                                location.hash = initialHash;
                            }
                        }
                        initialHash = '';
                        initialPage = -1;
                    }

                    function resetResults() {
                        document.querySelectorAll('.search-hidden').forEach(function(el){
                            el.classList.remove('search-hidden');
                        });
                        document.querySelectorAll('.search-selected').forEach(function(el){
                            el.classList.remove('search-selected');
                        });
                        searchResults = [];
                        selectedIndex = 0;
                    }

                    function clearSearch() {
                        resetResults();
                        restoreInitial();
                    }

                    function updateSearch() {
                        resetResults();
                        if (!searchBox) return;
                        var q = searchBox.value.trim().toLowerCase();
                        if (!q) return;
                        if (initialHash === '') {
                            initialHash = location.hash;
                            initialPage = currentPage();
                        }
                        var items = document.querySelectorAll('.bookmark-entries li');
                        items.forEach(function(li) {
                            var a = li.querySelector('a[target="_blank"]');
                            if (!a) return;
                            var text = a.textContent.toLowerCase();
                            var url = a.getAttribute('href').toLowerCase();
                            if (text.indexOf(q) !== -1 || url.indexOf(q) !== -1) {
                                searchResults.push(li);
                            } else {
                                li.classList.add('search-hidden');
                            }
                        });
                        if (searchResults.length > 0) {
                            searchResults[0].classList.add('search-selected');
                            var firstPage = searchResults[0].closest('.bookmarkPage');
                            if (firstPage) {
                                location.hash = '#' + firstPage.id;
                                firstPage.scrollIntoView({block: 'center'});
                            } else {
                                searchResults[0].scrollIntoView({block: 'nearest'});
                            }
                        }
                        searchBox.focus();
                    }

                    function moveSelection(delta) {
                        if (searchResults.length === 0) return;
                        var hadFocus = document.activeElement === searchBox;
                        var cur = searchResults[selectedIndex];
                        cur.classList.remove('search-selected');
                        selectedIndex = (selectedIndex + delta + searchResults.length) % searchResults.length;
                        var nextEl = searchResults[selectedIndex];
                        nextEl.classList.add('search-selected');
                        var curPage = cur.closest('.bookmarkPage');
                        var newPage = nextEl.closest('.bookmarkPage');
                        if (newPage && curPage !== newPage) {
                            location.hash = '#' + newPage.id;
                            newPage.scrollIntoView({block: 'center'});
                        } else {
                            nextEl.scrollIntoView({block: 'nearest'});
                        }
                        if (hadFocus) searchBox.focus();
                    }

                    function moveSelectionHorizontal(dir) {
                        if (searchResults.length === 0) return;
                        var hadFocus = document.activeElement === searchBox;
                        var cur = searchResults[selectedIndex];
                        var r0 = cur.getBoundingClientRect();
                        var midY = (r0.top + r0.bottom) / 2;
                        var best = -1;
                        var bestDist = Infinity;
                        searchResults.forEach(function(li, idx){
                            if (idx === selectedIndex) return;
                            var r = li.getBoundingClientRect();
                            var withinY = Math.abs(((r.top + r.bottom)/2) - midY);
                            if (dir < 0 && r.right <= r0.left) {
                                var dist = (r0.left - r.right) * (r0.left - r.right) + withinY * withinY;
                                if (dist < bestDist) { bestDist = dist; best = idx; }
                            } else if (dir > 0 && r.left >= r0.right) {
                                var dist = (r.left - r0.right) * (r.left - r0.right) + withinY * withinY;
                                if (dist < bestDist) { bestDist = dist; best = idx; }
                            }
                        });
                        if (best >= 0) {
                            var curPage = cur.closest('.bookmarkPage');
                            searchResults[selectedIndex].classList.remove('search-selected');
                            selectedIndex = best;
                            var nextEl = searchResults[selectedIndex];
                            nextEl.classList.add('search-selected');
                            var newPage = nextEl.closest('.bookmarkPage');
                            if (newPage && curPage !== newPage) {
                                location.hash = '#' + newPage.id;
                                newPage.scrollIntoView({block: 'center'});
                            } else {
                                nextEl.scrollIntoView({block: 'nearest'});
                            }
                        }
                        if (hadFocus) searchBox.focus();
                    }

                    if (searchBox) {
                        searchBox.addEventListener('input', updateSearch);
                        searchBox.addEventListener('keydown', function(e) {
                            if (e.key === 'ArrowDown') {
                                moveSelection(1);
                                e.preventDefault();
                            } else if (e.key === 'ArrowUp') {
                                moveSelection(-1);
                                e.preventDefault();
                            } else if (e.key === 'ArrowRight') {
                                moveSelectionHorizontal(1);
                                e.preventDefault();
                            } else if (e.key === 'ArrowLeft') {
                                moveSelectionHorizontal(-1);
                                e.preventDefault();
                            } else if (e.key === 'Enter') {
                                if (searchResults.length > 0) {
                                    var link = searchResults[selectedIndex].querySelector('a');
                                    if (link) {
                                        if (e.ctrlKey || e.metaKey) {
                                            window.open(link.href, '_blank', 'noopener');
                                        } else if (link.target && link.target === '_blank') {
                                            window.open(link.href, '_blank');
                                        } else {
                                            window.location.href = link.href;
                                        }
                                        searchBox.focus();
                                    }
                                }
                                e.preventDefault();
                            } else if (e.key === 'Escape') {
                                searchBox.blur();
                                e.preventDefault();
                                e.stopPropagation();
                            }
                        });
                    }

                    function changePage(delta) {
                        var pages = document.querySelectorAll('.bookmarkPage[id^="page"]');
                        if (!pages.length) return;
                        var hadFocus = document.activeElement === searchBox;
                        var cur = currentPage();
                        if (cur < 0) cur = 0;
                        var next = (cur + delta + pages.length) % pages.length;
                        var id = pages[next].id;
                        location.hash = '#' + id;
                        pages[next].scrollIntoView();
                        if (hadFocus) searchBox.focus();
                    }

                    function changeTab(delta) {
                        var tabs = document.querySelectorAll('#tab-list a');
                        if (!tabs.length) return;
                        var params = new URLSearchParams(window.location.search);
                        var idx = parseInt(params.get('tab') || '0');
                        if (isNaN(idx)) idx = 0;
                        var next = (idx + delta + tabs.length) % tabs.length;
                        window.location.href = tabs[next].href;
                    }

                    document.addEventListener('keydown', function(e) {
                        var active = document.activeElement;
                        var inInput = active && (active.tagName === 'INPUT' || active.tagName === 'TEXTAREA' || active.isContentEditable);

                        if (e.altKey && !e.ctrlKey && !e.metaKey) {
                            if (e.key === ']') { changePage(1); e.preventDefault(); return; }
                            if (e.key === '[') { changePage(-1); e.preventDefault(); return; }
                            if (e.key === '}') { changeTab(1); e.preventDefault(); return; }
                            if (e.key === '{') { changeTab(-1); e.preventDefault(); return; }
                            if (e.key.toLowerCase() === 'k') { if (searchBox) { searchBox.focus(); searchBox.select(); } e.preventDefault(); return; }
                        }

                        if (!inInput) {
                            if (e.key === 'ArrowDown') { moveSelection(1); e.preventDefault(); }
                            else if (e.key === 'ArrowUp') { moveSelection(-1); e.preventDefault(); }
                            else if (e.key === 'ArrowRight') { moveSelectionHorizontal(1); e.preventDefault(); }
                            else if (e.key === 'ArrowLeft') { moveSelectionHorizontal(-1); e.preventDefault(); }
                            else if (e.key === '?') {
                                alert('Keyboard shortcuts:\nAlt+[ and Alt+] - switch page\nAlt+{ and Alt+} - switch tab\nAlt+K - focus search\nArrows move selection\nEnter - open\nCtrl+Enter - open in background\nEsc twice - clear search and restore view');
                                e.preventDefault();
                            } else if (e.key === 'Escape') {
                                if (searchBox && searchBox.value !== '') {
                                    searchBox.value = '';
                                    clearSearch();
                                } else if (searchBox) {
                                    searchBox.blur();
                                }
                                e.preventDefault();
                            } else if (e.key === 'Enter') {
                                if (searchResults.length > 0) {
                                    var link = searchResults[selectedIndex].querySelector('a');
                                    if (link) {
                                        if (e.ctrlKey || e.metaKey) {
                                            window.open(link.href, '_blank', 'noopener');
                                        } else if (link.target && link.target === '_blank') {
                                            window.open(link.href, '_blank');
                                        } else {
                                            window.location.href = link.href;
                                        }
                                        if (searchBox) searchBox.focus();
                                    }
                                }
                            }
                        }
                    });
                });
