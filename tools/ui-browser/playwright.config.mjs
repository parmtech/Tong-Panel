import { defineConfig } from '@playwright/test';

const channel = process.env.TONG_UI_BROWSER_CHANNEL;
if (channel && channel !== 'chrome' && channel !== 'msedge') throw new Error('Unsupported test browser channel');

export default defineConfig({
    testDir: '.',
    testMatch: 'jobs.spec.mjs',
    timeout: 30000,
    expect: { timeout: 5000 },
    workers: 1,
    retries: 0,
    reporter: 'list',
    use: {
        browserName: 'chromium',
        channel: channel || undefined,
        headless: true,
        ignoreHTTPSErrors: true, // Only the disposable loopback Go TLS fixture.
        trace: 'off',
        video: 'off',
        screenshot: 'only-on-failure',
    },
});
