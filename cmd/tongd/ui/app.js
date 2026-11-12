import { loadPanelStatus } from './panel-status.mjs';
import { loadNginxPreview } from './site-preview.mjs';
import { loadSitesPage } from './sites-list.mjs';
import { mountACMEPreflight } from './acme-preflight.mjs';
import { downloadTenantFile, listTenantFiles } from './files-explorer.mjs';
import { loadMyJobsPage, loadMyJob, MyJobsError } from './my-jobs.mjs';
mountACMEPreflight();
const tiles = [
    { title: 'Websites', icon: '🌐', subtitle: 'Sites & PHP', stage: 'P2', searchTerms: ['PHP version'] },
    { title: 'Domains', icon: '🪐', subtitle: 'Aliases & redirects', stage: 'P2' },
    { title: 'Applications', icon: '🧩', subtitle: 'Node.js, Python & more', stage: 'P3', searchTerms: ['new WordPress', 'Node app'] },
    { title: 'File Explorer', icon: '📁', subtitle: 'Files, uploads & editor', stage: 'P2' },
    { title: 'Databases', icon: '🗃️', subtitle: 'MySQL & PostgreSQL', stage: 'P2/P4' },
    { title: 'Email', icon: '✉️', subtitle: 'Mailboxes & delivery', stage: 'P4', searchTerms: ['make email'] },
    { title: 'SSL & HTTPS', icon: '🔒', subtitle: 'Certificates & renewals', stage: 'P2', searchTerms: ['SSL redirect'] },
    { title: 'DNS', icon: '📡', subtitle: 'Records & nameservers', stage: 'P4' },
    { title: 'Backups', icon: '💾', subtitle: 'Back up & restore', stage: 'P2' },
    { title: 'Server Health', icon: '📊', subtitle: 'System monitoring', stage: 'P1+' },
    { title: 'User Accounts', icon: '👥', subtitle: 'Roles & packages', stage: 'P1/P4' },
    { title: 'Marketplace', icon: '🎁', subtitle: 'One-click app recipes', stage: 'P3+' },
];
const list = document.querySelector('#tiles');
const search = document.querySelector('#search');
const status = document.querySelector('#status');
const statusVersion = document.querySelector('#status-version');
const statusChecked = document.querySelector('#status-checked');
const refreshStatusButton = document.querySelector('#refresh-status');
const theme = document.querySelector('#theme');
const intro = document.querySelector('#intro');
const sitePreviewForm = document.querySelector('#site-preview-form');
const sitePreviewID = document.querySelector('#site-preview-id');
const sitePreviewStatus = document.querySelector('#site-preview-status');
const sitePreviewOutput = document.querySelector('#site-preview-output');
const sitePreviewSubmit = document.querySelector('#site-preview-submit');
let sitePreviewRequest = 0;
const normalizeSearchText = (text) => text.trim().toLowerCase().replace(/\s+/g, ' ');
const filterText = (text) => {
    if (!list)
        return;
    list.replaceChildren();
    const needle = normalizeSearchText(text);
    const shown = tiles.filter(t => normalizeSearchText(`${t.title} ${t.subtitle} ${(t.searchTerms ?? []).join(' ')}`).includes(needle));
    for (const t of shown) {
        const card = document.createElement('article');
        card.className = 'tile';
        const ico = document.createElement('span');
        ico.className = 'tile-icon';
        ico.textContent = t.icon;
        ico.setAttribute('aria-hidden', 'true');
        const head = document.createElement('h3');
        head.textContent = t.title;
        const detail = document.createElement('p');
        detail.textContent = t.subtitle;
        const tag = document.createElement('span');
        tag.className = 'planned';
        tag.textContent = `Planned · ${t.stage}`;
        card.append(ico, head, detail, tag);
        list.appendChild(card);
    }
    const counter = document.querySelector('#count');
    if (counter)
        counter.textContent = `${shown.length} module${shown.length === 1 ? '' : 's'} planned`;
};
search?.addEventListener('input', () => filterText(search.value));
filterText('');
theme?.addEventListener('click', () => {
    const dark = document.documentElement.dataset.theme !== 'dark';
    document.documentElement.dataset.theme = dark ? 'dark' : 'light';
    theme.setAttribute('aria-label', dark ? 'Switch to light mode' : 'Switch to dark mode');
});
document.querySelector('#tour')?.addEventListener('click', () => intro?.showModal());
document.querySelector('#close-tour')?.addEventListener('click', () => intro?.close());
async function refreshStatus() {
    if (!status || !refreshStatusButton)
        return;
    refreshStatusButton.disabled = true;
    refreshStatusButton.setAttribute('aria-busy', 'true');
    status.textContent = 'Checking panel process availability…';
    status.dataset.state = 'unknown';
    try {
        const result = await loadPanelStatus();
        status.textContent = result.message;
        status.dataset.state = result.state;
        if (statusVersion)
            statusVersion.textContent = result.version ?? '';
        if (statusChecked)
            statusChecked.textContent = new Intl.DateTimeFormat(undefined, { hour: 'numeric', minute: '2-digit' }).format(result.checkedAt);
    }
    catch {
        status.textContent = 'Panel status could not be checked. Retry to check again.';
        status.dataset.state = 'unknown';
        if (statusVersion)
            statusVersion.textContent = '';
        if (statusChecked)
            statusChecked.textContent = '—';
    }
    finally {
        refreshStatusButton.disabled = false;
        refreshStatusButton.removeAttribute('aria-busy');
    }
}
refreshStatusButton?.addEventListener('click', () => void refreshStatus());
void refreshStatus();
const sitesLoad = document.querySelector('#sites-load');
const sitesStatus = document.querySelector('#sites-status');
const sitesList = document.querySelector('#sites-list');
const sitesPager = document.querySelector('#sites-pagination');
const sitesPrevious = document.querySelector('#sites-previous');
const sitesNext = document.querySelector('#sites-next');
const sitesPageLabel = document.querySelector('#sites-page');
let sitesCursors = [''], sitesNextAfter = '', sitesRequest = 0;
let sitesController;
async function showSites(cursors, moveFocus = false) {
    if (!sitesStatus || !sitesList || !sitesLoad)
        return;
    const request = ++sitesRequest;
    sitesController?.abort();
    sitesController = new AbortController();
    const controller = sitesController;
    sitesList.replaceChildren();
    sitesList.setAttribute('aria-busy', 'true');
    if (sitesPager)
        sitesPager.hidden = true;
    sitesStatus.textContent = 'Loading sites…';
    sitesStatus.dataset.state = 'loading';
    if (moveFocus)
        sitesStatus.focus();
    try {
        const page = await loadSitesPage(cursors[cursors.length - 1], undefined, controller.signal);
        if (request !== sitesRequest || controller.signal.aborted)
            return;
        sitesCursors = cursors;
        sitesNextAfter = page.next_after;
        for (const site of page.sites) {
            const row = document.createElement('li');
            row.className = 'job-entry';
            const label = document.createElement('span');
            label.textContent = `${site.site_id} · ${site.primary_domain} · ${site.ownership_state}`;
            const button = document.createElement('button');
            button.type = 'button';
            button.textContent = 'Use for preview';
            button.setAttribute('aria-label', `Use ${site.site_id} for preview`);
            button.addEventListener('click', () => {
                ++sitePreviewRequest;
                if (sitePreviewOutput) {
                    sitePreviewOutput.hidden = true;
                    sitePreviewOutput.textContent = '';
                }
                if (sitePreviewStatus) {
                    sitePreviewStatus.dataset.state = '';
                    sitePreviewStatus.textContent = 'No preview loaded.';
                }
                if (sitePreviewID) {
                    sitePreviewID.value = site.site_id;
                    sitePreviewID.focus();
                }
            });
            row.append(label, button);
            sitesList.append(row);
        }
        sitesStatus.dataset.state = page.sites.length ? 'available' : 'empty';
        sitesStatus.textContent = page.sites.length ? `Page ${cursors.length}: ${page.sites.length} catalog sites. Domain status does not confirm hosting is provisioned.` : 'No sites found in this workspace.';
        if (sitesPager && sitesPrevious && sitesNext && sitesPageLabel) {
            sitesPrevious.disabled = cursors.length === 1;
            sitesNext.disabled = !page.next_after;
            sitesPageLabel.textContent = `Page ${cursors.length}`;
            sitesPager.hidden = cursors.length === 1 && !page.next_after;
        }
    }
    catch (error) {
        if (request !== sitesRequest || controller.signal.aborted)
            return;
        sitesCursors = [''];
        sitesNextAfter = '';
        sitesStatus.dataset.state = 'unavailable';
        sitesStatus.textContent = error instanceof Error ? error.message : 'Sites could not be loaded. Try again.';
    }
    finally {
        if (request === sitesRequest)
            sitesList.removeAttribute('aria-busy');
    }
}
sitesLoad?.addEventListener('click', () => void showSites(['']));
sitesPrevious?.addEventListener('click', () => { if (sitesCursors.length > 1)
    void showSites(sitesCursors.slice(0, -1), true); });
