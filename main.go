package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/quanglewangle/qrzlook/db"
	"github.com/quanglewangle/qrzlook/qrz"
	"github.com/quanglewangle/qrzlook/terrain50"
)

var buildHash = "dev"
var writeToken string

type losEntry struct {
	Clear  bool    `json:"clear"`
	ObsLat float64 `json:"obs_lat,omitempty"`
	ObsLon float64 `json:"obs_lon,omitempty"`
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>ROC Locations</title>
<link rel="icon" type="image/svg+xml" href="data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCA1IDMiPjxyZWN0IHdpZHRoPSI1IiBoZWlnaHQ9IjMiIGZpbGw9ImJsYWNrIi8+PHJlY3QgeD0iMCIgeT0iMSIgd2lkdGg9IjUiIGhlaWdodD0iMSIgZmlsbD0id2hpdGUiLz48cmVjdCB4PSIyIiB5PSIwIiB3aWR0aD0iMSIgaGVpZ2h0PSIzIiBmaWxsPSJ3aGl0ZSIvPjwvc3ZnPg==">
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css"/>
<link rel="stylesheet" href="https://unpkg.com/leaflet.markercluster@1.5.3/dist/MarkerCluster.css"/>
<link rel="stylesheet" href="https://unpkg.com/leaflet.markercluster@1.5.3/dist/MarkerCluster.Default.css"/>
<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
<script src="https://unpkg.com/leaflet.markercluster@1.5.3/dist/leaflet.markercluster.js"></script>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f0f4f8; color: #1a202c; min-height: 100vh; }

  nav {
    background: #2b6cb0;
    color: white;
    padding: 0 1.5rem;
    display: flex;
    align-items: center;
    gap: 0;
    height: 52px;
  }
  nav .brand { font-weight: 700; font-size: 1.1rem; margin-right: 1.5rem; }
  nav a {
    color: rgba(255,255,255,0.8);
    text-decoration: none;
    padding: 0 1rem;
    height: 52px;
    line-height: 52px;
    font-size: 0.95rem;
    transition: background 0.15s;
  }
  nav a:hover, nav a.active { background: rgba(0,0,0,0.2); color: white; }

  .view { display: none; padding: 2rem 1rem; }
  .view.active { display: block; }

  /* Lookup view */
  #view-lookup { max-width: 520px; margin: 0 auto; }
  #view-lookup h2 { font-size: 1.3rem; color: #2d3748; margin-bottom: 1.25rem; }
  .search-form { display: flex; gap: 0.5rem; margin-bottom: 1.5rem; }
  .search-form input {
    flex: 1; padding: 0.7rem 1rem; font-size: 1.05rem;
    border: 2px solid #cbd5e0; border-radius: 8px; outline: none;
    text-transform: uppercase; letter-spacing: 0.05em;
  }
  .search-form input:focus { border-color: #4299e1; }
  .btn { padding: 0.7rem 1.25rem; font-size: 0.95rem; font-weight: 600; border: none; border-radius: 8px; cursor: pointer; transition: background 0.15s; }
  .btn-primary { background: #3182ce; color: white; }
  .btn-primary:hover { background: #2b6cb0; }
  .btn-primary:disabled { background: #a0aec0; cursor: not-allowed; }
  .btn-success { background: #38a169; color: white; }
  .btn-success:hover { background: #2f855a; }
  .btn-danger { background: #e53e3e; color: white; }
  .btn-danger:hover { background: #c53030; }
  .btn-sm { padding: 0.3rem 0.7rem; font-size: 0.8rem; }

  .card { background: white; border-radius: 12px; padding: 1.5rem; box-shadow: 0 2px 12px rgba(0,0,0,0.08); margin-bottom: 1rem; }
  .callsign-big { font-size: 2rem; font-weight: 800; color: #2b6cb0; letter-spacing: 0.05em; }
  .name-big { font-size: 1.15rem; color: #2d3748; font-weight: 500; margin: 0.2rem 0 1rem; }
  .divider { border: none; border-top: 1px solid #e2e8f0; margin: 1rem 0; }
  .fields { display: flex; flex-direction: column; gap: 0.55rem; }
  .field { display: flex; justify-content: space-between; gap: 1rem; }
  .field-label { font-size: 0.78rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.08em; color: #a0aec0; }
  .field-value { font-size: 0.92rem; color: #2d3748; text-align: right; }
  .map-link { display: inline-block; margin-top: 1rem; font-size: 0.85rem; color: #3182ce; text-decoration: none; }
  .map-link:hover { text-decoration: underline; }
  .error-box { background: #fff5f5; border: 1px solid #fed7d7; color: #c53030; padding: 1rem 1.25rem; border-radius: 8px; font-size: 0.92rem; }
  .spinner { width: 32px; height: 32px; border: 3px solid #e2e8f0; border-top-color: #3182ce; border-radius: 50%; animation: spin 0.7s linear infinite; margin: 1.5rem auto; }
  @keyframes spin { to { transform: rotate(360deg); } }

  /* Sites view */
  #view-sites { max-width: 800px; margin: 0 auto; }
  .sites-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
  .sites-header h2 { font-size: 1.3rem; color: #2d3748; }
  table { width: 100%; border-collapse: collapse; background: white; border-radius: 12px; overflow: hidden; box-shadow: 0 2px 12px rgba(0,0,0,0.08); font-size: 0.88rem; }
  th { background: #edf2f7; color: #718096; font-size: 0.72rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.08em; padding: 0.65rem 1rem; text-align: left; }
  td { padding: 0.6rem 1rem; border-top: 1px solid #f0f4f8; color: #2d3748; }
  tr:hover td { background: #f7fafc; }
  .cs-link { font-weight: 700; color: #2b6cb0; cursor: pointer; }
  .cs-link:hover { text-decoration: underline; }
  .actions { display: flex; gap: 0.4rem; }

  /* Map view */
  #view-map { padding: 0; }
  #map { height: calc(100vh - 52px); width: 100%; }
  .pin-drop-btn { padding:6px 10px; background:white; border:2px solid #ccc; border-radius:6px; cursor:pointer; font-size:0.82rem; font-weight:600; box-shadow:0 1px 5px rgba(0,0,0,0.2); }
  .pin-drop-btn.active { background:#22c55e; color:white; border-color:#16a34a; }

  /* Modal */
  .modal-overlay { display: none; position: fixed; inset: 0; background: rgba(0,0,0,0.5); z-index: 1000; align-items: center; justify-content: center; }
  .modal-overlay.open { display: flex; }
  .modal-box { background: white; border-radius: 12px; padding: 1.75rem; width: 100%; max-width: 420px; margin: 1rem; box-shadow: 0 8px 32px rgba(0,0,0,0.2); }
  .modal-box h3 { font-size: 1.15rem; margin-bottom: 1.25rem; color: #2d3748; }
  .form-group { margin-bottom: 1rem; }
  .form-group label { display: block; font-size: 0.8rem; font-weight: 600; color: #4a5568; margin-bottom: 0.3rem; text-transform: uppercase; letter-spacing: 0.06em; }
  .form-group input { width: 100%; padding: 0.6rem 0.8rem; border: 2px solid #cbd5e0; border-radius: 8px; font-size: 0.95rem; outline: none; }
  .form-group input:focus { border-color: #4299e1; }
  .modal-footer { display: flex; gap: 0.5rem; justify-content: flex-end; margin-top: 1.25rem; }
  .btn-ghost { background: #e2e8f0; color: #4a5568; }
  .btn-ghost:hover { background: #cbd5e0; }
</style>
</head>
<body>

<nav>
  <span class="brand">ROC Locations</span>
  <a href="#" class="active" data-view="lookup" title="Look up and edit">Lookup</a>
  <a href="#" data-view="pins">Pins</a>
  <a href="#" data-view="qths">QTHs</a>
  <a href="#" data-view="map">Map</a>
  <button id="token-btn" onclick="promptToken()" style="margin-left:auto;background:transparent;border:1px solid rgba(255,255,255,0.45);color:rgba(255,255,255,0.85);border-radius:5px;padding:0.2rem 0.65rem;cursor:pointer;font-size:0.8rem;font-family:inherit;"></button>
</nav>

<div id="view-lookup" class="view active">
  <h2>Callsign Lookup</h2>
  <form class="search-form" id="lookup-form">
    <input type="text" id="cs-input" placeholder="e.g. G8GDS" autocomplete="off" autocorrect="off" spellcheck="false">
    <button class="btn btn-primary" id="lookup-btn" type="submit">Look up</button>
  </form>
  <div id="lookup-result"></div>
</div>

<div id="view-pins" class="view">
  <div class="sites-header">
    <h2>Pins</h2>
  </div>
  <table>
    <thead><tr><th title="Click a name to open the map centred on that pin">Name</th><th>Lat</th><th>Lon</th><th title="Height above sea level (m)">QNH (m)</th><th></th></tr></thead>
    <tbody id="pins-tbody"></tbody>
  </table>
</div>

<div id="view-qths" class="view">
  <div class="sites-header">
    <h2>ROC Sites</h2>
    <button class="btn btn-success" onclick="openAddModal()">+ Add New</button>
  </div>
  <table>
    <thead><tr><th title="Click a callsign to open the map centred on that site">Callsign</th><th>Name</th><th>Lat</th><th>Lon</th><th title="Antenna height above ground (m)">QNF</th><th title="Height above sea level (m)">QNH (m)</th><th></th></tr></thead>
    <tbody id="qths-tbody"></tbody>
  </table>
</div>

<div id="view-map" class="view">
  <div id="map"></div>
</div>

<div class="modal-overlay" id="modal">
  <div class="modal-box">
    <h3 id="modal-title">Add Site</h3>
    <form id="site-form">
      <div class="form-group">
        <label>Callsign</label>
        <input type="text" id="f-callsign" required
          pattern="[A-Z0-9]{1,3}[0-9][A-Z]{1,5}"
          title="Valid amateur callsign e.g. G8GDS, 2E0FVC, VE3YW"
          oninput="this.value=this.value.toUpperCase().replace(/[^A-Z0-9\/]/g,'')"
          autocomplete="off" autocorrect="off" spellcheck="false">
        <button type="button" class="btn btn-primary btn-sm" style="margin-top:0.4rem" onclick="qrzFill()">Fill from QRZ</button>
      </div>
      <div class="form-group"><label>Name</label><input type="text" id="f-name"></div>
      <div class="form-group"><label>Latitude</label><input type="number" id="f-lat" step="any"></div>
      <div class="form-group"><label>Longitude</label><input type="number" id="f-lon" step="any"></div>
      <div class="form-group"><label title="Antenna height above ground (m)">QNF</label><input type="number" id="f-qnf" step="any" value="3"></div>
      <div class="form-group"><label title="Height above sea level (m)">QNH (m)</label><input type="number" id="f-qnh" step="any" placeholder="from QRZ"></div>
      <div class="modal-footer">
        <button type="button" class="btn btn-ghost" onclick="closeModal()">Cancel</button>
        <button type="submit" class="btn btn-primary" id="modal-save">Save</button>
      </div>
    </form>
  </div>
</div>

<div class="modal-overlay" id="pin-modal">
  <div class="modal-box">
    <h3>Name this pin</h3>
    <div class="form-group">
      <label>Name</label>
      <input type="text" id="pin-name" placeholder="e.g. Hill top" autocomplete="off">
    </div>
    <div id="pin-elev" style="font-size:0.82rem;color:#718096;margin-bottom:0.5rem;display:none"></div>
    <div class="modal-footer">
      <button type="button" class="btn btn-ghost" onclick="closePinModal()">Cancel</button>
      <button type="button" class="btn btn-success" id="pin-save-btn" onclick="savePinSite()">Save</button>
    </div>
  </div>
</div>

<script>
// ── Write token ──────────────────────────────────────────────
let writeToken = localStorage.getItem('roc-write-token') || '';

function updateTokenBtn() {
  const btn = document.getElementById('token-btn');
  btn.textContent = writeToken ? 'Unlocked' : 'Locked';
  btn.title = writeToken ? 'Write access on — click to change' : 'Click to enter write token';
  btn.style.opacity = writeToken ? '1' : '0.55';
}
updateTokenBtn();

function promptToken() {
  const t = prompt('Write token (blank to clear):', writeToken);
  if (t === null) return;
  writeToken = t.trim();
  if (writeToken) localStorage.setItem('roc-write-token', writeToken);
  else localStorage.removeItem('roc-write-token');
  updateTokenBtn();
}

async function authFetch(url, opts = {}) {
  const headers = { ...(opts.headers || {}) };
  if (writeToken) headers['X-Write-Token'] = writeToken;
  const r = await fetch(url, { ...opts, headers });
  if (r.status === 401) {
    const t = prompt('Write token required:');
    if (!t) return r;
    writeToken = t.trim();
    localStorage.setItem('roc-write-token', writeToken);
    updateTokenBtn();
    headers['X-Write-Token'] = writeToken;
    return fetch(url, { ...opts, headers });
  }
  return r;
}

// ── Navigation ───────────────────────────────────────────────
let leafletMap = null;

document.querySelectorAll('nav a').forEach(a => {
  a.addEventListener('click', e => {
    e.preventDefault();
    const view = a.dataset.view;
    document.querySelectorAll('nav a').forEach(x => x.classList.remove('active'));
    a.classList.add('active');
    document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
    document.getElementById('view-' + view).classList.add('active');
    if (view === 'pins') loadPinsTable();
    if (view === 'qths') loadQthsTable();
    if (view === 'map') initMap();
  });
});

async function updateNavCounts() {
  const sites = await fetch('/qrz/sites').then(r => r.json()).catch(() => []);
  const pins = (sites || []).filter(s => s.site_type === 'pin').length;
  const qths = (sites || []).filter(s => s.site_type !== 'pin').length;
  document.querySelector('[data-view="pins"]').textContent = 'Pins (' + pins + ')';
  document.querySelector('[data-view="qths"]').textContent = 'QTHs (' + qths + ')';
}
updateNavCounts();

// ── Lookup ───────────────────────────────────────────────────
document.getElementById('lookup-form').addEventListener('submit', async e => {
  e.preventDefault();
  const cs = document.getElementById('cs-input').value.trim().toUpperCase();
  if (!cs) return;
  const btn = document.getElementById('lookup-btn');
  const res = document.getElementById('lookup-result');
  btn.disabled = true;
  res.innerHTML = '<div class="spinner"></div>';
  try {
    const data = await fetch('/qrz/lookup/' + encodeURIComponent(cs)).then(r => r.json());
    if (data.error) {
      res.innerHTML = '<div class="error-box">' + esc(data.error) + '</div>';
    } else {
      const loc = [data.city, data.state, data.country].filter(Boolean).join(', ');
      const csEl = (data.lat && data.lon)
        ? '<a class="callsign-big" style="text-decoration:none;cursor:pointer;" href="#" data-cs="' + esc(data.callsign) + '" data-lat="' + data.lat + '" data-lon="' + data.lon + '" onclick="goToMapLoS(this.dataset.cs,+this.dataset.lat,+this.dataset.lon);return false;">' + esc(data.callsign) + '</a>'
        : '<div class="callsign-big">' + esc(data.callsign) + '</div>';
      res.innerHTML = '<div class="card">' +
        csEl +
        '<div class="name-big">' + esc(data.name || '—') + '</div>' +
        '<hr class="divider">' +
        '<div class="fields">' +
        (loc ? fld('Location', loc) : '') +
        (data.grid ? fld('Grid', data.grid) : '') +
        (data.lat && data.lon ? fld('Lat / Lon', data.lat + ', ' + data.lon) : '') +
        '</div>' +
        '</div>';
    }
  } catch { res.innerHTML = '<div class="error-box">Request failed. Please try again.</div>'; }
  btn.disabled = false;
});

// ── Sites tables ─────────────────────────────────────────────
function reloadActiveSiteView() {
  updateNavCounts();
  if (document.getElementById('view-pins').classList.contains('active')) loadPinsTable();
  else if (document.getElementById('view-qths').classList.contains('active')) loadQthsTable();
}

async function loadPinsTable() {
  const tbody = document.getElementById('pins-tbody');
  tbody.innerHTML = '<tr><td colspan="5"><div class="spinner"></div></td></tr>';
  const sites = await fetch('/qrz/sites').then(r => r.json());
  const pins = (sites || []).filter(s => s.site_type === 'pin').sort((a, b) => a.name.localeCompare(b.name));
  if (!pins.length) { tbody.innerHTML = '<tr><td colspan="5" style="color:#a0aec0;padding:1rem">No pins yet. Drop one from the Map view.</td></tr>'; return; }
  tbody.innerHTML = pins.map(s => '<tr>' +
    '<td><a class="map-link" style="font-size:inherit;font-weight:600" href="#" data-cs="' + esc(s.call_sign) + '" data-lat="' + s.lat + '" data-lon="' + s.lon + '" onclick="goToMapLoS(this.dataset.cs,+this.dataset.lat,+this.dataset.lon);return false;">' + esc(s.name) + '</a></td>' +
    '<td>' + s.lat.toFixed(4) + '</td>' +
    '<td>' + s.lon.toFixed(4) + '</td>' +
    '<td>' + (s.qnh != null ? Math.round(s.qnh) : '—') + '</td>' +
    '<td><div class="actions">' +
    '<button class="btn btn-primary btn-sm" onclick=\'openEditModal(' + JSON.stringify(s) + ')\'>Edit</button>' +
    '<button class="btn btn-danger btn-sm" onclick="deleteSite(\'' + esc(s.call_sign) + '\')">Delete</button>' +
    '</div></td></tr>'
  ).join('');
}

async function loadQthsTable() {
  const tbody = document.getElementById('qths-tbody');
  tbody.innerHTML = '<tr><td colspan="7"><div class="spinner"></div></td></tr>';
  const sites = await fetch('/qrz/sites').then(r => r.json());
  const qths = (sites || []).filter(s => s.site_type !== 'pin');
  if (!qths.length) { tbody.innerHTML = '<tr><td colspan="7" style="color:#a0aec0;padding:1rem">No sites yet.</td></tr>'; return; }
  tbody.innerHTML = qths.map(s => '<tr>' +
    '<td><span class="cs-link" title="View on map" onclick="goToMapLoS(\'' + esc(s.call_sign) + '\',' + s.lat + ',' + s.lon + ')">' + esc(s.call_sign) + '</span></td>' +
    '<td>' + esc(s.name) + '</td>' +
    '<td>' + s.lat.toFixed(4) + '</td>' +
    '<td>' + s.lon.toFixed(4) + '</td>' +
    '<td>' + (s.qnf != null ? s.qnf : 3) + '</td>' +
    '<td>' + (s.qnh != null ? Math.round(s.qnh) : '—') + '</td>' +
    '<td><div class="actions">' +
    '<button class="btn btn-primary btn-sm" onclick=\'openEditModal(' + JSON.stringify(s) + ')\'>Edit</button>' +
    '<button class="btn btn-danger btn-sm" onclick="deleteSite(\'' + esc(s.call_sign) + '\')">Delete</button>' +
    '</div></td></tr>'
  ).join('');
}

async function deleteSite(cs) {
  if (!confirm('Delete ' + cs + '?')) return;
  await authFetch('/qrz/sites/' + encodeURIComponent(cs), { method: 'DELETE' });
  reloadActiveSiteView();
}

function lookupAndSwitch(cs) {
  document.querySelectorAll('nav a').forEach(x => x.classList.remove('active'));
  document.querySelector('[data-view="lookup"]').classList.add('active');
  document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
  document.getElementById('view-lookup').classList.add('active');
  document.getElementById('cs-input').value = cs;
  document.getElementById('lookup-form').dispatchEvent(new Event('submit'));
}

// ── Modal ────────────────────────────────────────────────────
let editMode = false;
let editCallsign = '';

function openAddModal() {
  editMode = false; editCallsign = '';
  document.getElementById('modal-title').textContent = 'Add Site';
  document.getElementById('f-callsign').value = '';
  document.getElementById('f-callsign').disabled = false;
  document.getElementById('f-name').value = '';
  document.getElementById('f-lat').value = '';
  document.getElementById('f-lon').value = '';
  document.getElementById('f-qnf').value = '3';
  document.getElementById('f-qnh').value = '';
  document.getElementById('modal').classList.add('open');
}

function openEditModal(s) {
  editMode = true; editCallsign = s.call_sign;
  document.getElementById('modal-title').textContent = 'Edit Site';
  document.getElementById('f-callsign').value = s.call_sign;
  document.getElementById('f-callsign').disabled = true;
  document.getElementById('f-name').value = s.name;
  document.getElementById('f-lat').value = s.lat;
  document.getElementById('f-lon').value = s.lon;
  document.getElementById('f-qnf').value = s.qnf != null ? s.qnf : 3;
  document.getElementById('f-qnh').value = s.qnh != null ? Math.round(s.qnh) : '';
  document.getElementById('modal').classList.add('open');
}

function closeModal() { document.getElementById('modal').classList.remove('open'); }

async function qrzFill() {
  const cs = document.getElementById('f-callsign').value.trim().toUpperCase();
  if (!cs) return;
  const data = await fetch('/qrz/lookup/' + encodeURIComponent(cs)).then(r => r.json());
  if (data.error) { alert(data.error); return; }
  document.getElementById('f-name').value = data.name || '';
  document.getElementById('f-lat').value = data.lat || '';
  document.getElementById('f-lon').value = data.lon || '';
  document.getElementById('f-qnh').value = data.altm || '';
}

document.getElementById('site-form').addEventListener('submit', async e => {
  e.preventDefault();
  const qnhVal = document.getElementById('f-qnh').value;
  const body = {
    call_sign: document.getElementById('f-callsign').value.trim().toUpperCase(),
    name: document.getElementById('f-name').value.trim(),
    lat: parseFloat(document.getElementById('f-lat').value),
    lon: parseFloat(document.getElementById('f-lon').value),
    qnf: parseFloat(document.getElementById('f-qnf').value) || 3,
    qnh: qnhVal !== '' ? parseFloat(qnhVal) : null,
  };
  const url = editMode ? '/qrz/sites/' + encodeURIComponent(editCallsign) : '/qrz/sites';
  const method = editMode ? 'PUT' : 'POST';
  const resp = await authFetch(url, { method, headers: {'Content-Type':'application/json'}, body: JSON.stringify(body) });
  if (resp.ok) { closeModal(); reloadActiveSiteView(); }
  else { const d = await resp.json(); alert(d.error || 'Save failed'); }
});

// ── Map ──────────────────────────────────────────────────────
let clusterGroup = null;
let losLayer = null;
let allSitesData = [];
let pinDropMode = false;
let pinDropBtn = null;
let pendingPinLatLng = null;

const greenPinIcon = L.divIcon({
  className: '',
  html: '<div style="width:14px;height:14px;border-radius:50%;background:#22c55e;border:2.5px solid white;box-shadow:0 1px 5px rgba(0,0,0,0.5)"></div>',
  iconSize: [14, 14],
  iconAnchor: [7, 7],
  popupAnchor: [0, -8],
});

async function loadMapSites(fitBounds) {
  clearLoS();
  if (clusterGroup) leafletMap.removeLayer(clusterGroup);
  clusterGroup = L.markerClusterGroup();
  const sites = await fetch('/qrz/sites').then(r => r.json());
  if (!sites || !sites.length) return;
  allSitesData = sites;
  const bounds = [];
  sites.forEach(s => {
    if (!s.lat || !s.lon) return;
    const isPinSite = s.site_type === 'pin';
    const m = isPinSite ? L.marker([s.lat, s.lon], {icon: greenPinIcon}) : L.marker([s.lat, s.lon]);
    const hDesc = (s.qnh != null ? Math.round(s.qnh) + ' m ASL' : 'ASL unknown') +
                  ' + ' + (s.qnf != null ? s.qnf : 3) + ' m ant';
    if (isPinSite) {
      m.bindPopup('<strong>' + esc(s.name) + '</strong>' +
                  '<br><small style="color:#16a34a;font-weight:600">&#128205; pin</small>' +
                  '<br><small style="color:#718096">' + hDesc + '</small>');
    } else {
      m.bindPopup('<strong>' + esc(s.call_sign) + '</strong><br>' + esc(s.name) +
                  '<br><small style="color:#718096">' + hDesc + '</small>');
    }
    m.on('click', function() { showLoS(s); });
    clusterGroup.addLayer(m);
    bounds.push([s.lat, s.lon]);
  });
  leafletMap.addLayer(clusterGroup);
  if (fitBounds && bounds.length) leafletMap.fitBounds(bounds, { padding: [30, 30] });
}

function initMapBase() {
  if (!leafletMap) {
    leafletMap = L.map('map').setView([51.5, -2.0], 6);
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
    }).addTo(leafletMap);

    const LosCtrl = L.Control.extend({
      onAdd: function() {
        const d = L.DomUtil.create('div');
        d.style.cssText = 'background:white;padding:8px 12px;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,0.2);font-size:0.76rem;line-height:1.7;pointer-events:none;';
        d.innerHTML =
          '<strong style="font-size:0.82rem">Radio line of sight</strong><br>' +
          'Click a marker to show LoS<br>to sites within 100 km<br>' +
          '<span style="color:#22c55e;font-weight:700">&#9135;&#9135;</span> Clear &nbsp;' +
          '<span style="color:#ef4444;font-weight:700">&#xFE31;&#xFE31;</span> Blocked<br>' +
          '<span style="color:#999;font-size:0.71rem">Height = QNH + QNF<br>Terrain profile + refraction</span>';
        return d;
      }
    });
    new LosCtrl({position: 'bottomright'}).addTo(leafletMap);

    const PinCtrl = L.Control.extend({
      onAdd: function() {
        pinDropBtn = L.DomUtil.create('button', 'pin-drop-btn');
        pinDropBtn.innerHTML = '&#128205; Drop pin';
        L.DomEvent.on(pinDropBtn, 'click', function(e) {
          L.DomEvent.stopPropagation(e);
          pinDropMode = !pinDropMode;
          pinDropBtn.classList.toggle('active', pinDropMode);
          leafletMap.getContainer().style.cursor = pinDropMode ? 'crosshair' : '';
        });
        return pinDropBtn;
      }
    });
    new PinCtrl({position: 'topleft'}).addTo(leafletMap);

    leafletMap.on('click', function(e) {
      if (pinDropMode) openPinModal(e.latlng);
      else clearLoS();
    });
  }
}

async function initMap() {
  initMapBase();
  await loadMapSites(true);
}

async function goToMapLoS(cs, lat, lon) {
  document.querySelectorAll('nav a').forEach(x => x.classList.remove('active'));
  document.querySelector('[data-view="map"]').classList.add('active');
  document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
  document.getElementById('view-map').classList.add('active');
  initMapBase();
  await loadMapSites(false);
  leafletMap.setView([lat, lon], 11);
  const site = allSitesData.find(s => s.call_sign && s.call_sign.toLowerCase() === cs.toLowerCase());
  if (site) showLoS(site);
}

// ── Pin drop ──────────────────────────────────────────────────
function openPinModal(latlng) {
  pendingPinLatLng = latlng;
  document.getElementById('pin-name').value = '';
  document.getElementById('pin-elev').style.display = 'none';
  document.getElementById('pin-modal').classList.add('open');
  setTimeout(() => document.getElementById('pin-name').focus(), 50);
}

function closePinModal() {
  document.getElementById('pin-modal').classList.remove('open');
  pendingPinLatLng = null;
  pinDropMode = false;
  if (pinDropBtn) pinDropBtn.classList.remove('active');
  if (leafletMap) leafletMap.getContainer().style.cursor = '';
}

async function savePinSite() {
  const name = document.getElementById('pin-name').value.trim();
  if (!name) { document.getElementById('pin-name').focus(); return; }
  const btn = document.getElementById('pin-save-btn');
  btn.disabled = true;
  try {
    const resp = await authFetch('/qrz/sites/pin', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({ name, lat: pendingPinLatLng.lat, lon: pendingPinLatLng.lng }),
    });
    if (resp.ok) {
      closePinModal();
      loadMapSites(false);
      updateNavCounts();
    } else {
      const d = await resp.json();
      alert(d.error || 'Save failed');
    }
  } finally { btn.disabled = false; }
}

document.getElementById('pin-name').addEventListener('keydown', function(e) {
  if (e.key === 'Enter') savePinSite();
  if (e.key === 'Escape') closePinModal();
});

// ── Line of sight ─────────────────────────────────────────────
async function showLoS(clicked) {
  clearLoS();
  if (!losLayer) losLayer = L.layerGroup().addTo(leafletMap);
  try {
    const result = await fetch('/qrz/sites/los/' + encodeURIComponent(clicked.call_sign)).then(r => r.json());
    clearLoS();
    if (!losLayer) losLayer = L.layerGroup().addTo(leafletMap);
    Object.entries(result).forEach(([cs, entry]) => {
      const s = allSitesData.find(x => x.call_sign === cs);
      if (!s) return;
      if (entry.clear) {
        L.polyline([[clicked.lat, clicked.lon], [s.lat, s.lon]], {
          color: '#22c55e', weight: 2, opacity: 0.85,
        }).addTo(losLayer);
      } else {
        L.polyline([[clicked.lat, clicked.lon], [entry.obs_lat, entry.obs_lon]], {
          color: '#ef4444', weight: 2, opacity: 0.85,
        }).addTo(losLayer);
        L.polyline([[entry.obs_lat, entry.obs_lon], [s.lat, s.lon]], {
          color: '#555', weight: 2, opacity: 0.35,
        }).addTo(losLayer);
      }
    });
  } catch(e) { /* ignore */ }
}

function clearLoS() {
  if (losLayer) losLayer.clearLayers();
}

// ── Helpers ──────────────────────────────────────────────────
function fld(label, value) {
  return '<div class="field"><span class="field-label">' + esc(label) + '</span><span class="field-value">' + esc(value) + '</span></div>';
}
function esc(s) {
  return String(s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}
</script>
<footer style="margin-top:2rem;padding:0.5rem 1rem;text-align:center;font-size:0.7rem;color:#888;">A quanglewangle website &copy; 2026. Contains OS data &copy; Crown copyright and database right [2026] &mdash; build {{BUILD}}</footer>
</body>
</html>`

func okToWrite(w http.ResponseWriter, r *http.Request) bool {
	if writeToken == "" || r.Header.Get("X-Write-Token") == writeToken {
		return true
	}
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "unauthorised"})
	return false
}

func main() {
	username := os.Getenv("QRZ_USERNAME")
	password := os.Getenv("QRZ_PASSWORD")
	port := os.Getenv("QRZ_PORT")
	writeToken = os.Getenv("QRZ_WRITE_TOKEN")

	if username == "" || password == "" {
		log.Fatal("QRZ_USERNAME and QRZ_PASSWORD environment variables are required")
	}
	if port == "" {
		port = "8091"
	}

	t50Dir := os.Getenv("TERRAIN50_DIR")

	client := qrz.NewClient(username, password)
	db.OpenDatabase()

	http.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "max-age=86400")
		w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 5 3"><rect width="5" height="3" fill="#000"/><rect x="0" y="1" width="5" height="1" fill="#fff"/><rect x="2" y="0" width="1" height="3" fill="#fff"/></svg>`))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(strings.Replace(indexHTML, "{{BUILD}}", buildHash, 1)))
	})

	// POST /sites/refresh-qnh — populate QNH for all sites from terrain50 (no QRZ call)
	http.HandleFunc("/sites/refresh-qnh", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !okToWrite(w, r) {
			return
		}
		if t50Dir == "" {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "TERRAIN50_DIR not configured"})
			return
		}
		sites, err := db.GetAllQTH()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		updated, skipped := 0, 0
		for _, s := range sites {
			if s.Lat == 0 && s.Lon == 0 {
				skipped++
				continue
			}
			elev, err := terrain50.ElevationAt(t50Dir, float64(s.Lat), float64(s.Lon))
			if err != nil {
				skipped++
				continue
			}
			if db.SetQNHFromTerrain(s.CallSign, elev) == nil {
				updated++
			}
		}
		json.NewEncoder(w).Encode(map[string]int{"updated": updated, "skipped": skipped})
	})

	// GET /sites/los/{callsign} — terrain-profiled LoS to all sites within 100 km
	http.HandleFunc("/sites/los/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if t50Dir == "" {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "terrain data not configured"})
			return
		}
		cs := strings.TrimPrefix(r.URL.Path, "/sites/los/")
		if cs == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "callsign required"})
			return
		}
		sites, err := db.GetAllQTH()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		var src *db.QTH
		for i := range sites {
			if strings.EqualFold(sites[i].CallSign, cs) {
				src = &sites[i]
				break
			}
		}
		if src == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "site not found"})
			return
		}
		qnh1 := 0.0
		if src.QNH != nil {
			qnh1 = *src.QNH
		}
		h1 := qnh1 + src.QNF
		result := map[string]losEntry{}
		for _, s := range sites {
			if s.CallSign == src.CallSign || (s.Lat == 0 && s.Lon == 0) {
				continue
			}
			dist := terrain50.HaversineM(float64(src.Lat), float64(src.Lon), float64(s.Lat), float64(s.Lon))
			if dist > 100000 {
				continue
			}
			qnh2 := 0.0
			if s.QNH != nil {
				qnh2 = *s.QNH
			}
			h2 := qnh2 + s.QNF
			r := terrain50.LoSCheck(t50Dir,
				float64(src.Lat), float64(src.Lon), h1,
				float64(s.Lat), float64(s.Lon), h2)
			if r.Clear {
				result[s.CallSign] = losEntry{Clear: true}
			} else {
				result[s.CallSign] = losEntry{ObsLat: r.ObsLat, ObsLon: r.ObsLon}
			}
		}
		json.NewEncoder(w).Encode(result)
	})

	// POST /sites/pin — create a pin site from dropped map marker
	http.HandleFunc("/sites/pin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !okToWrite(w, r) {
			return
		}
		var body struct {
			Name string  `json:"name"`
			Lat  float32 `json:"lat"`
			Lon  float32 `json:"lon"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
			return
		}
		if body.Name == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "name required"})
			return
		}
		var qnh *float64
		if t50Dir != "" {
			if elev, err := terrain50.ElevationAt(t50Dir, float64(body.Lat), float64(body.Lon)); err == nil {
				qnh = &elev
			}
		}
		id, err := db.AddPinSite(body.Name, body.Lat, body.Lon, qnh)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created", "id": id})
	})

	// GET /sites — list all; POST /sites — create
	http.HandleFunc("/sites", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			sites, err := db.GetAllQTH()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			if sites == nil {
				sites = []db.QTH{}
			}
			json.NewEncoder(w).Encode(sites)

		case http.MethodPost:
			if !okToWrite(w, r) {
				return
			}
			var body struct {
				CallSign string   `json:"call_sign"`
				Name     string   `json:"name"`
				Lat      float32  `json:"lat"`
				Lon      float32  `json:"lon"`
				QNF      float64  `json:"qnf"`
				QNH      *float64 `json:"qnh"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
				return
			}
			if body.QNF == 0 {
				body.QNF = 3
			}
			if err := db.AddQTH(body.CallSign, body.Name, body.Lat, body.Lon, body.QNF, body.QNH); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"status": "created"})

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// PUT /sites/{callsign} — update; DELETE /sites/{callsign} — delete
	http.HandleFunc("/sites/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		cs := strings.TrimPrefix(r.URL.Path, "/sites/")
		cs = strings.TrimSpace(cs)
		if cs == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "callsign required"})
			return
		}

		switch r.Method {
		case http.MethodPut:
			if !okToWrite(w, r) {
				return
			}
			var body struct {
				Name string   `json:"name"`
				Lat  float32  `json:"lat"`
				Lon  float32  `json:"lon"`
				QNF  float64  `json:"qnf"`
				QNH  *float64 `json:"qnh"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
				return
			}
			if body.QNF == 0 {
				body.QNF = 3
			}
			if err := db.UpdateQTH(cs, body.Name, body.Lat, body.Lon, body.QNF, body.QNH); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

		case http.MethodDelete:
			if !okToWrite(w, r) {
				return
			}
			if err := db.DeleteQTH(cs); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/lookup/", func(w http.ResponseWriter, r *http.Request) {
		callsign := strings.TrimPrefix(r.URL.Path, "/lookup/")
		callsign = strings.TrimSpace(callsign)
		if callsign == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "callsign required"})
			return
		}

		result, err := client.Lookup(callsign)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		if result.Lat != "" && result.Lon != "" {
			if result.AltM == "" && t50Dir != "" {
				lat, err1 := strconv.ParseFloat(result.Lat, 64)
				lon, err2 := strconv.ParseFloat(result.Lon, 64)
				if err1 == nil && err2 == nil {
					if elev, err := terrain50.ElevationAt(t50Dir, lat, lon); err == nil {
						result.AltM = strconv.FormatFloat(elev, 'f', 1, 64)
					}
				}
			}
			db.UpsertQTH(result.Callsign, result.Name, result.Lat, result.Lon, result.AltM)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// GET /qrz/los?lat1=&lon1=&lat2=&lon2=&h1=&h2=
	// Terrain LoS between two WGS84 points using OS Terrain 50.
	// h1/h2 = antenna height above ground in metres (default 10).
	// Returns 503 if TERRAIN50_DIR not set, 422 if either point outside GB.
	http.HandleFunc("/qrz/los", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if t50Dir == "" {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"error": "terrain data not configured"})
			return
		}
		q := r.URL.Query()
		lat1, e1 := strconv.ParseFloat(q.Get("lat1"), 64)
		lon1, e2 := strconv.ParseFloat(q.Get("lon1"), 64)
		lat2, e3 := strconv.ParseFloat(q.Get("lat2"), 64)
		lon2, e4 := strconv.ParseFloat(q.Get("lon2"), 64)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "lat1, lon1, lat2, lon2 required"})
			return
		}
		h1, _ := strconv.ParseFloat(q.Get("h1"), 64)
		h2, _ := strconv.ParseFloat(q.Get("h2"), 64)
		if h1 <= 0 {
			h1 = 10
		}
		if h2 <= 0 {
			h2 = 10
		}
		elev1, err := terrain50.ElevationAt(t50Dir, lat1, lon1)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"error": "my position outside GB grid"})
			return
		}
		elev2, err := terrain50.ElevationAt(t50Dir, lat2, lon2)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"error": "target outside GB grid"})
			return
		}
		distKm := terrain50.HaversineM(lat1, lon1, lat2, lon2) / 1000.0
		result := terrain50.LoSCheck(t50Dir, lat1, lon1, elev1+h1, lat2, lon2, elev2+h2)
		type losResp struct {
			Clear      bool    `json:"clear"`
			MyElev     float64 `json:"my_elev"`
			TargetElev float64 `json:"target_elev"`
			DistanceKm float64 `json:"distance_km"`
			ObsLat     float64 `json:"obs_lat,omitempty"`
			ObsLon     float64 `json:"obs_lon,omitempty"`
		}
		json.NewEncoder(w).Encode(losResp{
			Clear:      result.Clear,
			MyElev:     elev1,
			TargetElev: elev2,
			DistanceKm: distKm,
			ObsLat:     result.ObsLat,
			ObsLon:     result.ObsLon,
		})
	})

	log.Printf("qrzlook listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

