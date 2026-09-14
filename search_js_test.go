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

	testScript := `
global.document = {
    querySelectorAll: function(sel) {
        if (sel === 'meta[name=\'base-title\']') return null;
        if (sel === '.search-hidden') return [];
        if (sel === '.search-selected') return [];
        if (sel === '.tab-panel') return global.mockTabPanels;
        if (sel === '#tab-list li, #page-list li') return [];
        if (sel === '.bookmark-entries:not(#global-search-results) li' || sel === '.bookmark-entries li') {
            return global.mockItems;
        }
        return [];
    },
    createElement: function(tag) {
        return {
            tagName: tag,
            style: {},
            children: [],
            appendChild: function(child) {
                this.children.push(child);
                child.parentNode = this;
            },
            removeChild: function(child) {
                this.children = this.children.filter(c => c !== child);
            },
            classList: { add: function(){} }
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
            global.initApp = callback;
        }
    },
    dispatchEvent: function() {},
    body: { addEventListener: function(){}, dataset: {tab: '0'} },
    activeElement: null
};

global.window = { location: { hash: '', search: '' }, addEventListener: function() {}, open: function(){} };
global.localStorage = { getItem: function() { return null; }, setItem: function() {} };
global.location = global.window.location;
global.Event = class { constructor() {} };

global.mockSearchBox = { value: '', focus: function(){}, addEventListener: function(){} };
global.mockTabContent = {
    classList: { add: function(){}, remove: function(){} },
    dataset: {},
    children: [],
    insertBefore: function(node, ref) {
        this.firstChild = node;
        node.parentNode = this;
    },
    appendChild: function(node) {
        this.firstChild = node;
        node.parentNode = this;
    },
    removeChild: function(node) {
        if (this.firstChild === node) this.firstChild = null;
    },
    firstChild: null,
    querySelectorAll: function() { return global.mockTabPanels; }
};

global.mockTabPanels = [
    { style: {}, dataset: { tabIndex: '0' }, classList: { toggle: function(){} }, querySelectorAll: function(){ return []; } }
];

function createLI(title, url) {
    let li = {
        classList: { add: function(){}, remove: function(){} },
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

    // Minimal mock for parentNode
    li.parentNode = {
        insertBefore: function(newNode, refNode) {
            newNode.parentNode = this;
        },
        removeChild: function(child) {
            child.parentNode = null;
        }
    };
    li.nextSibling = null;
    return li;
}

global.mockItems = [
    createLI('Some docs', 'https://github.com/issues'),
    createLI('GitHub', 'https://github.com'),
];

var searchMode = false;
var currentTabIndex = 0;
function showAllTabsForSearch() {}
function currentPage() { return 1; }
function restoreInitial() {}
function setActiveTab() {}

` + jsScript + `

// Intercept addEventListener on searchBox
global.mockSearchBox._listeners = [];
global.mockSearchBox.addEventListener = function(event, callback) {
    if (event === 'input') {
        global.mockSearchBox._listeners.push(callback);
    }
};

// Execute the DOMContentLoaded callback
global.initApp();

function testCycle(query, expectedLength, expectedFirst) {
    global.mockSearchBox.value = query;
    global.mockSearchBox._listeners.forEach(cb => cb());

    var globalList = global.mockTabContent.firstChild;
    if (query === '') {
        if (globalList) {
            console.error("Expected global search results to be removed on empty query");
            process.exit(1);
        }
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

    var firstResult = globalList.children[0].querySelector('a[target="_blank"]').textContent;
    if (firstResult !== expectedFirst) {
        console.error("Expected '" + expectedFirst + "' to be first, got '" + firstResult + "'");
        process.exit(1);
    }

    // Ensure all items have placeholders
    globalList.children.forEach(li => {
        if (!li._searchPlaceholder) {
            console.error("Item missing search placeholder");
            process.exit(1);
        }
    });
}

// 1. Search for 'github'
testCycle('github', 2, 'GitHub');

// 2. Search for 'docs' (changes matches)
testCycle('docs', 1, 'Some docs');

// 3. Search for 'github' again
testCycle('github', 2, 'GitHub');

// 4. Clear search
testCycle('', 0, '');

// Verify items were restored (placeholders removed)
global.mockItems.forEach(li => {
    if (li._searchPlaceholder !== null) {
        console.error("Search placeholder not cleaned up");
        process.exit(1);
    }
});

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