sitesNext?.addEventListener('click', () => { if (sitesNextAfter)
    void showSites([...sitesCursors, sitesNextAfter], true); });
sitePreviewForm?.addEventListener('submit', async (event) => {
    event.preventDefault();
    if (!sitePreviewID || !sitePreviewStatus || !sitePreviewOutput || !sitePreviewSubmit)
        return;
    const request = ++sitePreviewRequest;
    sitePreviewSubmit.disabled = true;
    sitePreviewSubmit.setAttribute('aria-busy', 'true');
    sitePreviewStatus.dataset.state = 'loading';
    sitePreviewStatus.textContent = 'Loading read-only preview…';
    sitePreviewOutput.hidden = true;
    sitePreviewOutput.textContent = '';
    try {
        const config = await loadNginxPreview(sitePreviewID.value);
        if (request !== sitePreviewRequest)
            return;
        sitePreviewOutput.textContent = config;
        sitePreviewOutput.hidden = false;
        sitePreviewStatus.dataset.state = 'available';
        sitePreviewStatus.textContent = 'Preview loaded. Configuration has not been applied.';
    }
    catch (error) {
        if (request !== sitePreviewRequest)
            return;
        sitePreviewStatus.dataset.state = 'unavailable';
        sitePreviewStatus.textContent = error instanceof Error ? error.message : 'Preview unavailable. Try again.';
    }
    finally {
        sitePreviewSubmit.disabled = false;
        sitePreviewSubmit.removeAttribute('aria-busy');
    }
});
const jobsRefresh = document.querySelector('#jobs-refresh');
const jobsStatus = document.querySelector('#jobs-status');
const jobsNotices = document.querySelector('#jobs-notifications');
const jobsList = document.querySelector('#jobs-list');
const jobsPagination = document.querySelector('#jobs-pagination');
const jobsPrevious = document.querySelector('#jobs-previous');
const jobsNext = document.querySelector('#jobs-next');
const jobsPage = document.querySelector('#jobs-page');
const jobDetail = document.querySelector('#job-detail');
const jobDetailTitle = document.querySelector('#job-detail-title');
const jobDetailStatus = document.querySelector('#job-detail-status');
const jobDetailFields = document.querySelector('#job-detail-fields');
const jobDetailRetry = document.querySelector('#job-detail-retry');
let jobsRequest = 0;
let jobStateScope = '';
let detailRequest = 0;
let jobsController;
let detailController;
let selectedJob;
let jobsCursors = [''];
let jobsNextAfter = '';
const jobStates = new Map();
function renderJobChanges(jobs) {
    const scope = jobs.length && jobs.every(job => job.requested_by && job.tenant_id === jobs[0].tenant_id && job.requested_by === jobs[0].requested_by)
        ? `${jobs[0].tenant_id}\u0000${jobs[0].requested_by}` : '';
    if (scope !== jobStateScope) {
        jobStates.clear();
        jobsNotices?.replaceChildren();
        jobStateScope = scope;
    }
    if (!scope)
        return;
    const changes = [];
    for (const job of jobs) {
        const previous = jobStates.get(job.id);
        if (previous !== undefined && previous !== job.state)
            changes.push({ id: job.id, kind: job.kind, previous, current: job.state });
        if (!jobStates.has(job.id) && jobStates.size >= 5000)
            jobStates.delete(jobStates.keys().next().value);
        jobStates.set(job.id, job.state);
    }
    if (!jobsNotices || changes.length === 0)
        return;
    for (const change of changes) {
        const item = document.createElement('li');
        const message = document.createElement('span');
        message.textContent = `Job ${change.id} (${change.kind}) changed from ${change.previous} to ${change.current}.`;
        const dismiss = document.createElement('button');
        dismiss.type = 'button';
        dismiss.textContent = 'Dismiss';
        dismiss.setAttribute('aria-label', `Dismiss update for job ${change.id}`);
        dismiss.addEventListener('click', () => item.remove());
        item.append(message, dismiss);
        jobsNotices.prepend(item);
    }
    while (jobsNotices.children.length > 5)
        jobsNotices.lastElementChild?.remove();
}
function jobsErrorCode(error) {
    return error instanceof MyJobsError ? error.code : 'unavailable';
}
function jobsErrorMessage(code) {
    switch (code) {
        case 'session_required': return 'A secure browser session is required. Public sign-in is not available in this preview.';
        case 'auth_unconfigured': return 'Job access is not configured on this panel. Public sign-in is not available in this preview.';
        case 'forbidden': return 'Your current session cannot read jobs. Refresh after your access changes.';
        case 'not_found': return 'This job was not found for your current access. Retry or refresh the list.';
        case 'jobs_unconfigured': return 'Job storage is not enabled on this panel. Refresh jobs to retry.';
        case 'jobs_unavailable': return 'Job storage is unavailable. Try again.';
        case 'timeout': return 'The job request timed out. Try again.';
        case 'invalid_response': return 'The panel returned an unsupported job response. Try again.';
        default: return 'The job request could not be completed. Try again.';
    }
}
function isJobAccessFailure(code) {
    return code === 'session_required' || code === 'auth_unconfigured' || code === 'forbidden';
}
function clearJobDetail() {
    selectedJob = undefined;
    ++detailRequest;
    detailController?.abort();
    jobDetailFields?.replaceChildren();
    if (jobDetail) {
        jobDetail.hidden = true;
        jobDetail.removeAttribute('aria-busy');
    }
    if (jobDetailStatus)
        jobDetailStatus.textContent = '';
    if (jobDetailRetry)
        jobDetailRetry.hidden = true;
}
function clearJobsForAccessFailure(code) {
    ++jobsRequest;
    jobsController?.abort();
    jobsList?.replaceChildren();
    jobsList?.removeAttribute('aria-busy');
    jobsCursors = [''];
    jobsNextAfter = '';
    if (jobsPagination)
        jobsPagination.hidden = true;
    clearJobDetail();
    jobStates.clear();
    jobStateScope = '';
    jobsNotices?.replaceChildren();
    if (jobsStatus) {
        jobsStatus.textContent = jobsErrorMessage(code);
        jobsStatus.dataset.state = code;
    }
    if (jobsRefresh) {
        jobsRefresh.textContent = 'Refresh jobs';
        jobsRefresh.removeAttribute('aria-busy');
    }
}
function jobDate(raw) {
    const date = new Date(raw);
    return Number.isNaN(date.valueOf()) ? raw : date.toLocaleString();
}
function renderJobs(jobs) {
    if (!jobsList)
        return;
    jobsList.replaceChildren();
    for (const job of jobs) {
        const row = document.createElement('li');
        row.className = 'job-entry';
        const copy = document.createElement('div');
        copy.className = 'job-entry-copy';
        const kind = document.createElement('strong');
        kind.textContent = job.kind;
        const id = document.createElement('span');
        id.className = 'job-id';
        id.textContent = job.id;
        const summary = document.createElement('span');
        summary.className = 'job-summary';
        summary.textContent = `${job.state} · Created ${jobDate(job.created_at)}`;
        const button = document.createElement('button');
        button.type = 'button';
        button.textContent = 'View details';
        button.dataset.jobId = job.id;
        button.setAttribute('aria-label', `View details for job ${job.id}`);
        button.setAttribute('aria-controls', 'job-detail');
        button.setAttribute('aria-pressed', 'false');
        button.addEventListener('click', () => void selectJob(job.id, true));
        copy.append(kind, id, summary);
        row.append(copy, button);
        jobsList.append(row);
    }
}
async function showJobsPage(cursors, moveFocus = false) {
    if (!jobsStatus || !jobsList || !jobsRefresh)
        return;
    const request = ++jobsRequest;
    jobsController?.abort();
    jobsController = new AbortController();
    const controller = jobsController;
    clearJobDetail();
    jobsList.replaceChildren();
    jobsList.setAttribute('aria-busy', 'true');
    if (jobsPagination)
        jobsPagination.hidden = true;
    jobsRefresh.setAttribute('aria-busy', 'true');
    jobsRefresh.textContent = 'Refresh jobs';
    jobsStatus.dataset.state = 'loading';
    jobsStatus.textContent = 'Loading jobs for your current session…';
    if (moveFocus)
        jobsStatus.focus();
    try {
        const page = await loadMyJobsPage(cursors[cursors.length - 1], undefined, { signal: controller.signal });
        if (request !== jobsRequest || controller.signal.aborted)
            return;
        renderJobChanges(page.jobs);
        jobsCursors = cursors;
        jobsNextAfter = page.next_after;
        renderJobs(page.jobs);
        jobsStatus.dataset.state = page.jobs.length ? 'available' : 'empty';
        jobsStatus.textContent = page.jobs.length ? `Page ${cursors.length}: ${page.jobs.length} ${page.jobs.length === 1 ? 'job' : 'jobs'} returned for your current session. Select a job to read its metadata.` : 'No jobs returned for your current access.';
        if (jobsPagination && jobsPrevious && jobsNext && jobsPage) {
            jobsPrevious.disabled = cursors.length === 1;
            jobsNext.disabled = !page.next_after;
            jobsPage.textContent = `Page ${cursors.length}`;
            jobsPagination.hidden = cursors.length === 1 && !page.next_after;
        }
    }
    catch (error) {
        if (request !== jobsRequest || controller.signal.aborted)
            return;
        const code = jobsErrorCode(error);
        if (isJobAccessFailure(code)) {
            clearJobsForAccessFailure(code);
            return;
        }
        jobsStatus.dataset.state = code;
        jobsStatus.textContent = jobsErrorMessage(code);
    }
    finally {
        if (request === jobsRequest) {
            jobsList.removeAttribute('aria-busy');
            jobsRefresh.removeAttribute('aria-busy');
        }
    }
}
async function selectJob(id, moveFocus) {
    if (!jobDetail || !jobDetailStatus || !jobDetailFields || !jobDetailTitle || !jobDetailRetry)
        return;
    const request = ++detailRequest;
    detailController?.abort();
    detailController = new AbortController();
    const controller = detailController;
    selectedJob = id;
    for (const button of Array.from(jobsList?.querySelectorAll('button[data-job-id]') ?? []))
        button.setAttribute('aria-pressed', String(button.dataset.jobId === id));
    jobDetail.hidden = false;
    jobDetail.setAttribute('aria-busy', 'true');
    jobDetailFields.replaceChildren();
    jobDetailRetry.hidden = true;
    jobDetailTitle.textContent = `Job ${id}`;
    jobDetailStatus.dataset.state = 'loading';
    jobDetailStatus.textContent = 'Loading job metadata…';
    if (moveFocus)
        jobDetailTitle.focus();
    try {
        const job = await loadMyJob(id, undefined, { signal: controller.signal });
        if (request !== detailRequest || controller.signal.aborted)
            return;
        const values = [['Job ID', job.id], ['Kind', job.kind], ['State', job.state], ['Tenant', job.tenant_id], ['Requester', job.requested_by ?? 'Not recorded'], ['Created', jobDate(job.created_at)], ['Updated', jobDate(job.updated_at)], ['Idempotency key', job.idempotency_key]];
        for (const [label, value] of values) {
            const term = document.createElement('dt');
            term.textContent = label;
            const description = document.createElement('dd');
            description.textContent = value;
            jobDetailFields.append(term, description);
        }
        jobDetailStatus.dataset.state = 'available';
        jobDetailStatus.textContent = 'Recorded queue metadata. A completed validation.noop job does not provision a site or change the host.';
    }
    catch (error) {
        if (request !== detailRequest || controller.signal.aborted)
            return;
        const code = jobsErrorCode(error);
        if (isJobAccessFailure(code)) {
            clearJobsForAccessFailure(code);
            jobsRefresh?.focus();
            return;
        }
        jobDetailStatus.dataset.state = code;
        jobDetailStatus.textContent = jobsErrorMessage(code);
        jobDetailRetry.hidden = false;
    }
    finally {
        if (request === detailRequest)
            jobDetail.removeAttribute('aria-busy');
    }
}
jobsRefresh?.addEventListener('click', () => void showJobsPage(['']));
jobsPrevious?.addEventListener('click', () => { if (jobsCursors.length > 1)
    void showJobsPage(jobsCursors.slice(0, -1), true); });
