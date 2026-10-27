const requestTimeoutMs = 3000;

function requestError(kind) {
    const error = new Error(kind);
    error.kind = kind;
    return error;
}

async function getJSON(fetcher, path, timeoutMs) {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), timeoutMs);
    try {
        const response = await fetcher(path, { cache: 'no-store', signal: controller.signal });
        if (!response?.ok)
            throw requestError('unavailable');
        let data;
        try {
            data = await response.json();
        }
        catch {
            throw requestError('invalid');
        }
        if (!data || typeof data !== 'object' || Array.isArray(data))
            throw requestError('invalid');
        return data;
    }
    catch (error) {
        if (error?.kind)
            throw error;
        if (error?.name === 'AbortError')
            throw requestError('timeout');
        throw requestError('unavailable');
    }
    finally {
        clearTimeout(timeout);
    }
}

/** Checks only the local panel API process, not host or hosted-service health. */
export async function loadPanelStatus(fetcher = fetch, timeoutMs = requestTimeoutMs) {
    const [healthResult, versionResult] = await Promise.allSettled([
        getJSON(fetcher, '/healthz', timeoutMs),
        getJSON(fetcher, '/api/v1/version', timeoutMs),
    ]);
    const checkedAt = new Date();
    if (healthResult.status === 'rejected') {
        const isInvalid = healthResult.reason?.kind === 'invalid';
        return {
            state: isInvalid ? 'unknown' : 'unavailable',
            message: isInvalid
                ? 'Panel response could not be verified. Retry to check again.'
                : 'Panel process could not be reached. Retry to check again.',
            checkedAt,
        };
    }
    if (healthResult.value.status !== 'ok') {
        return {
            state: 'unknown',
            message: 'Panel response could not be verified. Retry to check again.',
            checkedAt,
        };
    }
    if (versionResult.status === 'rejected'
        || versionResult.value.product !== 'Tong Panel'
        || typeof versionResult.value.stage !== 'string'
        || versionResult.value.stage.length === 0) {
        return {
            state: 'available',
            message: 'Panel process is responding; version information is unavailable.',
            checkedAt,
        };
    }
    return {
        state: 'available',
        message: 'Panel process is responding.',
        version: `${versionResult.value.product} · ${versionResult.value.stage}`,
        checkedAt,
    };
}
