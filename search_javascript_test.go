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

	jsScript := string(htmlContent)
	startIdx := strings.Index(jsScript, "<script>")
	endIdx := strings.LastIndex(jsScript, "</script>")
	if startIdx != -1 && endIdx != -1 {
		jsScript = jsScript[startIdx+8 : endIdx]
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
        if (sel === '.search-hidden') return [];
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
    createLI('Apple', 'https://example.com/fruit'),
    createLI('Banana', 'https://github.com/banana'),
    createLI('Some docs', 'https://github.com/issues'),
    createLI('GitHub', 'https://github.com'),
];

global.mockOriginalLists[0].appendChild(items[0]);
global.mockOriginalLists[0].appendChild(items[1]);
global.mockOriginalLists[0].appendChild(items[2]);
global.mockOriginalLists[1].appendChild(items[3]);

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

function testCycle(query, expectedLength, expectedFirst, expectedSecond) {
    global.mockSearchBox.value = query;
    global.mockSearchBox._listeners.forEach(cb => cb());

    var globalList = global.mockTabContent.firstChild;
    if (query === '') {
        if (globalList) {
            console.error("Expected global search results to be removed on empty query");
            process.exit(1);
        }

        if (global.mockOriginalLists[0].children.length !== 3 || global.mockOriginalLists[1].children.length !== 1) {
            console.error("Expected lists to be fully restored. List 1: " + global.mockOriginalLists[0].children.length + ", List 2: " + global.mockOriginalLists[1].children.length);
            process.exit(1);
        }

        var title0 = global.mockOriginalLists[0].children[0].querySelector('a[target="_blank"]').textContent;
        var title2 = global.mockOriginalLists[0].children[2].querySelector('a[target="_blank"]').textContent;
        if (title0 !== 'Apple' || title2 !== 'Some docs') {
            console.error("Order incorrect after restore: " + title0 + " ... " + title2);
            process.exit(1);
        }

        [...global.mockOriginalLists[0].children, ...global.mockOriginalLists[1].children].forEach(li => {
            if (li._searchPlaceholder) {
                console.error("Placeholder remained after clear for: " + li.querySelector('a[target="_blank"]').textContent);
                process.exit(1);
            }
        });

        return;
    }

    if (!globalList || globalList.tagName !== 'ul') {
        console.error("No global search results list found");
        process.exit(1);
    }

    if (globalList.children.length !== expectedLength) {
        console.error("Expected " + expectedLength + " children in global list, got " + globalList.children.length);
        process.exit(1);
    }

    if (expectedFirst) {
        var firstResult = globalList.children[0].querySelector('a[target="_blank"]').textContent;
        if (firstResult !== expectedFirst) {
            console.error("Expected '" + expectedFirst + "' to be first, got '" + firstResult + "'");
            process.exit(1);
        }
    }

    if (expectedSecond) {
        var secondResult = globalList.children[1].querySelector('a[target="_blank"]').textContent;
        if (secondResult !== expectedSecond) {
            console.error("Expected '" + expectedSecond + "' to be second, got '" + secondResult + "'");
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
        if (li._searchPlaceholder) {
            console.error("Unmatched item unexpectedly has placeholder: " + li.querySelector('a[target="_blank"]').textContent);
            process.exit(1);
        }
    });
}

testCycle('github', 3, 'GitHub', 'Banana');
testCycle('docs', 1, 'Some docs');
testCycle('', 0);
testCycle('apple', 1, 'Apple');
testCycle('', 0);

console.log("Success");
`
	tmpFile, err := os.CreateTemp("", "search_test_*.js")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(testScript)); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}
	tmpFile.Close()

	cmd := exec.Command("node", tmpFile.Name())
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		t.Fatalf("Node script failed: %v\nOutput:\n%s", err, out.String())
	}
}