jobsNext?.addEventListener('click', () => { if (jobsNextAfter)
    void showJobsPage([...jobsCursors, jobsNextAfter], true); });
jobDetailRetry?.addEventListener('click', () => { if (selectedJob)
    void selectJob(selectedJob, true); });
const filesRefresh = document.querySelector('#files-refresh');
const filesStatus = document.querySelector('#files-status');
const filesList = document.querySelector('#files-list');
const filesBreadcrumbs = document.querySelector('#files-breadcrumbs');
let filesPath = [];
function formatFileSize(size) {
    if (size < 1024)
        return `${size} B`;
    if (size < 1024 * 1024)
        return `${(size / 1024).toFixed(1)} KB`;
    return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}
function renderFilesBreadcrumbs() {
    if (!filesBreadcrumbs)
        return;
    filesBreadcrumbs.replaceChildren();
    const labels = ['Files', ...filesPath];
    labels.forEach((label, index) => {
        if (index > 0) {
            const separator = document.createElement('span');
            separator.textContent = '›';
            separator.setAttribute('aria-hidden', 'true');
            filesBreadcrumbs.append(separator);
        }
        const button = document.createElement('button');
        button.type = 'button';
        button.textContent = label;
        button.setAttribute('aria-current', index === labels.length - 1 ? 'location' : 'false');
        button.addEventListener('click', () => {
            filesPath = filesPath.slice(0, index);
            void refreshFiles();
        });
        filesBreadcrumbs.append(button);
    });
}
function renderFilesEntries(entries) {
    if (!filesList)
        return;
    filesList.replaceChildren();
    const sorted = [...entries].sort((a, b) => {
        const aDirectory = a.kind === 'directory' ? 0 : 1;
        const bDirectory = b.kind === 'directory' ? 0 : 1;
        return aDirectory - bDirectory || a.name.localeCompare(b.name);
    });
    for (const entry of sorted) {
        const item = document.createElement('li');
        item.className = 'file-entry';
        const name = document.createElement('span');
        name.className = 'file-entry-name';
        name.textContent = entry.kind === 'directory' ? `📁 ${entry.name}` : entry.name;
        const detail = document.createElement('span');
        detail.className = 'file-entry-detail';
        detail.textContent = entry.kind === 'file' ? formatFileSize(entry.size) : entry.kind === 'directory' ? 'Folder' : entry.kind === 'symlink' ? 'Link · not followed' : 'Unsupported entry';
        item.append(name, detail);
        if (entry.kind === 'directory') {
            const open = document.createElement('button');
            open.type = 'button';
            open.textContent = 'Open folder';
            open.setAttribute('aria-label', `Open folder ${entry.name}`);
            open.addEventListener('click', () => {
                filesPath = [...filesPath, entry.name];
                void refreshFiles();
            });
            item.append(open);
        }
        else if (entry.kind === 'file') {
            const download = document.createElement('button');
            download.type = 'button';
            download.textContent = 'Download';
            download.setAttribute('aria-label', `Download ${entry.name}`);
            download.addEventListener('click', () => void downloadFile(entry.name, download));
            item.append(download);
        }
        filesList.append(item);
    }
    if (sorted.length === 0) {
        const empty = document.createElement('li');
        empty.className = 'file-entry';
        empty.textContent = 'This folder is empty.';
        filesList.append(empty);
    }
}
async function refreshFiles() {
    if (!filesRefresh || !filesStatus || !filesList)
        return;
    filesRefresh.disabled = true;
    filesRefresh.setAttribute('aria-busy', 'true');
    filesStatus.dataset.state = 'loading';
    filesStatus.textContent = 'Loading files…';
    try {
        const entries = await listTenantFiles(filesPath.join('/'));
        renderFilesBreadcrumbs();
        renderFilesEntries(entries);
        filesStatus.dataset.state = 'available';
        filesStatus.textContent = `${entries.length} ${entries.length === 1 ? 'item' : 'items'} loaded. Links are never followed.`;
    }
    catch (error) {
        renderFilesBreadcrumbs();
        filesList.replaceChildren();
        filesStatus.dataset.state = 'unavailable';
        filesStatus.textContent = error instanceof Error ? error.message : 'Files could not be loaded. Try again.';
    }
    finally {
        filesRefresh.disabled = false;
        filesRefresh.removeAttribute('aria-busy');
    }
}
async function downloadFile(name, button) {
    button.disabled = true;
    try {
        const file = await downloadTenantFile([...filesPath, name].join('/'));
        const objectURL = URL.createObjectURL(file.blob);
        const link = document.createElement('a');
        link.href = objectURL;
        link.download = file.name;
        link.hidden = true;
        document.body.append(link);
        link.click();
        link.remove();
        window.setTimeout(() => URL.revokeObjectURL(objectURL), 1000);
        if (filesStatus) {
            filesStatus.dataset.state = 'available';
            filesStatus.textContent = `Downloaded ${file.name}.`;
        }
    }
    catch (error) {
        if (filesStatus) {
            filesStatus.dataset.state = 'unavailable';
            filesStatus.textContent = error instanceof Error ? error.message : 'The file could not be downloaded. Try again.';
        }
    }
    finally {
        button.disabled = false;
    }
}
filesRefresh?.addEventListener('click', () => void refreshFiles());
renderFilesBreadcrumbs();
