package gobookmarks

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestSearchJSOrder(t *testing.T) {
	_, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required to run JS tests")
	}

	htmlContent, err := os.ReadFile("templates/tail.gohtml")
	if err != nil {
		t.Fatalf("failed to read tail.gohtml: %v", err)
	}

	rawHTML := string(htmlContent)
	var jsScript string
	for {
		startIdx := strings.Index(rawHTML, "<script>")
		if startIdx == -1 {
			break
		}
		endIdx := strings.Index(rawHTML[startIdx:], "</script>")
		if endIdx == -1 {
			break
		}
		block := rawHTML[startIdx+8 : startIdx+endIdx]
		if strings.Contains(block, "function updateSearch()") {
			jsScript = block
			break
		}
		rawHTML = rawHTML[startIdx+endIdx+9:]
	}

	jsScript = strings.Replace(jsScript, "var searchResults = [];", "global.searchResults = [];", 1)
	jsScript = strings.ReplaceAll(jsScript, "searchResults = [];", "global.searchResults = [];")
	jsScript = strings.ReplaceAll(jsScript, "searchResults.push", "global.searchResults.push")
	jsScript = strings.ReplaceAll(jsScript, "searchResults[", "global.searchResults[")
	jsScript = strings.ReplaceAll(jsScript, "searchResults.length", "global.searchResults.length")

	testScript := `
global.document = {
    querySelectorAll: function(sel) {
        if (sel === 'meta[name=\'base-title\']') return null;
        if (sel === '.search-hidden') {
            return global.mockItems.filter(i => i.isHidden);
        }
        if (sel === '.search-selected') return [];
        if (sel === '.tab-panel') return global.mockTabPanels;
        if (sel === '#tab-list li, #page-list li') return [];
        if (sel === '.bookmark-entries:not(#global-search-results) li' || sel === '.bookmark-entries li') {
            var results = [];
            global.mockOriginalLists.forEach(ul => {
                results = results.concat(ul.children);
            });
            if (sel.indexOf(':not') === -1 && global.mockTabContent.firstChild) {
                results = results.concat(global.mockTabContent.firstChild.children);
            }
            return results;
        }
        return [];
    },
    createElement: function(tag) {
        return {
            tagName: tag,
            style: {},
            children: [],
            appendChild: function(child) {
                if (child.parentNode) {
                    child.parentNode.removeChild(child);
                }
                this.children.push(child);
                child.parentNode = this;
            },
            removeChild: function(child) {
                this.children = this.children.filter(c => c !== child);
                child.parentNode = null;
            },
            insertBefore: function(newNode, refNode) {
                if (newNode.parentNode) {
                    newNode.parentNode.removeChild(newNode);
                }
                const idx = this.children.indexOf(refNode);
                if (idx !== -1) {
                    this.children.splice(idx, 0, newNode);
                } else {
                    this.children.push(newNode);
                }
                newNode.parentNode = this;
            },
            classList: { add: function(){}, remove: function(){} }
        };
    },
    createComment: function(text) {
        return { isComment: true, text: text, parentNode: null };
    },
    querySelector: function(sel) { return null; },
    getElementById: function(id) {
        if (id === 'search-box') return global.mockSearchBox;
        if (id === 'tab-content') return global.mockTabContent;
        if (id === 'global-search-results') return global.mockTabContent.firstChild;
        if (id === 'tab-list') return { querySelectorAll: function(){ return []; } };
        return null;
    },
    addEventListener: function(event, callback) {
        if (event === 'DOMContentLoaded') {
            global.initAppCallbacks.push(callback);
        }
    },
    dispatchEvent: function() {},
    body: { addEventListener: function(){}, dataset: {tab: '0'} },
    activeElement: null,
    documentElement: { getAttribute: function(){ return null; }, setAttribute: function(){}, removeAttribute: function(){} }
};

global.initAppCallbacks = [];
global.window = { location: { hash: '', search: '' }, addEventListener: function() {}, open: function(){} };
global.localStorage = { getItem: function() { return null; }, setItem: function() {}, removeItem: function() {} };
global.location = global.window.location;
global.Event = class { constructor() {} };

global.mockSearchBox = { value: '', focus: function(){}, addEventListener: function(){} };
global.mockTabContent = {
    classList: { add: function(){}, remove: function(){} },
    dataset: {},
    children: [],
    insertBefore: function(node, ref) {
        if (node.parentNode) node.parentNode.removeChild(node);
        this.firstChild = node;
        node.parentNode = this;
    },
    appendChild: function(node) {
        if (node.parentNode) node.parentNode.removeChild(node);
        this.firstChild = node;
        node.parentNode = this;
    },
    removeChild: function(node) {
        if (this.firstChild === node) this.firstChild = null;
        node.parentNode = null;
    },
    firstChild: null,
    querySelectorAll: function() { return global.mockTabPanels; }
};

global.mockTabPanels = [
    { style: {}, dataset: { tabIndex: '0' }, classList: { toggle: function(){} }, querySelectorAll: function(){ return []; } }
];

function createLI(title, url, hiddenClass) {
    let li = {
        classList: {
            add: function(c){ if(c==='search-hidden') li.isHidden = true; },
            remove: function(c){ if(c==='search-hidden') li.isHidden = false; }
        },
        isHidden: hiddenClass,
        querySelector: function(sel) {
            if (sel === 'a[target="_blank"]') {
                return {
                    textContent: title,
                    getAttribute: function(attr) { if(attr === 'href') return url; return ''; }
                };
            }
            if (sel === 'input.search-widget') return null;
            return null;
        },
        closest: function() { return null; },
        scrollIntoView: function() {}
    };
    li.parentNode = null;
    li.nextSibling = null;
    li._searchPlaceholder = undefined;
    return li;
}

global.mockOriginalLists = [
    global.document.createElement('ul'),
    global.document.createElement('ul')
];

let items = [
    createLI('Apple', 'https://example.com/fruit', false),
    createLI('Banana docs', 'https://github.com/banana', false),
    createLI('Some docs', 'https://github.com/issues', false),
    createLI('GitHub', 'https://github.com', false),
    createLI('Extra github', 'https://github.com/extra', false), // title+URL match
    createLI('Docs repo', 'https://github.com/docs', false),
];

// List 1
global.mockOriginalLists[0].appendChild(items[0]);
global.mockOriginalLists[0].appendChild(items[1]);
global.mockOriginalLists[0].appendChild(items[2]);
// List 2
global.mockOriginalLists[1].appendChild(items[3]);
global.mockOriginalLists[1].appendChild(items[4]);
global.mockOriginalLists[1].appendChild(items[5]);

global.mockItems = items;

var searchMode = false;
var currentTabIndex = 0;
function showAllTabsForSearch() {}
function currentPage() { return 1; }
function restoreInitial() {}
function setActiveTab() {}
function updateVisibility() {}
function updatePageList() {}

` + jsScript + `

global.mockSearchBox._listeners = [];
global.mockSearchBox.addEventListener = function(event, callback) {
    if (event === 'input') {
        global.mockSearchBox._listeners.push(callback);
    }
};

global.initAppCallbacks.forEach(cb => cb());

function testCycle(query, expectedTitlesInOrder) {
    global.mockSearchBox.value = query;
    global.mockSearchBox._listeners.forEach(cb => cb());

    var globalList = global.mockTabContent.firstChild;
    if (query === '') {
        if (globalList) {
            console.error("Expected global search results to be removed on empty query");
            process.exit(1);
        }

        if (global.mockOriginalLists[0].children.length !== 3 || global.mockOriginalLists[1].children.length !== 3) {
            console.error("Expected lists to be fully restored. List 1: " + global.mockOriginalLists[0].children.length + ", List 2: " + global.mockOriginalLists[1].children.length);
            process.exit(1);
        }

        var titles = [
            global.mockOriginalLists[0].children[0].querySelector('a[target="_blank"]').textContent,
            global.mockOriginalLists[0].children[1].querySelector('a[target="_blank"]').textContent,
            global.mockOriginalLists[0].children[2].querySelector('a[target="_blank"]').textContent,
            global.mockOriginalLists[1].children[0].querySelector('a[target="_blank"]').textContent,
            global.mockOriginalLists[1].children[1].querySelector('a[target="_blank"]').textContent,
            global.mockOriginalLists[1].children[2].querySelector('a[target="_blank"]').textContent,
        ];

        var expectedTitles = ['Apple', 'Banana docs', 'Some docs', 'GitHub', 'Extra github', 'Docs repo'];
        for (let i = 0; i < titles.length; i++) {
            if (titles[i] !== expectedTitles[i]) {
                console.error("Order incorrect after restore at index " + i + ": " + titles[i] + " != " + expectedTitles[i]);
                process.exit(1);
            }
        }

        [...global.mockOriginalLists[0].children, ...global.mockOriginalLists[1].children].forEach(li => {
            if (li._searchPlaceholder) {
                console.error("Placeholder remained after clear for: " + li.querySelector('a[target="_blank"]').textContent);
                process.exit(1);
            }
            if (li.isHidden) {
                console.error("Item remained hidden after clear for: " + li.querySelector('a[target="_blank"]').textContent);
                process.exit(1);
            }
        });

        return;
    }

    if (!globalList || globalList.tagName !== 'ul') {
        console.error("No global search results list found");
        process.exit(1);
    }

    if (globalList.children.length !== expectedTitlesInOrder.length) {
        console.error("Expected " + expectedTitlesInOrder.length + " children in global list, got " + globalList.children.length);
        process.exit(1);
    }

    for (let i = 0; i < expectedTitlesInOrder.length; i++) {
        var res = globalList.children[i].querySelector('a[target="_blank"]').textContent;
        if (res !== expectedTitlesInOrder[i]) {
            console.error("Expected '" + expectedTitlesInOrder[i] + "' at index " + i + ", got '" + res + "'");
            process.exit(1);
        }
    }

    for (let i = 0; i < global.searchResults.length; i++) {
        if (global.searchResults[i] !== globalList.children[i]) {
            console.error("Keyboard order does not match visual order at index " + i);
            process.exit(1);
        }
    }

    globalList.children.forEach(li => {
        if (!li._searchPlaceholder) {
            console.error("Item missing search placeholder: " + li.querySelector('a[target="_blank"]').textContent);
            process.exit(1);
        }
    });

    [...global.mockOriginalLists[0].children, ...global.mockOriginalLists[1].children].forEach(li => {
        if (li && li.querySelector) {
            if (li._searchPlaceholder) {
                console.error("Unmatched item unexpectedly has placeholder: " + li.querySelector('a[target="_blank"]').textContent);
                process.exit(1);
            }
            if (!li.isHidden) {
                console.error("Unmatched item is not hidden: " + li.querySelector('a[target="_blank"]').textContent);
                process.exit(1);
            }
        }
    });
}

// 1. Query 'github':
// Title matches: 'GitHub' (List 2), 'Extra github' (List 2)
// URL matches: 'Banana docs' (List 1), 'Some docs' (List 1), 'Docs repo' (List 2)
// Should sort: Title matches (original order), URL matches (original order)
// Result: 'GitHub', 'Extra github', 'Banana docs', 'Some docs', 'Docs repo'
testCycle('github', ['GitHub', 'Extra github', 'Banana docs', 'Some docs', 'Docs repo']);

// 2. Query 'docs':
// Title matches: 'Banana docs' (List 1), 'Some docs' (List 1), 'Docs repo' (List 2)
// URL matches: none
testCycle('docs', ['Banana docs', 'Some docs', 'Docs repo']);

// 3. Clear
testCycle('', []);

// 4. Query 'apple'
testCycle('apple', ['Apple']);

// 5. Query 'issues'
// Title matches: none
// URL matches: 'Some docs' (List 1)
testCycle('issues', ['Some docs']);

// 6. Clear again
testCycle('', []);

console.log("Success");
`
	tmpFile, err := os.CreateTemp("", "search_test_*.js")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.Write([]byte(testScript)); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}
	_ = tmpFile.Close()

	cmd := exec.Command("node", tmpFile.Name())
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		t.Fatalf("Node script failed: %v\nOutput:\n%s", err, out.String())
	}
}
