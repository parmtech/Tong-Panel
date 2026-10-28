const siteIDPattern = /^[A-Za-z0-9_-]{1,63}$/;
class SitePreviewError extends Error {}

const responseMessages = {
    unauthorized: 'Sign in with a secure browser session to view this site preview.',
    forbidden: 'Your account cannot view this site preview.',
    not_found: 'This site is not available in your workspace.',
    site_not_ready: 'Verify the site’s primary domain before previewing its configuration.',
    site_preview_unavailable: 'The read-only site preview is not enabled on this server.',
};

export async function loadNginxPreview(siteID, fetcher = globalThis.fetch) {
    if (!siteIDPattern.test(siteID))
        throw new SitePreviewError('Enter a site ID using letters, numbers, hyphens or underscores.');
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 10000);
    try {
        const response = await fetcher(`/api/v1/sites/${encodeURIComponent(siteID)}/preview/nginx`, {
            method: 'GET',
            credentials: 'same-origin',
            cache: 'no-store',
            headers: { Accept: 'text/plain' },
            signal: controller.signal,
        });
        if (!response.ok) {
            let code = '';
            try {
                code = (await response.json())?.error?.code ?? '';
            }
            catch {
                // Use a fixed message for malformed or non-JSON error bodies.
            }
            throw new SitePreviewError(responseMessages[code] ?? 'The site preview could not be loaded. Try again.');
        }
        if (!response.headers?.get('Content-Type')?.toLowerCase().startsWith('text/plain'))
            throw new SitePreviewError('The server returned an unexpected preview format.');
        return await response.text();
    }
    catch (error) {
        if (error instanceof SitePreviewError)
            throw error;
        if (error instanceof Error && error.name === 'AbortError')
            throw new SitePreviewError('The preview request timed out. Try again.');
        throw new SitePreviewError('The site preview could not be reached. Check your connection and try again.');
    }
    finally {
        clearTimeout(timeout);
    }
}
