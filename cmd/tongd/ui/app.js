import { loadPanelStatus } from './panel-status.mjs';
import { loadNginxPreview } from './site-preview.mjs';
import { downloadTenantFile, listTenantFiles } from './files-explorer.mjs';

const tiles = [
    { title: 'Websites', icon: '🌐', subtitle: 'Sites & PHP', stage: 'P2' },
    { title: 'Domains', icon: '🪐', subtitle: 'Aliases & redirects', stage: 'P2' },
    { title: 'Applications', icon: '🧩', subtitle: 'Node.js, Python & more', stage: 'P3' },
    { title: 'File Explorer', icon: '📁', subtitle: 'Files, uploads & editor', stage: 'P2' },
    { title: 'Databases', icon: '🗃️', subtitle: 'MySQL & PostgreSQL', stage: 'P2/P4' },
    { title: 'Email', icon: '✉️', subtitle: 'Mailboxes & delivery', stage: 'P4' },
    { title: 'SSL & HTTPS', icon: '🔒', subtitle: 'Certificates & renewals', stage: 'P2' },
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
const filesRefresh = document.querySelector('#files-refresh');
const filesStatus = document.querySelector('#files-status');
const filesList = document.querySelector('#files-list');
const filesBreadcrumbs = document.querySelector('#files-breadcrumbs');
const filterText = (text) => {
    if (!list)
        return;
    list.replaceChildren();
    const needle = text.trim().toLowerCase();
    const shown = tiles.filter(t => `${t.title} ${t.subtitle}`.toLowerCase().includes(needle));
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
        counter.textContent = `${shown.length} modules planned`;
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

sitePreviewForm?.addEventListener('submit', async (event) => {
    event.preventDefault();
    if (!sitePreviewID || !sitePreviewStatus || !sitePreviewOutput || !sitePreviewSubmit)
        return;
    sitePreviewSubmit.disabled = true;
    sitePreviewSubmit.setAttribute('aria-busy', 'true');
    sitePreviewStatus.dataset.state = 'loading';
    sitePreviewStatus.textContent = 'Loading read-only preview…';
    sitePreviewOutput.hidden = true;
    sitePreviewOutput.textContent = '';
    try {
        const config = await loadNginxPreview(sitePreviewID.value);
        sitePreviewOutput.textContent = config;
        sitePreviewOutput.hidden = false;
        sitePreviewStatus.dataset.state = 'available';
        sitePreviewStatus.textContent = 'Preview loaded. Configuration has not been applied.';
    }
    catch (error) {
        sitePreviewStatus.dataset.state = 'unavailable';
        sitePreviewStatus.textContent = error instanceof Error ? error.message : 'Preview unavailable. Try again.';
    }
    finally {
        sitePreviewSubmit.disabled = false;
        sitePreviewSubmit.removeAttribute('aria-busy');
    }
});

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
export {};
