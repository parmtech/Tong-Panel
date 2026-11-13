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
    return listFiles(path, null, fetcher, timeoutOverride);
}

export async function listTenantFilesPage(path = '', after = '', fetcher = fetch, timeoutOverride) {
    if (typeof after !== 'string' || after.length > 255 || /[\/\\\u0000-\u001f\u007f]/u.test(after) || after === '.' || after === '..')
        throw new FilesExplorerError('The file page is invalid. Refresh the folder.');
    return listFiles(path, after, fetcher, timeoutOverride);
}

async function listFiles(path, after, fetcher, timeoutOverride) {
    try {
        const url = filesURL(path) + (after === null ? '' : `&page=1&after=${encodeURIComponent(after)}`);
        return await withTimeout(fetcher, url, {
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
            if (after === null) return entries;
            if (typeof body.next_after !== 'string' || body.next_after.length > 255 || /[\/\\\u0000-\u001f\u007f]/u.test(body.next_after) || body.next_after === '.' || body.next_after === '..' || (body.next_after && body.next_after === after))
                throw new FilesExplorerError('The server returned an unexpected file listing.');
            return { entries, next_after: body.next_after };
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

// Text preview uses bounded streaming; never buffer an unbounded response.
export async function previewTenantText(path, fetcher = fetch, timeoutOverride) {
    const safePath = validateFilesPath(path);
    if (!safePath) throw new FilesExplorerError('Choose a file to preview.');
    const maxBytes = 256 * 1024;
    try {
        return await withTimeout(fetcher, filesURL(safePath, true), {
            method: 'GET', credentials: 'same-origin', cache: 'no-store',
            headers: { Accept: 'application/octet-stream' },
        }, async response => {
            if (!response.ok) throw new FilesExplorerError(failureMessage(response.status));
            if (response.headers?.get?.('Content-Type')?.toLowerCase() !== 'application/octet-stream')
                throw new FilesExplorerError('The server returned an unexpected file format.');
            const length = response.headers?.get?.('Content-Length');
            if (length !== null && length !== undefined && (!/^\d+$/u.test(length) || Number(length) > maxBytes)) {
                if (response.body?.cancel) void response.body.cancel().catch(() => {});
                throw new FilesExplorerError('This file is too large to preview.');
            }
            if (!response.body?.getReader) throw new FilesExplorerError('Text preview is unavailable in this browser.');
            const reader = response.body.getReader();
            const chunks = []; let total = 0;
            try {
                while (true) {
                    const {value, done} = await reader.read();
                    if (done) break;
                    if (!(value instanceof Uint8Array) || value.byteLength > maxBytes - total)
                        throw new FilesExplorerError('This file is too large to preview.');
                    total += value.byteLength;
                    chunks.push(value);
                }
            } catch (error) {
                void reader.cancel().catch(() => {});
                throw error;
            } finally { reader.releaseLock(); }
            const bytes = new Uint8Array(total); let offset = 0;
            for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.byteLength; }
            let text;
            try { text = new TextDecoder('utf-8', {fatal:true}).decode(bytes); }
            catch { throw new FilesExplorerError('Only UTF-8 text files can be previewed.'); }
            if (/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/u.test(text))
                throw new FilesExplorerError('Only UTF-8 text files can be previewed.');
            return text;
        }, timeoutOverride);
    } catch (error) {
        if (error instanceof FilesExplorerError) throw error;
        if (error?.name === 'AbortError') throw new FilesExplorerError('The file request timed out. Try again.');
        throw new FilesExplorerError('Files could not be reached. Check your connection and try again.');
    }
}
