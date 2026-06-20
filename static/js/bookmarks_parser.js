function extractCategoryByIndex(bookmarks, tabIndex, pageIndex, categoryIndex) {
    let categoryCount = -1;
    let result = [];
    let collecting = false;

    const lines = bookmarks.split('\n');

    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];

        const trimmed = line.trim();
        const lower = trimmed.toLowerCase();
        const isCategory = lower.startsWith("category:");

        if (isCategory) {
            categoryCount++;
            if (categoryCount === categoryIndex) {
                collecting = true;
            } else if (collecting) {
                collecting = false;
                break; // We've finished collecting the target category
            }
        } else if (collecting && (lower === "tab" || lower.startsWith("tab ") || lower.startsWith("tab:") || lower.startsWith("page") || lower === "column" || lower.startsWith("column:") || trimmed === "--")) {
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
