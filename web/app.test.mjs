import test from 'node:test';
import assert from 'node:assert';
import { performSearch } from './app.mjs';

test('search ordering regression cases (#249)', (t) => {
    let items = [
        { title: 'apple', url: 'https://example.com/fruit', originalItem: 'AppleItem' },
        { title: 'banana docs', url: 'https://github.com/banana', originalItem: 'BananaDocsItem' },
        { title: 'some docs', url: 'https://github.com/issues', originalItem: 'SomeDocsItem' },
        { title: 'github', url: 'https://github.com', originalItem: 'GitHubItem' },
        { title: 'extra github', url: 'https://github.com/extra', originalItem: 'ExtraGitHubItem' }, // title+URL match for 'github'
        { title: 'docs repo', url: 'https://github.com/docs', originalItem: 'DocsRepoItem' },
    ];

    function runSearch(query) {
        let res = performSearch(items, query);
        return [...res.titleMatches, ...res.urlMatches];
    }

    let resGithub = runSearch('github');
    assert.deepStrictEqual(resGithub, [
        'GitHubItem', 'ExtraGitHubItem',
        'BananaDocsItem', 'SomeDocsItem', 'DocsRepoItem'
    ]);

    let resDocs = runSearch('docs');
    assert.deepStrictEqual(resDocs, [
        'BananaDocsItem', 'SomeDocsItem', 'DocsRepoItem'
    ]);

    let resApple = runSearch('apple');
    assert.deepStrictEqual(resApple, ['AppleItem']);

    let resIssues = runSearch('issues');
    assert.deepStrictEqual(resIssues, ['SomeDocsItem']);
});
