{{define "bookmarks_parser.js"}}
function extractCategoryByIndex(bookmarks, tabIndex, pageIndex, categoryIndex) {
    let tabCount = 0;
    let pageCount = 0;
    let categoryCount = 0;

    let inTargetTab = false;
    let inTargetPage = false;

    let result = [];
    let collecting = false;

    const lines = bookmarks.split('\n');

    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];

        const isTab = line.toLowerCase().startsWith("tab:");
        const isPage = line.toLowerCase().startsWith("page:");
        const isCategory = line.toLowerCase().startsWith("category:");

        if (isTab) {
            inTargetTab = (tabCount === tabIndex);
            if (inTargetTab) {
                pageCount = 0; // Reset page count for new tab
            } else {
                collecting = false; // Stop collecting if we leave the target tab
            }
            tabCount++;
        }

        if (inTargetTab && isPage) {
            inTargetPage = (pageCount === pageIndex);
            if (inTargetPage) {
                categoryCount = 0; // Reset category count for new page
            } else {
                collecting = false; // Stop collecting if we leave the target page
            }
            pageCount++;
        }

        if (inTargetTab && inTargetPage && isCategory) {
            if (categoryCount === categoryIndex) {
                collecting = true;
            } else if (collecting) {
                collecting = false;
                break; // We've finished collecting the target category
            }
            categoryCount++;
        } else if (collecting && (isTab || line.toLowerCase() === "tab" || isPage || line.toLowerCase() === "column" || line.toLowerCase().startsWith("column:") || line.trim() === "--")) {
            collecting = false;
            break;
        }

        if (collecting) {
            result.push(line);
        }
    }

    return result.join('\n');
}

function extractTabByIndex(bookmarks, tabIndex) {
    let tabCount = 0;
    let collecting = false;
    let result = [];

    const lines = bookmarks.split('\n');

    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];

        if (line.toLowerCase().startsWith("tab:")) {
            if (tabCount === tabIndex) {
                collecting = true;
            } else if (collecting) {
                collecting = false;
                break;
            }
            tabCount++;
        } else if (collecting && line.toLowerCase().startsWith("tab")) { // handles "Tab" (not just Tab:)
           if (line.toLowerCase() === "tab" || line.toLowerCase().startsWith("tab ")) {
             collecting = false;
             break;
           }
        }

        if (collecting) {
            result.push(line);
        }
    }

    return result.join('\n');
}

function extractPage(bookmarks, tabIndex, pageIndex) {
    let tabCount = 0;
    let pageCount = 0;

    let inTargetTab = false;
    let collecting = false;

    let result = [];

    const lines = bookmarks.split('\n');

    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];

        if (line.toLowerCase().startsWith("tab:")) {
            inTargetTab = (tabCount === tabIndex);
            if (inTargetTab) {
                pageCount = 0;
            } else {
                collecting = false;
            }
            tabCount++;
        }

        if (inTargetTab && line.toLowerCase().startsWith("page:")) {
            if (pageCount === pageIndex) {
                collecting = true;
            } else if (collecting) {
                collecting = false;
                break;
            }
            pageCount++;
        } else if (collecting && (line.toLowerCase().startsWith("tab:") || line.toLowerCase() === "tab" || line.trim() === "--")) {
            collecting = false;
            break;
        }

        if (collecting) {
            result.push(line);
        }
    }

    return result.join('\n');
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = { extractCategoryByIndex, extractTabByIndex, extractPage };
}
{{end}}
