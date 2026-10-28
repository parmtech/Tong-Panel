const maxFileBytes = 16 * 1024 * 1024;
const timeoutMilliseconds = 10000;

const responseMessages = new Map([
    [401, 'Sign in with a secure browser session to browse files.'],
    [403, 'Your account cannot access these files.'],
    [404, 'That file or folder is not available.'],
    [413, 'The file or folder listing exceeds the supported limit.'],
    [503, 'File access is not enabled on this server yet.'],
]);

export class FilesExplorerError extends Error {
    constructor(message) {
        super(message);
        this.name = 'FilesExplorerError';
    }
}

export function validateFilesPath(path) {
    if (typeof path !== 'string' || path.startsWith('/') || path.includes('\\'))
        throw new FilesExplorerError('The selected path is invalid.');
    if (path === '')
        return '';
    const parts = path.split('/');
    if (parts.some(part => part === '' || part === '.' || part === '..' || /[\u0000-\u001f\u007f]/u.test(part)))
        throw new FilesExplorerError('The selected path is invalid.');
    return path;
}

function filesURL(path, content = false) {
    const params = new URLSearchParams({ path: validateFilesPath(path) });
    return `/api/v1/files${content ? '/content' : ''}?${params.toString()}`;
}

async function withTimeout(fetcher, url, options, consume, timeoutMillisecondsOverride) {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), timeoutMillisecondsOverride ?? timeoutMilliseconds);
    try {
        const response = await fetcher(url, { ...options, signal: controller.signal });
        return await consume(response);
    }
    finally {
        clearTimeout(timeout);
    }
}

function failureMessage(status) {
    return responseMessages.get(status) ?? 'Files could not be loaded. Check the connection and try again.';
}

export async function listTenantFiles(path = '', fetcher = fetch, timeoutOverride) {
    try {
        return await withTimeout(fetcher, filesURL(path), {
            method: 'GET',
            credentials: 'same-origin',
            cache: 'no-store',
            headers: { Accept: 'application/json' },
        }, async response => {
            if (!response.ok)
                throw new FilesExplorerError(failureMessage(response.status));
            let body;
            try {
                body = await response.json();
            }
            catch (error) {
                if (error?.name === 'AbortError')
                    throw error;
                throw new FilesExplorerError('The server returned an unexpected file listing.');
            }
            if (!body || !Array.isArray(body.entries) || body.entries.length > 1000)
                throw new FilesExplorerError('The server returned an unexpected file listing.');
            const entries = body.entries.map(entry => {
                if (!entry || typeof entry !== 'object' || typeof entry.name !== 'string' ||
                    entry.name === '' || entry.name === '.' || entry.name === '..' ||
                    entry.name.includes('/') || entry.name.includes('\\') ||
                    /[\u0000-\u001f\u007f]/u.test(entry.name) ||
                    !['file', 'directory', 'symlink', 'other'].includes(entry.kind) ||
                    !Number.isSafeInteger(entry.size) || entry.size < 0) {
                    throw new FilesExplorerError('The server returned an unexpected file listing.');
                }
                return { name: entry.name, kind: entry.kind, size: entry.size };
            });
            return entries;
        }, timeoutOverride);
    }
    catch (error) {
        if (error instanceof FilesExplorerError)
            throw error;
        if (error?.name === 'AbortError')
            throw new FilesExplorerError('The file request timed out. Try again.');
        throw new FilesExplorerError('Files could not be reached. Check your connection and try again.');
    }
}

export async function downloadTenantFile(path, fetcher = fetch, timeoutOverride) {
    const safePath = validateFilesPath(path);
    if (!safePath || safePath.split('/').some(part => part === ''))
        throw new FilesExplorerError('Choose a file to download.');
    try {
        return await withTimeout(fetcher, filesURL(safePath, true), {
            method: 'GET',
            credentials: 'same-origin',
            cache: 'no-store',
            headers: { Accept: 'application/octet-stream' },
        }, async response => {
            if (!response.ok)
                throw new FilesExplorerError(failureMessage(response.status));
            if (response.headers?.get?.('Content-Type')?.toLowerCase() !== 'application/octet-stream')
                throw new FilesExplorerError('The server returned an unexpected file format.');
            let bytes;
            try {
                bytes = await response.arrayBuffer();
            }
            catch (error) {
                if (error?.name === 'AbortError')
                    throw error;
                throw new FilesExplorerError('The file response could not be read.');
            }
            if (!(bytes instanceof ArrayBuffer) || bytes.byteLength > maxFileBytes)
                throw new FilesExplorerError('The file exceeds the supported download limit.');
            return {
                name: safePath.slice(safePath.lastIndexOf('/') + 1),
                blob: new Blob([bytes], { type: 'application/octet-stream' }),
            };
        }, timeoutOverride);
    }
    catch (error) {
        if (error instanceof FilesExplorerError)
            throw error;
        if (error?.name === 'AbortError')
            throw new FilesExplorerError('The download timed out. Try again.');
        throw new FilesExplorerError('The file could not be reached. Check your connection and try again.');
    }
}
